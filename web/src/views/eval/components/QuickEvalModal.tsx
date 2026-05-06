import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Modal, Button, Space, Select, InputNumber, Progress, message, Statistic, Row, Col } from 'antd'
import { PlayCircleOutlined, StopOutlined } from '@ant-design/icons'
import { evalApi, executionApi, llmConfigApi } from '../../../api/client'
import type { Execution, EvalReport } from '../../../api/client'
import { useStore } from '../../../store'

interface QuickEvalModalProps {
  assetId: string
  assetName: string
  open: boolean
  onClose: () => void
}

function QuickEvalModal({ assetId, assetName, open, onClose }: QuickEvalModalProps) {
  const navigate = useNavigate()
  const [models, setModels] = useState<string[]>([])
  const [modelsLoading, setModelsLoading] = useState(false)
  const [evalMode, setEvalMode] = useState<string>('single')
  const [evalModel, setEvalModel] = useState<string>('')
  const [evalTemperature, setEvalTemperature] = useState<number>(0.7)
  const [concurrency, setConcurrency] = useState<number>(1)
  const [executing, setExecuting] = useState(false)
  const [executionId, setExecutionId] = useState<string | null>(null)
  const [execution, setExecution] = useState<Execution | null>(null)
  const [report, setReport] = useState<EvalReport | null>(null)
  const [cancelling, setCancelling] = useState(false)

  const runningEvals = useStore((s) => s.runningEvals)
  const addRunningEval = useStore((s) => s.addRunningEval)
  const updateRunningEval = useStore((s) => s.updateRunningEval)


  const thisRunningEval = runningEvals.find(
    (e) => e.assetId === assetId && (e.status === 'running' || e.status === 'pending')
  )

  // Reset state when modal opens
  useEffect(() => {
    if (open) {
      setExecutionId(null)
      setExecution(null)
      setReport(null)
      setCancelling(false)
      setModelsLoading(true)
      llmConfigApi
        .get()
        .then((configs) => {
          const modelSet = new Set<string>()
          configs.forEach((c) => {
            if (c.default_model) modelSet.add(c.default_model)
          })
          const modelList = Array.from(modelSet)
          setModels(modelList)
          if (!evalModel && modelList.length > 0) {
            setEvalModel(modelList[0])
          }
        })
        .catch(() => {
          message.warning('Failed to load model list')
        })
        .finally(() => setModelsLoading(false))
    }
  }, [open])

  // Poll local execution status
  useEffect(() => {
    if (!executionId || !thisRunningEval) return

    let cancelled = false
    const poll = async () => {
      try {
        const exec = await executionApi.get(executionId)
        if (cancelled) return
        setExecution(exec)
        if (exec.status === 'running' || exec.status === 'pending' || exec.status === 'initializing') {
          setTimeout(poll, 2000)
        } else {
          // Fetch report on completion
          try {
            const r = await evalApi.report(executionId)
            setReport(r)
          } catch {
            // ignore
          }
        }
      } catch {
        // stop polling on error
        return
      }
    }
    poll()
    return () => {
      cancelled = true
    }
  }, [executionId, thisRunningEval?.status])

  const handleRun = async () => {
    setExecuting(true)
    try {
      const result = await evalApi.execute({
        asset_id: assetId,
        mode: evalMode,
        concurrency,
        model: evalModel || undefined,
        temperature: evalTemperature,
      })

      addRunningEval({
        id: result.execution_id,
        assetId,
        assetName,
        status: 'running',
        startedAt: Date.now(),
      })

      setExecutionId(result.execution_id)
      const exec = await executionApi.get(result.execution_id)
      setExecution(exec)
      message.info('Eval started')
    } catch {
      message.error('Failed to start eval')
    } finally {
      setExecuting(false)
    }
  }

  const handleCancel = async () => {
    if (!executionId) return
    setCancelling(true)
    try {
      await evalApi.cancelExecution(executionId)
      updateRunningEval(executionId, { status: 'cancelling' })
      message.info('Cancellation requested')
    } catch {
      message.error('Failed to cancel')
    } finally {
      setCancelling(false)
    }
  }

  const isRunning =
    thisRunningEval !== undefined ||
    (execution && ['running', 'pending', 'initializing'].includes(execution.status))
  const isCompleted =
    execution && ['completed', 'passed', 'failed', 'cancelled'].includes(execution.status)

  return (
    <Modal title={`Quick Eval: ${assetName}`} open={open} onCancel={onClose} footer={null} width={520}>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* Config */}
        <Row gutter={12}>
          <Col span={12}>
            <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>Model</div>
            <Select
              value={evalModel || undefined}
              onChange={setEvalModel}
              style={{ width: '100%' }}
              disabled={!!isRunning}
              loading={modelsLoading}
              placeholder="Select model"
              allowClear
              options={models.map((m) => ({ value: m, label: m }))}
            />
          </Col>
          <Col span={12}>
            <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>Mode</div>
            <Select
              value={evalMode}
              onChange={setEvalMode}
              style={{ width: '100%' }}
              disabled={!!isRunning}
              options={[
                { value: 'single', label: 'Single' },
                { value: 'batch', label: 'Batch' },
                { value: 'matrix', label: 'Matrix' },
              ]}
            />
          </Col>
        </Row>
        <Row gutter={12}>
          <Col span={12}>
            <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>Temperature</div>
            <InputNumber
              value={evalTemperature}
              onChange={(v) => setEvalTemperature(v ?? 0.7)}
              min={0}
              max={2}
              step={0.1}
              disabled={!!isRunning}
              style={{ width: '100%' }}
            />
          </Col>
          <Col span={12}>
            <div style={{ marginBottom: 4, fontSize: 12, color: '#888' }}>Concurrency</div>
            <InputNumber
              value={concurrency}
              onChange={(v) => setConcurrency(v ?? 1)}
              min={1}
              max={10}
              disabled={!!isRunning}
              style={{ width: '100%' }}
            />
          </Col>
        </Row>

        {/* Actions */}
        <Space style={{ width: '100%' }}>
          <Button
            type="primary"
            icon={<PlayCircleOutlined />}
            onClick={handleRun}
            loading={executing}
            disabled={!!isRunning}
            style={{ flex: 1 }}
            block
          >
            {isRunning ? 'Eval Running...' : 'Start Eval'}
          </Button>
          {isRunning && (
            <Button danger icon={<StopOutlined />} onClick={handleCancel} loading={cancelling}>
              Cancel
            </Button>
          )}
        </Space>

        {/* Progress */}
        {isRunning && execution && (
          <div style={{ padding: 12, background: '#f6ffed', borderRadius: 6 }}>
            {execution.total_cases === 0 ? (
              <Progress percent={0} status="active" showInfo={false} />
            ) : (
              <Progress
                percent={Math.round((execution.completed_cases / execution.total_cases) * 100)}
                status="active"
                format={() => `${execution!.completed_cases} / ${execution!.total_cases} cases`}
              />
            )}
            {execution.total_cases === 0 && (
              <div style={{ textAlign: 'center', fontSize: 12, color: '#888' }}>Initializing...</div>
            )}
          </div>
        )}

        {/* Result Snapshot */}
        {isCompleted && execution && (
          <div
            style={{
              padding: 12,
              background: '#f6ffed',
              borderRadius: 6,
              border: '1px solid #b7eb8f',
            }}
          >
            <Row gutter={16}>
              <Col span={report ? 6 : 12}>
                <Statistic
                  title="Status"
                  value={execution.status}
                  valueStyle={{
                    color:
                      execution.status === 'passed' || execution.status === 'completed'
                        ? '#52c41a'
                        : '#ff4d4f',
                  }}
                />
              </Col>
              {report && (
                <>
                  <Col span={6}>
                    <Statistic title="Overall" value={report.overall_score ?? 0} suffix="/ 100" />
                  </Col>
                  <Col span={6}>
                    <Statistic
                      title="Deterministic"
                      value={Math.round((report.deterministic_score ?? 0) * 100)}
                      suffix="/ 100"
                    />
                  </Col>
                  <Col span={6}>
                    <Statistic title="Rubric" value={report.rubric_score ?? 0} suffix="/ 100" />
                  </Col>
                </>
              )}
              {!report && (
                <Col span={12}>
                  <Statistic
                    title="Progress"
                    value={`${execution.completed_cases} / ${execution.total_cases}`}
                  />
                </Col>
              )}
            </Row>
            <div style={{ marginTop: 12, textAlign: 'center' }}>
              <Button
                type="primary"
                onClick={() => {
                  onClose()
                  navigate(`/assets/${assetId}/eval/report/${executionId}`)
                }}
              >
                View Full Report →
              </Button>
            </div>
          </div>
        )}
      </Space>
    </Modal>
  )
}

export default QuickEvalModal
