import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Button, Space, Select, InputNumber, Progress, Tag, message, Row, Col, Statistic, Input, Radio, Checkbox, Tooltip } from 'antd'
import { PlayCircleOutlined, StopOutlined, SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { evalApi, executionApi, llmConfigApi } from '../../api/client'
import type { Execution, EvalReport, OrchestrateResponse } from '../../api/client'
import { useStore } from '../../store'

interface Preset {
  name: string
  mode: string
  model: string
  temperature: number
  concurrency: number
}

const PRESETS_KEY = 'eval-presets'

function loadPresets(): Preset[] {
  try {
    const raw = localStorage.getItem(PRESETS_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

function savePresets(presets: Preset[]) {
  localStorage.setItem(PRESETS_KEY, JSON.stringify(presets))
}

function EvalRunView() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const evalConcurrency = useStore((s) => s.evalConcurrency)
  const runningEvals = useStore((s) => s.runningEvals)
  const addRunningEval = useStore((s) => s.addRunningEval)

  const [models, setModels] = useState<string[]>([])
  const [modelsLoading, setModelsLoading] = useState(false)
  const [evalMode, setEvalMode] = useState<string>('single')
  const [evalModel, setEvalModel] = useState<string>('')
  const [evalTemperature, setEvalTemperature] = useState<number>(0.7)
  const [concurrency, setConcurrency] = useState<number>(evalConcurrency)
  const [executing, setExecuting] = useState(false)
  const [localExecution, setLocalExecution] = useState<Execution | null>(null)
  const [latestReport, setLatestReport] = useState<EvalReport | null>(null)
  const [presets, setPresets] = useState<Preset[]>(loadPresets)
  const [selectedPreset, setSelectedPreset] = useState<string>('')
  const [savePresetName, setSavePresetName] = useState('')
  // Orchestrate-specific state
  const [evalType, setEvalType] = useState<'single' | 'orchestrate'>('single')
  const [selectedPlugins, setSelectedPlugins] = useState<string[]>(['bertscore'])
  const [injectionStrategy, setInjectionStrategy] = useState<string>('default')
  const [parallelism, setParallelism] = useState<number>(1)
  const [confidenceLevel, setConfidenceLevel] = useState<number>(0.95)
  const [baselineId, setBaselineId] = useState<string>('')
  const [orchestrateResult, setOrchestrateResult] = useState<OrchestrateResponse | null>(null)

  // Fetch available models from backend
  useEffect(() => {
    setModelsLoading(true)
    llmConfigApi
      .get()
      .then((configs) => {
        const modelSet = new Set<string>()
        configs.forEach((c) => {
          if (c.default_model) modelSet.add(c.default_model)
        })
        setModels(Array.from(modelSet))
      })
      .catch(() => {
        message.warning('Failed to load model list')
      })
      .finally(() => setModelsLoading(false))
  }, [])

  // Find running eval for this asset
  const assetRunningEvals = runningEvals.filter((e) => e.assetId === id)
  const hasRunning = assetRunningEvals.some((e) => e.status === 'pending' || e.status === 'running')

  // Poll local execution status for detailed view
  useEffect(() => {
    if (!localExecution?.id || !hasRunning) return

    let cancelled = false
    const poll = async () => {
      try {
        const exec = await executionApi.get(localExecution.id)
        if (cancelled) return
        setLocalExecution(exec)
        if (exec.status === 'running' || exec.status === 'pending' || exec.status === 'initializing') {
          setTimeout(poll, 2000)
        } else {
          // Execution finished, fetch report for score snapshot
          try {
            const report = await evalApi.report(exec.id)
            setLatestReport(report)
          } catch {
            // ignore report fetch errors
          }
        }
      } catch {
        // ignore
      }
    }
    poll()
    return () => {
      cancelled = true
    }
  }, [localExecution?.id, hasRunning])

  const handleLoadPreset = (presetName: string) => {
    const preset = presets.find((p) => p.name === presetName)
    if (preset) {
      setEvalMode(preset.mode)
      setEvalModel(preset.model)
      setEvalTemperature(preset.temperature)
      setConcurrency(preset.concurrency)
      setSelectedPreset(presetName)
    }
  }

  const handleSavePreset = () => {
    if (!savePresetName.trim()) {
      message.warning('Please enter a preset name')
      return
    }
    const newPreset: Preset = {
      name: savePresetName.trim(),
      mode: evalMode,
      model: evalModel,
      temperature: evalTemperature,
      concurrency,
    }
    const updated = [...presets.filter((p) => p.name !== newPreset.name), newPreset]
    setPresets(updated)
    savePresets(updated)
    setSelectedPreset(newPreset.name)
    message.success('Preset saved')
  }

  const handleRunEval = async () => {
    if (!id) return
    setExecuting(true)
    setOrchestrateResult(null)
    try {
      if (evalType === 'orchestrate') {
        if (selectedPlugins.length === 0) {
          message.warning(t('eval_orchestrate_select_plugin_warning'))
          setExecuting(false)
          return
        }
        // Multi-plugin orchestration
        const result = await evalApi.orchestrate({
          asset_id: id,
          plugins: selectedPlugins,
          injection_strategy: injectionStrategy,
          parallelism,
          confidence_level: confidenceLevel,
          baseline_id: baselineId || undefined,
        })
        setOrchestrateResult(result)
        message.success(t('eval_orchestrate_completed'))
      } else {
        // Single plugin evaluation (existing flow)
        const result = await evalApi.execute({
          asset_id: id,
          mode: evalMode,
          concurrency,
          model: evalModel || undefined,
          temperature: evalTemperature,
        })

        addRunningEval({
          id: result.execution_id,
          assetId: id,
          assetName: id,
          status: 'running',
          startedAt: Date.now(),
        })

        // Fetch initial execution state for local display
        const exec = await executionApi.get(result.execution_id)
        setLocalExecution(exec)

        message.info('Eval started: ' + result.execution_id.slice(-8))
      }
    } catch {
      message.error('Failed to start eval')
    } finally {
      setExecuting(false)
    }
  }

  const handleCancelExecution = async (executionId: string) => {
    try {
      await evalApi.cancelExecution(executionId)
      message.info('Cancellation requested')
    } catch {
      message.error('Failed to cancel execution')
    }
  }

  const latestCompleted = runningEvals.find(
    (e) => e.assetId === id && (e.status === 'completed' || e.status === 'failed')
  )

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* Run Configuration Card */}
      <Card title={t('eval_panel_run_eval')}>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          {/* Eval Type Toggle */}
          <Radio.Group
            value={evalType}
            onChange={(e) => setEvalType(e.target.value)}
            optionType="button"
            buttonStyle="solid"
          >
            <Radio.Button value="single">{t('eval_orchestrate_single_plugin')}</Radio.Button>
            <Radio.Button value="orchestrate">{t('eval_orchestrate_multi_plugin')}</Radio.Button>
          </Radio.Group>

          {/* Orchestrate-specific fields */}
          {evalType === 'orchestrate' && (
            <div style={{ padding: '12px 16px', background: '#f6ffed', borderRadius: 6, border: '1px solid #b7eb8f' }}>
              <Row gutter={16}>
                <Col span={12}>
                  <div style={{ marginBottom: 8, fontSize: 12, color: '#888' }}>{t('eval_orchestrate_plugins')}</div>
                  <Checkbox.Group
                    value={selectedPlugins}
                    onChange={(vals) => setSelectedPlugins(vals as string[])}
                    style={{ width: '100%' }}
                    options={[
                      { label: 'G-Eval', value: 'geval' },
                      { label: 'BERTScore', value: 'bertscore' },
                      { label: 'Belief Revision', value: 'beliefrevision' },
                      { label: 'Constraint', value: 'constraint' },
                      { label: 'FACTScore', value: 'factscore' },
                      { label: 'SelfCheckGPT', value: 'selfcheck' },
                    ]}
                  />
                </Col>
                <Col span={12}>
                  <div style={{ marginBottom: 8, fontSize: 12, color: '#888' }}>{t('eval_orchestrate_injection_strategy')}</div>
                  <Select
                    value={injectionStrategy}
                    onChange={setInjectionStrategy}
                    style={{ width: '100%' }}
                    options={[
                      { value: 'default', label: 'Default' },
                      { value: 'position_swap', label: 'Position Swap' },
                      { value: 'constraint_conflict', label: 'Constraint Conflict' },
                      { value: 'adversarial_prefix', label: 'Adversarial Prefix' },
                    ]}
                  />
                </Col>
              </Row>
              <Row gutter={16} style={{ marginTop: 12 }}>
                <Col span={8}>
                  <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>{t('eval_orchestrate_parallelism')}</div>
                  <InputNumber
                    value={parallelism}
                    onChange={(v) => setParallelism(v ?? 1)}
                    min={1}
                    max={10}
                    style={{ width: '100%' }}
                  />
                </Col>
                <Col span={8}>
                  <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>{t('eval_orchestrate_confidence_level')}</div>
                  <InputNumber
                    value={confidenceLevel}
                    onChange={(v) => setConfidenceLevel(v ?? 0.95)}
                    min={0.5}
                    max={0.99}
                    step={0.01}
                    style={{ width: '100%' }}
                  />
                </Col>
                <Col span={8}>
                  <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>
                    <Tooltip title="Optional: Compare against a baseline snapshot">
                      {t('eval_orchestrate_baseline_id')}
                    </Tooltip>
                  </div>
                  <Input
                    value={baselineId}
                    onChange={(e) => setBaselineId(e.target.value)}
                    placeholder="Optional"
                    style={{ width: '100%' }}
                  />
                </Col>
              </Row>
            </div>
          )}

          <Row gutter={16}>
            <Col span={6}>
              <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>Mode</div>
              <Select
                value={evalMode}
                onChange={setEvalMode}
                style={{ width: '100%' }}
                disabled={hasRunning}
                options={[
                  { value: 'single', label: t('eval_panel_mode_single') },
                  { value: 'batch', label: t('eval_panel_mode_batch') },
                  { value: 'matrix', label: t('eval_panel_mode_matrix') },
                ]}
              />
            </Col>
            <Col span={6}>
              <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>Model</div>
              <Select
                value={evalModel || undefined}
                onChange={setEvalModel}
                style={{ width: '100%' }}
                disabled={hasRunning || modelsLoading}
                loading={modelsLoading}
                placeholder={t('eval_panel_select_model')}
                allowClear
                options={models.map((m) => ({ value: m, label: m }))}
              />
            </Col>
            <Col span={6}>
              <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>Temperature</div>
              <InputNumber
                value={evalTemperature}
                onChange={(v) => setEvalTemperature(v ?? 0.7)}
                min={0}
                max={2}
                step={0.1}
                disabled={hasRunning}
                style={{ width: '100%' }}
              />
            </Col>
            <Col span={6}>
              <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>Concurrency</div>
              <InputNumber
                value={concurrency}
                onChange={(v) => setConcurrency(v ?? 1)}
                min={1}
                max={10}
                disabled={hasRunning}
                style={{ width: '100%' }}
              />
            </Col>
          </Row>

          {/* Presets */}
          <Space wrap>
            <Select
              placeholder="Load preset"
              value={selectedPreset || undefined}
              onChange={handleLoadPreset}
              style={{ width: 160 }}
              allowClear
              options={presets.map((p) => ({ value: p.name, label: p.name }))}
            />
            <Space>
              <Input
                placeholder="Preset name"
                value={savePresetName}
                onChange={(e) => setSavePresetName(e.target.value)}
                style={{ width: 140 }}
                size="small"
              />
              <Button icon={<SaveOutlined />} onClick={handleSavePreset} size="small">
                Save Preset
              </Button>
            </Space>
          </Space>

          {/* Actions */}
          <Space>
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              onClick={handleRunEval}
              loading={executing}
              disabled={hasRunning}
            >
              {t('eval_panel_run_eval_button')}
            </Button>
          </Space>
        </Space>
      </Card>

      {/* Active Executions for this asset */}
      {assetRunningEvals.length > 0 && (
        <Card title="Active Executions">
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            {assetRunningEvals.map((re) => (
              <div
                key={re.id}
                style={{
                  padding: 12,
                  background: '#f6ffed',
                  borderRadius: 6,
                  border: '1px solid #b7eb8f',
                }}
              >
                <Space direction="vertical" size="small" style={{ width: '100%' }}>
                  <Space>
                    <Tag color={re.status === 'running' ? 'blue' : re.status === 'pending' ? 'orange' : 'default'}>
                      {re.status}
                    </Tag>
                    <code style={{ fontSize: 11 }}>{re.id.slice(-8)}</code>
                  </Space>
                  {re.progress && re.progress.total > 0 && (
                    <Progress
                      percent={Math.round((re.progress.completed / re.progress.total) * 100)}
                      status={re.status === 'running' ? 'active' : 'normal'}
                      format={() => `${re.progress!.completed} / ${re.progress!.total} cases`}
                    />
                  )}
                  {(re.status === 'running' || re.status === 'pending') && (
                    <Button
                      danger
                      size="small"
                      icon={<StopOutlined />}
                      onClick={() => handleCancelExecution(re.id)}
                    >
                      {t('eval_panel_cancel')}
                    </Button>
                  )}
                </Space>
              </div>
            ))}
          </Space>
        </Card>
      )}

      {/* Latest Result Snapshot */}
      {latestCompleted && localExecution && localExecution.status !== 'running' && localExecution.status !== 'pending' && (
        <Card title="Latest Result">
          <Row gutter={16}>
            <Col span={latestReport ? 4 : 6}>
              <Statistic title="Status" value={localExecution.status} />
            </Col>
            <Col span={latestReport ? 4 : 6}>
              <Statistic
                title="Progress"
                value={`${localExecution.completed_cases} / ${localExecution.total_cases}`}
              />
            </Col>
            {latestReport && (
              <>
                <Col span={4}>
                  <Statistic title="Overall" value={latestReport.overall_score} precision={2} suffix="/ 1.0" />
                </Col>
                <Col span={4}>
                  <Statistic title="Deterministic" value={latestReport.deterministic_score} precision={2} suffix="/ 1.0" />
                </Col>
                <Col span={4}>
                  <Statistic title="Rubric" value={latestReport.rubric_score} precision={2} suffix="/ 1.0" />
                </Col>
              </>
            )}
            {!latestReport && (
              <>
                <Col span={6}>
                  <Statistic title="Model" value={localExecution.model || 'N/A'} />
                </Col>
                <Col span={6}>
                  <Statistic title="Temperature" value={localExecution.temperature} />
                </Col>
              </>
            )}
          </Row>
          {latestReport && (
            <div style={{ marginTop: 12, textAlign: 'right' }}>
              <Button size="small" onClick={() => navigate(`/assets/${id}/eval/report/${localExecution.id}`)}>
                View Full Report →
              </Button>
            </div>
          )}
        </Card>
      )}

      {/* Orchestrate Result */}
      {orchestrateResult && (
        <Card title={t('eval_orchestrate_result')}>
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
              <Col span={6}>
                <Statistic
                  title={t('eval_orchestrate_score_delta')}
                  value={orchestrateResult.baseline_comparison.score_delta}
                  precision={2}
                  valueStyle={{ color: orchestrateResult.baseline_comparison.score_delta >= 0 ? '#52c41a' : '#ff4d4f' }}
                />
              </Col>
            )}
            {orchestrateResult.elo_result && (
              <Col span={6}>
                <Statistic title={t('eval_orchestrate_elo_rating')} value={orchestrateResult.elo_result.new_rating} />
              </Col>
            )}
          </Row>

          {/* Plugin Results */}
          {Object.keys(orchestrateResult.plugin_results).length > 0 && (
            <div style={{ marginTop: 16 }}>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>{t('eval_orchestrate_plugin_results')}</div>
              <Row gutter={16}>
                {Object.entries(orchestrateResult.plugin_results).map(([pluginName, result]) => (
                  <Col span={6} key={pluginName}>
                    <Card size="small" title={pluginName}>
                      <Statistic
                        title="Score"
                        value={result.score}
                        precision={2}
                        suffix="/ 1.0"
                      />
                      {result.confidence_interval && (
                        <div style={{ fontSize: 11, color: '#888', marginTop: 4 }}>
                          CI: [{result.confidence_interval.low.toFixed(2)}, {result.confidence_interval.high.toFixed(2)}]
                        </div>
                      )}
                      <div style={{ fontSize: 11, color: '#888', marginTop: 4 }}>
                        {result.work_item_results.length} work items
                      </div>
                    </Card>
                  </Col>
                ))}
              </Row>
            </div>
          )}

          {/* Summary */}
          {orchestrateResult.summary && (
            <div style={{ marginTop: 16 }}>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 4 }}>{t('eval_orchestrate_summary')}</div>
              <div style={{ padding: 12, background: '#f5f5f5', borderRadius: 6 }}>
                {orchestrateResult.summary}
              </div>
            </div>
          )}

          {/* Baseline Comparison Details */}
          {orchestrateResult.baseline_comparison && (
            <div style={{ marginTop: 16 }}>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>{t('eval_orchestrate_baseline_comparison')}</div>
              <Row gutter={16}>
                <Col span={6}>
                  <Statistic title={t('eval_orchestrate_effect_size')} value={orchestrateResult.baseline_comparison.effect_size} precision={3} />
                </Col>
                <Col span={6}>
                  <Statistic title={t('eval_orchestrate_p_value')} value={orchestrateResult.baseline_comparison.p_value} precision={4} />
                </Col>
                <Col span={6}>
                  <Tag color={orchestrateResult.baseline_comparison.is_significant ? 'green' : 'orange'}>
                    {t(orchestrateResult.baseline_comparison.is_significant ? 'eval_orchestrate_significant' : 'eval_orchestrate_not_significant')}
                  </Tag>
                </Col>
              </Row>
            </div>
          )}
        </Card>
      )}
    </Space>
  )
}

export default EvalRunView
