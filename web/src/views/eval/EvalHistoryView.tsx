import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Table, Tag, Button, Space, Spin, message, Row, Col, Statistic } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import * as echarts from 'echarts'
import { useTranslation } from 'react-i18next'
import { evalApi } from '../../api/client'
import type { EvalRun, EvalReport } from '../../api/client'

function EvalHistoryView() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [runs, setRuns] = useState<EvalRun[]>([])
  const [currentRun, setCurrentRun] = useState<EvalReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedRuns, setSelectedRuns] = useState<Set<string>>(new Set())

  const chartRef = useRef<HTMLDivElement>(null)
  const chartInstance = useRef<echarts.ECharts | null>(null)

  useEffect(() => {
    if (id) loadData(id)
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
      const runsData = await evalApi.list(assetId)
      setRuns(runsData)
      if (runsData.length > 0) {
        const latestReport = await evalApi.report(runsData[0].id)
        setCurrentRun(latestReport)
      }
    } catch {
      message.error('Failed to load eval history')
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  const passRate = currentRun && currentRun.rubric_details.length > 0
    ? (currentRun.rubric_details.filter((r) => r.passed).length / currentRun.rubric_details.length) * 100
    : 0

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* Score Cards */}
      <Card>
        <Row gutter={16}>
          <Col span={6}>
            <Statistic title="Deterministic Score" value={currentRun?.deterministic_score ?? 0} precision={2} suffix="/ 1.0" />
          </Col>
          <Col span={6}>
            <Statistic title="Rubric Score" value={currentRun?.rubric_score ?? 0} precision={2} suffix="/ 1.0" />
          </Col>
          <Col span={6}>
            <Statistic title="Overall Score" value={currentRun?.overall_score ?? 0} precision={2} suffix="/ 1.0" />
          </Col>
          <Col span={6}>
            <Statistic title="Pass Rate" value={passRate} precision={1} suffix="%" />
          </Col>
        </Row>
      </Card>

      {/* Trend Chart */}
      <Card
        title={t('eval_panel_score_trend')}
        extra={<Button icon={<ReloadOutlined />} onClick={() => id && loadData(id)}>{t('eval_panel_refresh')}</Button>}
      >
        <div ref={chartRef} style={{ height: 300 }} />
      </Card>

      {/* Runs Table with selection */}
      <Card title="Recent Runs">
        {selectedRuns.size === 2 && (
          <div style={{ marginBottom: 12 }}>
            <Button type="primary" onClick={() => {
              const [a, b] = Array.from(selectedRuns)
              navigate(`/compare?asset=${id}&v1=${a}&v2=${b}`)
            }}>
              Compare Selected Runs
            </Button>
          </div>
        )}
        <Table
          dataSource={runs.slice(0, 20)}
          rowKey="id"
          size="small"
          pagination={false}
          rowSelection={{
            type: 'checkbox',
            selectedRowKeys: Array.from(selectedRuns),
            onChange: (keys) => setSelectedRuns(new Set(keys as string[])),
            getCheckboxProps: () => ({
              disabled: false,
            }),
          }}
          columns={[
            { title: t('eval_panel_col_run_id'), dataIndex: 'id', key: 'id', render: (id: string) => id.slice(-8) },
            {
              title: t('eval_panel_col_status'),
              dataIndex: 'status',
              key: 'status',
              render: (status: string) => {
                const color = status === 'passed' ? 'green' : status === 'failed' ? 'red' : 'blue'
                return <Tag color={color}>{status}</Tag>
              },
            },
            { title: t('eval_panel_col_det_score'), dataIndex: 'deterministic_score', key: 'deterministic_score', render: (s: number) => s?.toFixed(2) ?? 'N/A' },
            { title: t('eval_panel_col_rubric_score'), dataIndex: 'rubric_score', key: 'rubric_score', render: (s: number) => s?.toFixed(2) ?? 'N/A' },
            { title: t('eval_panel_col_created'), dataIndex: 'created_at', key: 'created_at', render: (d: string) => d ? new Date(d).toLocaleString() : 'N/A' },
            {
              title: t('eval_panel_col_action'),
              key: 'action',
              render: (_: unknown, record: EvalRun) => (
                <Button size="small" onClick={() => navigate(`/assets/${id}/eval/report/${record.id}`)}>
                  {t('eval_panel_details')}
                </Button>
              ),
            },
          ]}
        />
      </Card>
    </Space>
  )
}

export default EvalHistoryView
