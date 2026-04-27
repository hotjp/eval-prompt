import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Row, Col, Statistic, Button, Space, Spin, message, Table, Tag, Select, Progress, InputNumber } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, ReloadOutlined, PlayCircleOutlined, StopOutlined } from '@ant-design/icons'
import * as echarts from 'echarts'
import { assetApi, evalApi } from '../api/client'
import type { AssetDetail, EvalRun, EvalReport, Execution } from '../api/client'
import { useStore } from '../store'
import { useTranslation } from 'react-i18next'

function EvalPanelView() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [runs, setRuns] = useState<EvalRun[]>([])
  const [currentRun, setCurrentRun] = useState<EvalReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [running, setRunning] = useState(false)
  const [execution, setExecution] = useState<Execution | null>(null)
  const [evalMode, setEvalMode] = useState<string>('single')
  const [evalModel, setEvalModel] = useState<string>('')
  const [evalTemperature, setEvalTemperature] = useState<number>(0.7)
  const evalConcurrency = useStore((state) => state.evalConcurrency)

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
        assetApi.get(assetId).catch(() => null),
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
    setExecution(null)
    useStore.getState().setRunningEval({ id: '', assetId: id, assetName: asset?.name || id })
    try {
      const result = await evalApi.execute({
        asset_id: id,
        mode: evalMode,
        concurrency: evalConcurrency,
        model: evalModel || undefined,
        temperature: evalTemperature,
      })
      useStore.getState().setRunningEval({ id: result.execution_id, assetId: id, assetName: asset?.name || id })
      message.info('Eval started, execution ID: ' + result.execution_id)

      const poll = async () => {
        const exec = await evalApi.getExecution(result.execution_id)
        setExecution(exec)

        if (exec.status === 'running' || exec.status === 'pending' || exec.status === 'initializing') {
          setTimeout(poll, 2000)
        } else {
          setRunning(false)
          useStore.getState().setRunningEval(null)
          message.success('Eval completed with status: ' + exec.status)
        }
      }
      poll()
    } catch {
      message.error('Failed to start eval')
      setRunning(false)
      useStore.getState().setRunningEval(null)
    }
  }

  const handleCancelExecution = async () => {
    if (!execution?.id) return
    try {
      await evalApi.cancelExecution(execution.id)
      message.info('Cancellation requested')
    } catch {
      message.error('Failed to cancel execution')
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!asset) {
    return <div>{t('eval_panel_asset_not_found')}</div>
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
          title={t('eval_panel_score_trend')}
          extra={<Button icon={<ReloadOutlined />} onClick={() => id && loadData(id)}>{t('eval_panel_refresh')}</Button>}
        >
          <div ref={chartRef} style={{ height: 300 }} />
        </Card>

        <Card
          title={t('eval_panel_run_eval')}
          extra={
            <Space>
              <Select
                value={evalMode}
                onChange={setEvalMode}
                style={{ width: 120 }}
                disabled={running}
                options={[
                  { value: 'single', label: t('eval_panel_mode_single') },
                  { value: 'batch', label: t('eval_panel_mode_batch') },
                  { value: 'matrix', label: t('eval_panel_mode_matrix') },
                ]}
              />
              <Select
                value={evalModel}
                onChange={setEvalModel}
                style={{ width: 160 }}
                disabled={running}
                placeholder={t('eval_panel_select_model')}
                allowClear
                options={[
                  { value: 'gpt-4o', label: 'GPT-4o' },
                  { value: 'gpt-4o-mini', label: 'GPT-4o Mini' },
                  { value: 'claude-3-5-sonnet', label: 'Claude 3.5 Sonnet' },
                  { value: 'claude-3-5-haiku', label: 'Claude 3.5 Haiku' },
                ]}
              />
              <span>{t('eval_panel_temp')}:</span>
              <InputNumber
                value={evalTemperature}
                onChange={(v) => setEvalTemperature(v ?? 0.7)}
                min={0}
                max={2}
                step={0.1}
                disabled={running}
                style={{ width: 80 }}
              />
              <span>{t('eval_panel_concurrency')}: {evalConcurrency}</span>
              <Button
                type="primary"
                icon={<PlayCircleOutlined />}
                onClick={handleRunEval}
                loading={running}
              >
                {t('eval_panel_run_eval_button')}
              </Button>
              {running && execution && (
                <Button
                  danger
                  icon={<StopOutlined />}
                  onClick={handleCancelExecution}
                >
                  {t('eval_panel_cancel')}
                </Button>
              )}
            </Space>
          }
        >
          {running && execution && (
            <div style={{ marginBottom: 16 }}>
              <Progress
                percent={execution.total_cases > 0
                  ? Math.round((execution.completed_cases / execution.total_cases) * 100)
                  : 0}
                status="active"
                format={() =>
                  `${execution.completed_cases} / ${execution.total_cases} ${t('eval_panel_cases')}`
                }
              />
              <div style={{ marginTop: 8, color: '#888' }}>
                {t('eval_panel_status')}: {execution.status} | {t('eval_panel_concurrency')}: {execution.concurrency} | {t('eval_panel_model')}: {execution.model}
              </div>
            </div>
          )}
        </Card>

        <Card title={t('eval_panel_rubric_details')}>
          {currentRun ? (
            <Table
              dataSource={currentRun.rubric_details}
              rowKey="check_id"
              size="small"
              pagination={false}
              columns={[
                { title: t('eval_panel_col_check_id'), dataIndex: 'check_id', key: 'check_id' },
                {
                  title: t('eval_panel_col_status'),
                  dataIndex: 'passed',
                  key: 'passed',
                  render: (passed: boolean) => (
                    <Tag icon={passed ? <CheckCircleOutlined /> : <CloseCircleOutlined />} color={passed ? 'green' : 'red'}>
                      {passed ? t('eval_panel_passed') : t('eval_panel_failed')}
                    </Tag>
                  ),
                },
                { title: t('eval_panel_col_score'), dataIndex: 'score', key: 'score', render: (s: number) => s.toFixed(2) },
                { title: t('eval_panel_col_details'), dataIndex: 'details', key: 'details', ellipsis: true },
              ]}
            />
          ) : (
            <div style={{ textAlign: 'center', color: '#888' }}>{t('eval_panel_no_results')}</div>
          )}
        </Card>

        <Card title={t('eval_panel_recent_runs')}>
          <Table
            dataSource={runs.slice(0, 10)}
            rowKey="id"
            size="small"
            pagination={false}
            columns={[
              { title: t('eval_panel_col_run_id'), dataIndex: 'id', key: 'id', render: (id) => id.slice(-8) },
              {
                title: t('eval_panel_col_status'),
                dataIndex: 'status',
                key: 'status',
                render: (status) => {
                  const color = status === 'passed' ? 'green' : status === 'failed' ? 'red' : 'blue'
                  return <Tag color={color}>{status}</Tag>
                },
              },
              { title: t('eval_panel_col_det_score'), dataIndex: 'deterministic_score', key: 'deterministic_score', render: (s) => s?.toFixed(2) ?? 'N/A' },
              { title: t('eval_panel_col_rubric_score'), dataIndex: 'rubric_score', key: 'rubric_score', render: (s) => s?.toFixed(2) ?? 'N/A' },
              { title: t('eval_panel_col_created'), dataIndex: 'created_at', key: 'created_at', render: (d) => d ? new Date(d).toLocaleString() : 'N/A' },
              {
                title: t('eval_panel_col_action'),
                key: 'action',
                render: () => (
                  <Button size="small" onClick={() => navigate(`/assets/${id}/versions`)}>
                    {t('eval_panel_details')}
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
