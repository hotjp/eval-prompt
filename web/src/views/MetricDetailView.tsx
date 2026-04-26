import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Tabs, Tag, Button, Space, Spin, message, List, Collapse, Row, Col } from 'antd'
import { EditOutlined, HistoryOutlined, LinkOutlined } from '@ant-design/icons'
import { assetApi } from '../api/client'
import type { AssetDetail } from '../api/client'

interface MetricDetailViewProps {
  asset?: AssetDetail | null
}

function MetricDetailView({ asset: propAsset }: MetricDetailViewProps) {
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

  const rubric = asset.rubric || []
  const usedBy = asset.used_by || []

  return (
    <div>
      <Card
        title={asset.name}
        extra={
          <Space>
            <Tag>{asset.category || 'metric'}</Tag>
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
                  <Card title="Metric Info">
                    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                      <div>
                        <span style={{ color: '#888' }}>Total Rubric Checks: </span>
                        <span>{rubric.length}</span>
                      </div>
                      <div>
                        <span style={{ color: '#888' }}>Total Weight: </span>
                        <span>{rubric.reduce((sum, r) => sum + r.weight, 0)}%</span>
                      </div>
                    </Space>
                  </Card>
                </Col>
                <Col span={12}>
                  <Card title="Used By">
                    {usedBy.length > 0 ? (
                      <Space direction="vertical" size="small">
                        {usedBy.map((ref) => (
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
                      <span style={{ color: '#888' }}>Not used by any eval cases</span>
                    )}
                  </Card>
                </Col>
              </Row>
            ),
          },
          {
            key: 'rubric_editor',
            label: 'Rubric Editor',
            children: (
              <Row gutter={16}>
                {/* Left: Rubric List */}
                <Col span={16}>
                  <Card
                    title="Rubric Checks"
                    extra={
                      <Button icon={<EditOutlined />} size="small">
                        Edit Rubric
                      </Button>
                    }
                  >
                    {rubric.length > 0 ? (
                      <Collapse
                        items={rubric.map((item, idx) => ({
                          key: idx,
                          label: (
                            <Space>
                              <span>{item.check}</span>
                              <Tag>{(item.weight * 100).toFixed(0)}%</Tag>
                            </Space>
                          ),
                          children: (
                            <Card size="small" bodyStyle={{ background: '#f5f5f5', padding: 8 }}>
                              <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{item.criteria}</pre>
                            </Card>
                          ),
                        }))}
                      />
                    ) : (
                      <div style={{ color: '#888', textAlign: 'center', padding: 40 }}>
                        No rubric defined
                      </div>
                    )}
                  </Card>
                </Col>

                {/* Right: Used By */}
                <Col span={8}>
                  <Card title="Used By">
                    {usedBy.length > 0 ? (
                      <List
                        dataSource={usedBy}
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
                      <div style={{ color: '#888' }}>Not used by any eval cases</div>
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

export default MetricDetailView
