import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Tag, Button, Space, Spin, Row, Col, Statistic, Table, Collapse, Empty, Tooltip } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, EyeOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { evalApi, executionApi } from '../../api/client'
import type { EvalReport, LLMCall, OrchestrateResponse, PluginResult } from '../../api/client'

function EvalReportView() {
  const { t } = useTranslation()
  const { id, runId } = useParams<{ id: string; runId: string }>()
  const navigate = useNavigate()
  const [report, setReport] = useState<EvalReport | null>(null)
  const [orchestrateResult, setOrchestrateResult] = useState<OrchestrateResponse | null>(null)
  const [calls, setCalls] = useState<LLMCall[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (id && runId) loadData(id, runId)
  }, [id, runId])

  const loadData = async (_assetId: string, rid: string) => {
    setLoading(true)
    setError(null)
    try {
      const [reportData, callsData] = await Promise.all([
        evalApi.report(rid).catch((e) => { setError(e?.response?.data?.error || 'Failed to load report'); return null }),
        executionApi.getCalls(rid).catch(() => []),
      ])
      setReport(reportData)
      // If report has plugin_results, it's an orchestrate-style response
      if (reportData && 'plugin_results' in reportData) {
        setOrchestrateResult(reportData as unknown as OrchestrateResponse)
      }
      setCalls(callsData)
    } catch (e: any) {
      setError(e?.message || 'Failed to load report')
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (error) {
    return (
      <Space direction="vertical" size="large" style={{ width: '100%', marginTop: 40 }}>
        <Card>
          <Empty description={error}>
            <Button type="primary" onClick={() => id && runId && loadData(id, runId)}>Retry</Button>
          </Empty>
        </Card>
      </Space>
    )
  }

  if (!report && !orchestrateResult) {
    return (
      <Space direction="vertical" size="large" style={{ width: '100%', marginTop: 40 }}>
        <Card>
          <Empty description={t('eval_orchestrate_report_not_found') || 'Report not found'}>
            <Button onClick={() => navigate(`/assets/${id}/eval/history`)}>Back to History</Button>
          </Empty>
        </Card>
      </Space>
    )
  }

  const passRate = report && report.rubric_details.length > 0
    ? (report.rubric_details.filter((r) => r.passed).length / report.rubric_details.length) * 100
    : 0

  // Render orchestrate-style report
  if (orchestrateResult) {
    return (
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* Header */}
        <Card
          size="small"
          title={`${t('eval_orchestrate_result')}: ${runId?.slice(-8)}`}
          extra={
            <Tag color="purple">{t('eval_orchestrate_multi_plugin')}</Tag>
          }
        >
          <Button size="small" onClick={() => navigate(`/assets/${id}/eval/history`)}>
            {t('eval_orchestrate_back_to_history')}
          </Button>
        </Card>

        {/* Overall Score Cards */}
        <Card>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic title={t('eval_orchestrate_overall_score')} value={orchestrateResult.overall_score} precision={2} suffix="/ 1.0" />
            </Col>
            {orchestrateResult.confidence_interval && (
              <Col span={6}>
                <Statistic
                  title={t('eval_orchestrate_confidence_interval')}
                  value={`[${orchestrateResult.confidence_interval.low.toFixed(2)}, ${orchestrateResult.confidence_interval.high.toFixed(2)}]`}
                />
              </Col>
            )}
            {orchestrateResult.baseline_comparison && (
              <>
                <Col span={6}>
                  <Statistic
                    title={t('eval_orchestrate_score_delta')}
                    value={orchestrateResult.baseline_comparison.score_delta}
                    precision={2}
                    valueStyle={{ color: orchestrateResult.baseline_comparison.score_delta >= 0 ? '#52c41a' : '#ff4d4f' }}
                  />
                </Col>
                <Col span={6}>
                  <Statistic
                    title={t('eval_orchestrate_effect_size')}
                    value={orchestrateResult.baseline_comparison.effect_size}
                    precision={3}
                  />
                </Col>
              </>
            )}
            {orchestrateResult.elo_result && (
              <Col span={6}>
                <Statistic title={t('eval_orchestrate_elo_rating')} value={orchestrateResult.elo_result.new_rating} suffix={orchestrateResult.elo_result.previous_rating ? `(${orchestrateResult.elo_result.previous_rating})` : ''} />
              </Col>
            )}
          </Row>
        </Card>

        {/* Plugin Results */}
        {Object.keys(orchestrateResult.plugin_results).length > 0 && (
          <Card title={t('eval_orchestrate_plugin_results')}>
            <Row gutter={16}>
              {Object.entries(orchestrateResult.plugin_results).map(([pluginName, result]: [string, PluginResult]) => (
                <Col span={8} key={pluginName}>
                  <Card size="small" title={pluginName}>
                    <Statistic
                      title="Score"
                      value={result.score}
                      precision={2}
                      suffix="/ 1.0"
                    />
                    {result.confidence_interval && (
                      <div style={{ fontSize: 12, color: '#888', marginTop: 8 }}>
                        95% CI: [{result.confidence_interval.low.toFixed(3)}, {result.confidence_interval.high.toFixed(3)}]
                      </div>
                    )}
                    <div style={{ fontSize: 12, color: '#888', marginTop: 8 }}>
                      {result.work_item_results.length} work items evaluated
                    </div>
                  </Card>
                </Col>
              ))}
            </Row>
          </Card>
        )}

        {/* Baseline Comparison Details */}
        {orchestrateResult.baseline_comparison && (
          <Card title={t('eval_orchestrate_baseline_comparison')}>
            <Row gutter={16}>
              <Col span={6}>
                <Statistic title={t('eval_orchestrate_score_delta')} value={orchestrateResult.baseline_comparison.score_delta} precision={3} />
              </Col>
              <Col span={6}>
                <Statistic title={t('eval_orchestrate_effect_size')} value={orchestrateResult.baseline_comparison.effect_size} precision={3} />
              </Col>
              <Col span={6}>
                <Statistic title="T-Statistic" value={orchestrateResult.baseline_comparison.t_stat} precision={3} />
              </Col>
              <Col span={6}>
                <Statistic title={t('eval_orchestrate_p_value')} value={orchestrateResult.baseline_comparison.p_value} precision={4} />
              </Col>
            </Row>
            <div style={{ marginTop: 16 }}>
              <Tag color={orchestrateResult.baseline_comparison.is_significant ? 'green' : 'orange'} style={{ fontSize: 14, padding: '4px 12px' }}>
                {t(orchestrateResult.baseline_comparison.is_significant ? 'eval_orchestrate_significant' : 'eval_orchestrate_not_significant')}
              </Tag>
              <span style={{ marginLeft: 12, color: '#888' }}>
                Interpretation: {orchestrateResult.baseline_comparison.effect_interpretation}
              </span>
            </div>
          </Card>
        )}

        {/* ELO Result */}
        {orchestrateResult.elo_result && (
          <Card title="ELO Rating Change">
            <Row gutter={16}>
              <Col span={8}>
                <Statistic title="Previous Rating" value={orchestrateResult.elo_result.previous_rating} />
              </Col>
              <Col span={8}>
                <Statistic title="New Rating" value={orchestrateResult.elo_result.new_rating} />
              </Col>
              <Col span={8}>
                <Statistic
                  title="Outcome"
                  value={orchestrateResult.elo_result.outcome === 1 ? 'Win' : orchestrateResult.elo_result.outcome === 0.5 ? 'Draw' : 'Loss'}
                  valueStyle={{ color: orchestrateResult.elo_result.outcome === 1 ? '#52c41a' : orchestrateResult.elo_result.outcome === 0 ? '#ff4d4f' : '#faad14' }}
                />
              </Col>
            </Row>
          </Card>
        )}

        {/* Summary */}
        {orchestrateResult.summary && (
          <Card title="Summary">
            <div style={{ padding: 12, background: '#f5f5f5', borderRadius: 6, fontSize: 14 }}>
              {orchestrateResult.summary}
            </div>
          </Card>
        )}
      </Space>
    )
  }

  // Render standard EvalReport
  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* Header */}
      <Card
        size="small"
        title={`Run: ${runId?.slice(-8)}`}
        extra={
          <Tag color={report!.status === 'passed' ? 'green' : report!.status === 'failed' ? 'red' : 'blue'}>
            {report!.status}
          </Tag>
        }
      >
        <Button size="small" onClick={() => navigate(`/assets/${id}/eval/history`)}>
          ← Back to History
        </Button>
      </Card>

      {/* Score Cards */}
      <Card>
        <Row gutter={16}>
          <Col span={6}>
            <Statistic title="Overall Score" value={report?.overall_score ?? 0} precision={0} suffix="/ 100" />
          </Col>
          <Col span={6}>
            <Statistic title="Deterministic Score" value={(report?.deterministic_score ?? 0) * 100} precision={0} suffix="/ 100" />
          </Col>
          <Col span={6}>
            <Statistic title="Rubric Score" value={report?.rubric_score ?? 0} precision={0} suffix="/ 100" />
          </Col>
          <Col span={6}>
            <Statistic title="Pass Rate" value={passRate} precision={1} suffix="%" />
          </Col>
        </Row>
      </Card>

      {/* Rubric Diagnosis */}
      <Card title="Rubric Diagnosis">
        <Collapse
          items={(report?.rubric_details || []).map((detail) => ({
            key: detail.check_id,
            label: (
              <Space>
                <Tag
                  icon={detail.passed ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
                  color={detail.passed ? 'green' : 'red'}
                >
                  {detail.passed ? 'PASS' : 'FAIL'}
                </Tag>
                <span>{detail.check_id}</span>
                <span style={{ color: '#888' }}>Score: {detail.score.toFixed(2)}</span>
              </Space>
            ),
            children: (
              <Space direction="vertical" size="small" style={{ width: '100%' }}>
                <div>
                  <strong style={{ color: '#888' }}>Details:</strong>
                  <div style={{ padding: 8, background: '#f5f5f5', borderRadius: 4, marginTop: 4 }}>
                    {detail.details}
                  </div>
                </div>
                {calls.length > 0 && (
                  <div>
                    <strong style={{ color: '#888' }}>Related Calls:</strong>
                    <div style={{ marginTop: 4 }}>
                      {calls.slice(0, 3).map((call) => (
                        <Tooltip title={call.id} key={call.id}>
                          <Tag
                            icon={<EyeOutlined />}
                            style={{ cursor: 'pointer' }}
                            onClick={() => navigate(`/executions/${runId}/calls`)}
                          >
                            {call.run_id?.slice(-8) || 'call'}
                          </Tag>
                        </Tooltip>
                      ))}
                    </div>
                  </div>
                )}
              </Space>
            ),
          }))}
        />
      </Card>

      {/* LLM Calls */}
      <Card title="LLM Calls">
        <Table
          dataSource={calls}
          rowKey="id"
          size="small"
          pagination={{ pageSize: 10 }}
          columns={[
            { title: 'ID', dataIndex: 'id', key: 'id', render: (callId: string) => <Tooltip title={callId}><span>{callId.slice(-8)}</span></Tooltip> },
            {
              title: 'Status',
              dataIndex: 'status',
              key: 'status',
              render: (status: string) => (
                <Tag color={status === 'completed' ? 'green' : status === 'failed' ? 'red' : 'blue'}>
                  {status}
                </Tag>
              ),
            },
            { title: 'Model', dataIndex: 'model', key: 'model' },
            {
              title: 'Latency',
              dataIndex: 'latency_ms',
              key: 'latency_ms',
              render: (v?: number) => (v ? `${v}ms` : 'N/A'),
            },
            {
              title: 'Action',
              key: 'action',
              render: (_: unknown, _record: LLMCall) => (
                <Button size="small" onClick={() => navigate(`/executions/${runId}/calls`)}>
                  View
                </Button>
              ),
            },
          ]}
        />
      </Card>
    </Space>
  )
}

export default EvalReportView
