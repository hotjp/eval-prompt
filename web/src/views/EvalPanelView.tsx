import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Row, Col, Statistic, Button, Space, Spin, message, Table, Tag } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import * as echarts from 'echarts'
import { assetApi, evalApi } from '../api/client'
import type { AssetDetail, EvalRun, EvalReport } from '../api/client'
import { useStore } from '../store'

function EvalPanelView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [runs, setRuns] = useState<EvalRun[]>([])
  const [currentRun, setCurrentRun] = useState<EvalReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [running, setRunning] = useState(false)

  const chartRef = useRef<HTMLDivElement>(null)
  const chartInstance = useRef<echarts.ECharts | null>(null)

  useEffect(() => {
    if (id) {
      loadData(id)
    }
  }, [id])

  useEffect(() => {
    if (chartRef.current && !chartInstance.current) {
      chartInstance.current = echarts.init(chartRef.current)
    }
    return () => {
      chartInstance.current?.dispose()
    }
  }, [])

  useEffect(() => {
    if (chartInstance.current && runs.length > 0) {
      const sortedRuns = [...runs].sort(
        (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      )
      const option = {
        title: { text: 'Eval Score Trend', left: 'center' },
        xAxis: {
          type: 'category',
          data: sortedRuns.map((r) => new Date(r.created_at).toLocaleDateString()),
        },
        yAxis: { type: 'value', min: 0, max: 1 },
        series: [
          {
            name: 'Deterministic',
            type: 'line',
            data: sortedRuns.map((r) => r.deterministic_score),
            smooth: true,
          },
          {
            name: 'Rubric',
            type: 'line',
            data: sortedRuns.map((r) => r.rubric_score),
            smooth: true,
          },
        ],
        legend: { bottom: 0 },
        tooltip: { trigger: 'axis' },
      }
      chartInstance.current.setOption(option)
    }
  }, [runs])

  const loadData = async (assetId: string) => {
    setLoading(true)
    try {
      const [assetData, runsData] = await Promise.all([
        assetApi.get(assetId),
        evalApi.list(assetId),
      ])
      setAsset(assetData)
      setRuns(runsData)
      if (runsData.length > 0) {
        const latestReport = await evalApi.report(runsData[0].id)
        setCurrentRun(latestReport)
      }
    } catch {
      message.error('Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  const handleRunEval = async () => {
    if (!id) return
    setRunning(true)
    useStore.getState().setRunningEval({ id: '', assetId: id, assetName: asset?.name || id })
    try {
      const result = await evalApi.run(id)
      useStore.getState().setRunningEval({ id: result.run_id, assetId: id, assetName: asset?.name || id })
      message.info('Eval started, run ID: ' + result.run_id)
      const poll = async () => {
        const run = await evalApi.get(result.run_id)
        if (run.status === 'running' || run.status === 'pending') {
          setTimeout(poll, 2000)
        } else {
          const report = await evalApi.report(result.run_id)
          setCurrentRun(report)
          setRuns((prev) => [run, ...prev])
          setRunning(false)
          useStore.getState().setRunningEval(null)
        }
      }
      poll()
    } catch {
      message.error('Failed to start eval')
      setRunning(false)
      useStore.getState().setRunningEval(null)
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!asset) {
    return <div>Asset not found</div>
  }

  const passRate = currentRun && currentRun.rubric_details.length > 0
    ? (currentRun.rubric_details.filter((r) => r.passed).length / currentRun.rubric_details.length) * 100
    : 0

  return (
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card title={asset.name} extra={<Tag>{asset.state}</Tag>}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic
                title="Deterministic Score"
                value={currentRun?.deterministic_score ?? 0}
                precision={2}
                suffix="/ 1.0"
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="Rubric Score"
                value={currentRun?.rubric_score ?? 0}
                precision={2}
                suffix="/ 1.0"
              />
            </Col>
            <Col span={6}>
              <Statistic title="Overall Score" value={currentRun?.overall_score ?? 0} precision={2} suffix="/ 1.0" />
            </Col>
            <Col span={6}>
              <Statistic title="Pass Rate" value={passRate} precision={1} suffix="%" />
            </Col>
          </Row>
        </Card>

        <Card
          title="Score Trend"
          extra={<Button icon={<ReloadOutlined />} onClick={() => id && loadData(id)}>Refresh</Button>}
        >
          <div ref={chartRef} style={{ height: 300 }} />
        </Card>

        <Card
          title="Rubric Details"
          extra={<Button type="primary" onClick={handleRunEval} loading={running}>Run Eval</Button>}
        >
          {currentRun ? (
            <Table
              dataSource={currentRun.rubric_details}
              rowKey="check_id"
              size="small"
              pagination={false}
              columns={[
                { title: 'Check ID', dataIndex: 'check_id', key: 'check_id' },
                {
                  title: 'Status',
                  dataIndex: 'passed',
                  key: 'passed',
                  render: (passed: boolean) => (
                    <Tag icon={passed ? <CheckCircleOutlined /> : <CloseCircleOutlined />} color={passed ? 'green' : 'red'}>
                      {passed ? 'Passed' : 'Failed'}
                    </Tag>
                  ),
                },
                { title: 'Score', dataIndex: 'score', key: 'score', render: (s: number) => s.toFixed(2) },
                { title: 'Details', dataIndex: 'details', key: 'details', ellipsis: true },
              ]}
            />
          ) : (
            <div style={{ textAlign: 'center', color: '#888' }}>No eval results yet</div>
          )}
        </Card>

        <Card title="Recent Runs">
          <Table
            dataSource={runs.slice(0, 10)}
            rowKey="id"
            size="small"
            pagination={false}
            columns={[
              { title: 'Run ID', dataIndex: 'id', key: 'id', render: (id) => id.slice(-8) },
              {
                title: 'Status',
                dataIndex: 'status',
                key: 'status',
                render: (status) => {
                  const color = status === 'passed' ? 'green' : status === 'failed' ? 'red' : 'blue'
                  return <Tag color={color}>{status}</Tag>
                },
              },
              { title: 'Det. Score', dataIndex: 'deterministic_score', key: 'deterministic_score', render: (s) => s?.toFixed(2) ?? 'N/A' },
              { title: 'Rubric Score', dataIndex: 'rubric_score', key: 'rubric_score', render: (s) => s?.toFixed(2) ?? 'N/A' },
              { title: 'Created', dataIndex: 'created_at', key: 'created_at', render: (d) => d ? new Date(d).toLocaleString() : 'N/A' },
              {
                title: 'Action',
                key: 'action',
                render: () => (
                  <Button size="small" onClick={() => navigate(`/assets/${id}/versions`)}>
                    Details
                  </Button>
                ),
              },
            ]}
          />
        </Card>
      </Space>
    </div>
  )
}

export default EvalPanelView
