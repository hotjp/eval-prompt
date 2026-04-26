import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, List, Tag, Button, Space, Spin, Tabs, message } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import { executionApi } from '../api/client'
import type { Execution, LLMCall } from '../api/client'

function CallLogView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [execution, setExecution] = useState<Execution | null>(null)
  const [calls, setCalls] = useState<LLMCall[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedCall, setSelectedCall] = useState<LLMCall | null>(null)

  useEffect(() => {
    if (id) {
      loadData(id)
    }
  }, [id])

  const loadData = async (executionId: string) => {
    setLoading(true)
    try {
      const [execData, callsData] = await Promise.all([
        executionApi.get(executionId),
        executionApi.getCalls(executionId),
      ])
      setExecution(execData)
      setCalls(callsData)
      if (callsData.length > 0) {
        setSelectedCall(callsData[0])
      }
    } catch {
      message.error('Failed to load execution data')
    } finally {
      setLoading(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />
      case 'failed':
        return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
      case 'running':
        return <Tag color="blue">Running</Tag>
      default:
        return <Tag>{status}</Tag>
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'green'
      case 'failed': return 'red'
      default: return 'default'
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!execution) {
    return <div>Execution not found</div>
  }

  return (
    <div>
      <Card
        title={<Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/executions')} size="small" />
          Execution: {execution.id}
        </Space>}
        extra={<Tag color={getStatusColor(execution.status)}>{execution.status}</Tag>}
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Space split={<span style={{ color: '#888' }}>|</span>}>
            <span>Model: {execution.model}</span>
            <span>Temperature: {execution.temperature}</span>
            <span>Progress: {execution.completed_cases}/{execution.total_cases}</span>
          </Space>
        </Space>
      </Card>

      <div style={{ display: 'flex', gap: 16, marginTop: 16 }}>
        {/* Left: Call List */}
        <Card
          title="Calls"
          style={{ width: 280, flexShrink: 0 }}
          bodyStyle={{ padding: 0, maxHeight: 600, overflow: 'auto' }}
        >
          <List
            dataSource={calls}
            rowKey="id"
            renderItem={(call) => (
              <List.Item
                onClick={() => setSelectedCall(call)}
                style={{
                  padding: '8px 12px',
                  cursor: 'pointer',
                  background: selectedCall?.id === call.id ? '#e6f7ff' : 'transparent',
                }}
              >
                <Space direction="vertical" size="small" style={{ width: '100%' }}>
                  <Space>
                    {getStatusIcon(call.status)}
                    <span style={{ fontSize: 11, fontFamily: 'monospace' }}>
                      {call.run_id.slice(-8)}
                    </span>
                  </Space>
                  <div style={{ fontSize: 11, color: '#888' }}>
                    {call.created_at ? new Date(call.created_at).toLocaleTimeString() : ''}
                  </div>
                </Space>
              </List.Item>
            )}
          />
        </Card>

        {/* Right: Call Detail */}
        <Card style={{ flex: 1 }} title="Call Detail">
          {selectedCall ? (
            <Tabs
              defaultActiveKey="prompt"
              items={[
                {
                  key: 'prompt',
                  label: 'Prompt',
                  children: (
                    <Card size="small" bodyStyle={{ background: '#f5f5f5' }}>
                      <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>
                        {selectedCall.prompt_content || 'No prompt content'}
                      </pre>
                    </Card>
                  ),
                },
                {
                  key: 'response',
                  label: 'Response',
                  children: (
                    <Card size="small" bodyStyle={{ background: '#f5f5f5' }}>
                      <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>
                        {selectedCall.response_content || 'No response content'}
                      </pre>
                    </Card>
                  ),
                },
                {
                  key: 'raw',
                  label: 'Raw JSON',
                  children: (
                    <Card size="small" bodyStyle={{ background: '#f5f5f5' }}>
                      <pre style={{ whiteSpace: 'pre-wrap', fontSize: 11 }}>
                        {selectedCall.raw_json || JSON.stringify(selectedCall, null, 2)}
                      </pre>
                    </Card>
                  ),
                },
              ]}
            />
          ) : (
            <div style={{ color: '#888', textAlign: 'center', padding: 40 }}>
              Select a call to view details
            </div>
          )}

          {/* Metadata footer */}
          {selectedCall && (
            <div style={{ marginTop: 16, padding: '12px 16px', background: '#fafafa', borderRadius: 4 }}>
              <Space split={<span style={{ color: '#888' }}>|</span>}>
                <span>Model: {selectedCall.model}</span>
                <span>Temperature: {selectedCall.temperature}</span>
                {selectedCall.tokens_in !== undefined && <span>Tokens in: {selectedCall.tokens_in}</span>}
                {selectedCall.tokens_out !== undefined && <span>Tokens out: {selectedCall.tokens_out}</span>}
                {selectedCall.latency_ms !== undefined && <span>Latency: {selectedCall.latency_ms}ms</span>}
              </Space>
              {selectedCall.error && (
                <div style={{ marginTop: 8, color: '#ff4d4f' }}>
                  Error: {selectedCall.error}
                </div>
              )}
            </div>
          )}
        </Card>
      </div>
    </div>
  )
}

export default CallLogView
