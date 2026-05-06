import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Button, Space, Tag, Spin, message, Collapse, List, Row, Col, Checkbox, Modal, Form, Input } from 'antd'
import { PlusOutlined, PlayCircleOutlined, LinkOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { assetApi, evalApi } from '../../api/client'
import type { AssetDetail, TestCase } from '../../api/client'
import { useStore } from '../../store'

function EvalDesignView() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const addRunningEval = useStore((s) => s.addRunningEval)
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedCaseIds, setSelectedCaseIds] = useState<Set<string>>(new Set())
  const [addCaseModalOpen, setAddCaseModalOpen] = useState(false)
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)
  const [runningCaseId, setRunningCaseId] = useState<string | null>(null)
  const [runningSelected, setRunningSelected] = useState(false)
  const [activeKeys, setActiveKeys] = useState<(string | number)[]>([])

  useEffect(() => {
    if (id) loadAsset(id)
  }, [id])

  useEffect(() => {
    if (asset?.test_cases && asset.test_cases.length > 0) {
      setActiveKeys([asset.test_cases[0].id || 0])
    }
  }, [asset?.test_cases?.length])

  const loadAsset = async (assetId: string) => {
    setLoading(true)
    try {
      const data = await assetApi.get(assetId)
      setAsset(data)
    } catch {
      message.error('Failed to load asset')
    } finally {
      setLoading(false)
    }
  }

  const toggleCase = (caseId: string) => {
    setSelectedCaseIds((prev) => {
      const next = new Set(prev)
      if (next.has(caseId)) next.delete(caseId)
      else next.add(caseId)
      return next
    })
  }

  const handleRunSelected = async () => {
    if (!id || selectedCaseIds.size === 0 || runningSelected) return
    setRunningSelected(true)
    try {
      const result = await evalApi.execute({
        asset_id: id,
        case_ids: Array.from(selectedCaseIds),
        mode: 'single',
        concurrency: 1,
      })
      addRunningEval({
        id: result.execution_id,
        assetId: id,
        assetName: asset?.name || id,
        status: 'running',
        startedAt: Date.now(),
      })
      message.info('Eval started for selected cases')
    } catch {
      message.error('Failed to start eval')
    } finally {
      setRunningSelected(false)
    }
  }

  const handleRunCase = async (caseId: string) => {
    if (!id || runningCaseId) return
    setRunningCaseId(caseId)
    try {
      const result = await evalApi.execute({
        asset_id: id,
        case_ids: [caseId],
        mode: 'single',
        concurrency: 1,
      })
      addRunningEval({
        id: result.execution_id,
        assetId: id,
        assetName: asset?.name || id,
        status: 'running',
        startedAt: Date.now(),
      })
      message.info('Case eval started')
    } catch {
      message.error('Failed to start case eval')
    } finally {
      setRunningCaseId(null)
    }
  }

  const handleAddCase = async (values: { name: string; input: string; expected?: string }) => {
    if (!id || !asset) return
    setSubmitting(true)
    try {
      const newCase: TestCase = {
        id: `case_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
        name: values.name,
        input: values.input,
        expected: values.expected,
      }
      const updatedCases = [...(asset.test_cases || []), newCase]
      await assetApi.update(id, { test_cases: updatedCases } as any)
      setAsset({ ...asset, test_cases: updatedCases })
      setAddCaseModalOpen(false)
      form.resetFields()
      message.success('Test case added')
    } catch {
      message.error('Failed to add test case')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!asset) {
    return <div>{t('eval_cases_asset_not_found')}</div>
  }

  const testCases = asset.test_cases || []
  const metricRefs = asset.metric_refs || []

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Row gutter={16}>
        {/* Left: Test Cases */}
        <Col span={16}>
          <Card
            title={
              <Space>
                {t('eval_cases_test_cases')}
                {selectedCaseIds.size > 0 && (
                  <Tag color="blue">{selectedCaseIds.size} selected</Tag>
                )}
              </Space>
            }
            extra={
              <Space>
                {selectedCaseIds.size > 0 && (
                  <Button size="small" icon={<PlayCircleOutlined />} onClick={handleRunSelected} loading={runningSelected}>
                    Run Selected
                  </Button>
                )}
                {testCases.length > 0 && (
                  <>
                    <Button size="small" onClick={() => setActiveKeys(testCases.map((tc, idx) => tc.id || idx))}>
                      Expand All
                    </Button>
                    <Button size="small" onClick={() => setActiveKeys([])}>
                      Collapse All
                    </Button>
                  </>
                )}
                <Button size="small" icon={<PlusOutlined />} onClick={() => setAddCaseModalOpen(true)}>
                  Add Case
                </Button>
              </Space>
            }
          >
            {testCases.length > 0 ? (
              <Collapse
                activeKey={activeKeys}
                onChange={(keys) => setActiveKeys(keys as (string | number)[])}
                items={testCases.map((tc, idx) => ({
                  key: tc.id || idx,
                  label: (
                    <Space>
                      <Checkbox
                        checked={selectedCaseIds.has(tc.id || String(idx))}
                        onChange={() => toggleCase(tc.id || String(idx))}
                        onClick={(e) => e.stopPropagation()}
                      />
                      <span>{t('eval_cases_case')} {idx + 1}: {tc.name}</span>
                      {tc.description && <Tag>{tc.description}</Tag>}
                    </Space>
                  ),
                  children: (
                    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                      <div>
                        <strong style={{ color: '#888' }}>{t('eval_cases_input')}:</strong>
                        <Card size="small" bodyStyle={{ background: '#f5f5f5', padding: 8 }}>
                          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{tc.input}</pre>
                        </Card>
                      </div>
                      {tc.expected && (
                        <div>
                          <strong style={{ color: '#888' }}>{t('eval_cases_expected')}:</strong>
                          <Card size="small" bodyStyle={{ background: '#f5f5f5', padding: 8 }}>
                            <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{tc.expected}</pre>
                          </Card>
                        </div>
                      )}
                      {tc.rubric && (
                        <div>
                          <strong style={{ color: '#888' }}>{t('eval_cases_rubric')}:</strong>
                          <Card size="small" bodyStyle={{ background: '#f5f5f5', padding: 8 }}>
                            <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{tc.rubric}</pre>
                          </Card>
                        </div>
                      )}
                      <Button
                        size="small"
                        icon={<PlayCircleOutlined />}
                        onClick={() => handleRunCase(tc.id || String(idx))}
                        loading={runningCaseId === (tc.id || String(idx))}
                      >
                        Run this case
                      </Button>
                    </Space>
                  ),
                }))}
              />
            ) : (
              <div style={{ color: '#888', textAlign: 'center', padding: 40 }}>
                {t('eval_cases_no_test_cases')}
              </div>
            )}
          </Card>
        </Col>

        {/* Right: Metrics */}
        <Col span={8}>
          <Card title={t('eval_cases_referenced_metrics_card')}>
            {metricRefs.length > 0 ? (
              <List
                dataSource={metricRefs}
                renderItem={(ref) => (
                  <List.Item>
                    <Tag icon={<LinkOutlined />} color="blue">
                      {ref}
                    </Tag>
                  </List.Item>
                )}
              />
            ) : (
              <div style={{ color: '#888' }}>{t('eval_cases_no_referenced_metrics')}</div>
            )}
          </Card>
        </Col>
      </Row>

      <Modal
        title={t('eval_cases_add_case')}
        open={addCaseModalOpen}
        onCancel={() => { setAddCaseModalOpen(false); form.resetFields() }}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleAddCase}
        >
          <Form.Item
            name="name"
            label={t('eval_cases_name')}
            rules={[{ required: true, message: 'Please enter a name' }]}
          >
            <Input placeholder="e.g., Basic prompt test" />
          </Form.Item>
          <Form.Item
            name="input"
            label={t('eval_cases_input')}
            rules={[{ required: true, message: 'Please enter input' }]}
          >
            <Input.TextArea rows={4} placeholder="Test input..." />
          </Form.Item>
          <Form.Item
            name="expected"
            label={t('eval_cases_expected')}
          >
            <Input.TextArea rows={3} placeholder="Expected output (optional)" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>
                {t('common_add')}
              </Button>
              <Button onClick={() => { setAddCaseModalOpen(false); form.resetFields() }}>
                {t('common_cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  )
}

export default EvalDesignView
