import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Tabs, Tag, Button, Space, Spin, message, List, Collapse, Row, Col } from 'antd'
import { HistoryOutlined, LinkOutlined } from '@ant-design/icons'
import { assetApi } from '../api/client'
import type { AssetDetail } from '../api/client'

interface EvalCasesViewProps {
  asset?: AssetDetail | null
}

function EvalCasesView({ asset: propAsset }: EvalCasesViewProps) {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (propAsset) {
      setAsset(propAsset)
      setLoading(false)
      return
    }
    if (id) {
      loadAsset(id)
    }
  }, [id, propAsset])

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

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!asset) {
    return <div>Asset not found</div>
  }

  const testCases = asset.test_cases || []
  const metricRefs = asset.metric_refs || []

  return (
    <div>
      <Card
        title={asset.name}
        extra={
          <Space>
            <Tag>{asset.category || 'eval'}</Tag>
            <Tag color={asset.state === 'active' ? 'green' : 'orange'}>{asset.state}</Tag>
          </Space>
        }
      >
        <Space direction="vertical" size="small" style={{ width: '100%' }}>
          <div>{asset.description || 'No description'}</div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {asset.tags?.map((tag) => (
              <Tag key={tag}>{tag}</Tag>
            ))}
          </div>
        </Space>
      </Card>

      <Tabs
        defaultActiveKey="overview"
        style={{ marginTop: 16 }}
        items={[
          {
            key: 'overview',
            label: 'Overview',
            children: (
              <Row gutter={16}>
                <Col span={12}>
                  <Card title="Basic Info">
                    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                      <div>
                        <span style={{ color: '#888' }}>Total Test Cases: </span>
                        <span>{testCases.length}</span>
                      </div>
                      <div>
                        <span style={{ color: '#888' }}>Referenced Metrics: </span>
                        <span>{metricRefs.length}</span>
                      </div>
                    </Space>
                  </Card>
                </Col>
                <Col span={12}>
                  <Card title="Referenced Metrics">
                    {metricRefs.length > 0 ? (
                      <Space direction="vertical" size="small">
                        {metricRefs.map((ref) => (
                          <Tag
                            key={ref}
                            icon={<LinkOutlined />}
                            color="blue"
                            style={{ cursor: 'pointer' }}
                            onClick={() => navigate(`/assets/${ref}`)}
                          >
                            {ref}
                          </Tag>
                        ))}
                      </Space>
                    ) : (
                      <span style={{ color: '#888' }}>No referenced metrics</span>
                    )}
                  </Card>
                </Col>
              </Row>
            ),
          },
          {
            key: 'cases_editor',
            label: 'Cases Editor',
            children: (
              <Row gutter={16}>
                {/* Left: Test Cases List */}
                <Col span={16}>
                  <Card title="Test Cases">
                    {testCases.length > 0 ? (
                      <Collapse
                        items={testCases.map((tc, idx) => ({
                          key: tc.id || idx,
                          label: (
                            <Space>
                              <span>Case {idx + 1}: {tc.name}</span>
                              {tc.description && <Tag>{tc.description}</Tag>}
                            </Space>
                          ),
                          children: (
                            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                              <div>
                                <strong style={{ color: '#888' }}>Input:</strong>
                                <Card size="small" bodyStyle={{ background: '#f5f5f5', padding: 8 }}>
                                  <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{tc.input}</pre>
                                </Card>
                              </div>
                              {tc.expected && (
                                <div>
                                  <strong style={{ color: '#888' }}>Expected:</strong>
                                  <Card size="small" bodyStyle={{ background: '#f5f5f5', padding: 8 }}>
                                    <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{tc.expected}</pre>
                                  </Card>
                                </div>
                              )}
                              {tc.rubric && (
                                <div>
                                  <strong style={{ color: '#888' }}>Rubric:</strong>
                                  <Card size="small" bodyStyle={{ background: '#f5f5f5', padding: 8 }}>
                                    <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{tc.rubric}</pre>
                                  </Card>
                                </div>
                              )}
                            </Space>
                          ),
                        }))}
                      />
                    ) : (
                      <div style={{ color: '#888', textAlign: 'center', padding: 40 }}>
                        No test cases defined
                      </div>
                    )}
                  </Card>
                </Col>

                {/* Right: Metric References */}
                <Col span={8}>
                  <Card title="Referenced Metrics">
                    {metricRefs.length > 0 ? (
                      <List
                        dataSource={metricRefs}
                        renderItem={(ref) => (
                          <List.Item>
                            <Tag
                              icon={<LinkOutlined />}
                              color="blue"
                              style={{ cursor: 'pointer' }}
                              onClick={() => navigate(`/assets/${ref}`)}
                            >
                              {ref}
                            </Tag>
                          </List.Item>
                        )}
                      />
                    ) : (
                      <div style={{ color: '#888' }}>No referenced metrics</div>
                    )}
                  </Card>
                </Col>
              </Row>
            ),
          },
          {
            key: 'versions',
            label: 'Versions',
            children: (
              <Card
                extra={
                  <Button icon={<HistoryOutlined />} onClick={() => navigate(`/assets/${id}/versions`)}>
                    View Full History
                  </Button>
                }
              >
                <div style={{ color: '#888', textAlign: 'center', padding: 40 }}>
                  Version history is available in the full history view
                </div>
              </Card>
            ),
          },
        ]}
      />
    </div>
  )
}

export default EvalCasesView
