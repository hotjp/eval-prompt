import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Input, Button, Space, message, Spin, Tabs, Tag } from 'antd'
import { SaveOutlined, PlayCircleOutlined, SwapOutlined } from '@ant-design/icons'
import MonacoEditor from '@monaco-editor/react'
import { assetApi, triggerApi } from '../api/client'
import type { AssetDetail } from '../api/client'

const { TextArea } = Input

function EditorView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [promptValue, setPromptValue] = useState('')
  const [variablesValue, setVariablesValue] = useState('{}\n\n{\n  "key": "value"\n}')
  const [injectedResult, setInjectedResult] = useState('')
  const [validating, setValidating] = useState(false)
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message?: string } | null>(null)

  useEffect(() => {
    if (id) {
      loadAsset(id)
    }
  }, [id])

  const loadAsset = async (assetId: string) => {
    setLoading(true)
    try {
      const data = await assetApi.get(assetId)
      setAsset(data)
      if (data.labels?.prompt) {
        setPromptValue(data.labels.prompt)
      }
    } catch {
      message.error('Failed to load asset')
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    if (!id) return
    setSaving(true)
    try {
      await assetApi.update(id, { labels: { ...asset?.labels, prompt: promptValue } } as any)
      message.success('Saved')
    } catch {
      message.error('Failed to save')
    } finally {
      setSaving(false)
    }
  }

  const handleValidate = async () => {
    setValidating(true)
    try {
      const result = await triggerApi.validate(promptValue)
      setValidationResult(result)
      if (result.valid) {
        message.success('Prompt is valid')
      } else {
        message.warning(result.message || 'Invalid prompt')
      }
    } catch {
      message.error('Validation failed')
    } finally {
      setValidating(false)
    }
  }

  const handleInject = async () => {
    try {
      const vars = JSON.parse(variablesValue.replace(/^\{/, '').replace(/\}$/, ''))
      const result = await triggerApi.inject(promptValue, vars)
      setInjectedResult(result.result)
    } catch {
      message.error('Failed to inject variables')
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!asset) {
    return <div>Asset not found</div>
  }

  return (
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card
          title={asset.name}
          extra={
            <Space>
              <Tag>{asset.biz_line}</Tag>
              <Tag color={asset.state === 'active' ? 'green' : 'orange'}>{asset.state}</Tag>
            </Space>
          }
        >
          <p>{asset.description}</p>
        </Card>

        <Tabs
          defaultActiveKey="editor"
          items={[
            {
              key: 'editor',
              label: 'Editor',
              children: (
                <Card>
                  <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                    <MonacoEditor
                      height="400px"
                      language="json"
                      value={promptValue}
                      onChange={(value) => setPromptValue(value || '')}
                      theme="vs-dark"
                      options={{ minimap: { enabled: false } }}
                    />
                    <Space>
                      <Button
                        type="primary"
                        icon={<SaveOutlined />}
                        onClick={handleSave}
                        loading={saving}
                      >
                        Save
                      </Button>
                      <Button
                        icon={<PlayCircleOutlined />}
                        onClick={handleValidate}
                        loading={validating}
                      >
                        Validate
                      </Button>
                    </Space>
                    {validationResult && (
                      <Tag color={validationResult.valid ? 'green' : 'red'}>
                        {validationResult.valid ? 'Valid' : validationResult.message}
                      </Tag>
                    )}
                  </Space>
                </Card>
              ),
            },
            {
              key: 'preview',
              label: 'Preview',
              children: (
                <Card title="Variable Injection">
                  <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                    <TextArea
                      rows={4}
                      value={variablesValue}
                      onChange={(e) => setVariablesValue(e.target.value)}
                      placeholder="Enter variables as JSON"
                    />
                    <Button type="primary" onClick={handleInject}>
                      Inject
                    </Button>
                    {injectedResult && (
                      <Card title="Result" style={{ background: '#f5f5f5' }}>
                        <pre>{injectedResult}</pre>
                      </Card>
                    )}
                  </Space>
                </Card>
              ),
            },
          ]}
        />

        <Card title="Snapshots">
          <Space>
            <Button onClick={() => navigate(`/assets/${id}/versions`)}>
              View Version Tree
            </Button>
            <Button onClick={() => navigate(`/assets/${id}/eval`)}>
              Run Evaluation
            </Button>
            <Button icon={<SwapOutlined />} onClick={() => navigate('/compare')}>
              Compare
            </Button>
          </Space>
        </Card>
      </Space>
    </div>
  )
}

export default EditorView
