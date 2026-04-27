import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Tabs, Tag, Button, Space, Spin, message, Timeline, Row, Col, Statistic } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, EditOutlined, HistoryOutlined, PlayCircleOutlined } from '@ant-design/icons'
import { assetApi } from '../api/client'
import type { AssetDetail, EvalHistoryEntry } from '../api/client'
import { useTranslation } from 'react-i18next'

interface ContentDetailViewProps {
  asset?: AssetDetail | null
}

function ContentDetailView({ asset: propAsset }: ContentDetailViewProps) {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedHistory, setSelectedHistory] = useState<EvalHistoryEntry | null>(null)

  useEffect(() => {
    if (propAsset) {
      setAsset(propAsset)
      setLoading(false)
      if (propAsset.eval_history && propAsset.eval_history.length > 0) {
        setSelectedHistory(propAsset.eval_history[0])
      }
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
      // Auto-select first history entry
      if (data.eval_history && data.eval_history.length > 0) {
        setSelectedHistory(data.eval_history[0])
      }
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
    return <div>{t('content_detail_asset_not_found')}</div>
  }

  const sortedHistory = [...(asset.eval_history || [])].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )

  return (
    <div>
      <Card
        title={asset.name}
        extra={
          <Space>
            <Tag>{asset.category || 'content'}</Tag>
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
            label: t('content_detail_tab_overview'),
            children: (
              <Card>
                <Row gutter={16}>
                  <Col span={8}>
                    <Statistic title={t('content_detail_latest_score')} value={asset.latest_score ?? 0} precision={2} suffix="/ 1.0" />
                  </Col>
                  <Col span={8}>
                    <Statistic title={t('content_detail_total_evals')} value={asset.eval_history?.length ?? 0} />
                  </Col>
                  <Col span={8}>
                    <Statistic title={t('content_detail_triggers')} value={asset.triggers?.length ?? 0} />
                  </Col>
                </Row>
                {asset.recommended_snapshot_id && (
                  <div style={{ marginTop: 16 }}>
                    <Tag color="blue">{t('content_detail_recommended_snapshot')}: {asset.recommended_snapshot_id}</Tag>
                  </div>
                )}
              </Card>
            ),
          },
          {
            key: 'editor',
            label: t('content_detail_tab_editor'),
            children: (
              <Card
                extra={
                  <Button icon={<EditOutlined />} onClick={() => navigate(`/assets/${id}/edit`)}>
                    {t('content_detail_edit')}
                  </Button>
                }
              >
                <div style={{ padding: 16, background: '#f5f5f5', borderRadius: 4, minHeight: 200 }}>
                  <p style={{ color: '#888' }}>{t('content_detail_click_edit')}</p>
                </div>
              </Card>
            ),
          },
          {
            key: 'versions',
            label: t('content_detail_tab_versions'),
            children: (
              <Card
                extra={
                  <Button icon={<HistoryOutlined />} onClick={() => navigate(`/assets/${id}/versions`)}>
                    {t('content_detail_view_full_history')}
                  </Button>
                }
              >
                <Timeline
                  items={sortedHistory.length > 0
                    ? sortedHistory.slice(0, 5).map((entry) => ({
                        color: entry.status === 'completed' ? 'green' : 'gray',
                        children: (
                          <Card size="small">
                            <Space>
                              <Tag>{entry.run_id.slice(-8)}</Tag>
                              {entry.score !== undefined && (
                                <Tag color={entry.score >= 0.8 ? 'green' : entry.score >= 0.6 ? 'orange' : 'red'}>
                                  {(entry.score * 100).toFixed(0)}%
                                </Tag>
                              )}
                              <span style={{ fontSize: 12, color: '#888' }}>
                                {entry.created_at ? new Date(entry.created_at).toLocaleString() : ''}
                              </span>
                            </Space>
                          </Card>
                        ),
                      }))
                    : [{ color: 'gray', children: <span style={{ color: '#888' }}>{t('content_detail_no_version_history')}</span> }]
                  }
                />
              </Card>
            ),
          },
          {
            key: 'eval_history',
            label: t('content_detail_tab_eval_history'),
            children: (
              <div style={{ display: 'flex', gap: 16 }}>
                {/* Left: Timeline */}
                <Card title={t('content_detail_timeline')} style={{ width: 280, flexShrink: 0 }}>
                  <Timeline
                    items={sortedHistory.map((entry) => ({
                      color: entry.status === 'completed' ? 'green' : entry.status === 'failed' ? 'red' : 'blue',
                      children: (
                        <div
                          onClick={() => setSelectedHistory(entry)}
                          style={{
                            cursor: 'pointer',
                            padding: 4,
                            borderRadius: 4,
                            background: selectedHistory?.run_id === entry.run_id ? '#e6f7ff' : 'transparent',
                          }}
                        >
                          <Space direction="vertical" size="small">
                            <Space>
                              {entry.status === 'completed' ? (
                                <CheckCircleOutlined style={{ color: '#52c41a' }} />
                              ) : entry.status === 'failed' ? (
                                <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                              ) : null}
                              <span style={{ fontSize: 11 }}>{entry.run_id.slice(-8)}</span>
                            </Space>
                            <div style={{ fontSize: 11, color: '#888' }}>
                              {entry.created_at ? new Date(entry.created_at).toLocaleString() : ''}
                            </div>
                            {entry.score !== undefined && (
                              <Tag color={entry.score >= 0.8 ? 'green' : 'orange'}>
                                {(entry.score * 100).toFixed(0)}%
                              </Tag>
                            )}
                          </Space>
                        </div>
                      ),
                    }))}
                  />
                </Card>

                {/* Right: Detail Card */}
                <Card title={t('content_detail_execution_detail')} style={{ flex: 1 }}>
                  {selectedHistory ? (
                    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                      <Row gutter={16}>
                        <Col span={8}>
                          <Statistic title={t('content_detail_status')} value={selectedHistory.status} />
                        </Col>
                        <Col span={8}>
                          <Statistic title={t('content_detail_model')} value={selectedHistory.model || 'N/A'} />
                        </Col>
                        <Col span={8}>
                          <Statistic
                            title={t('content_detail_deterministic_score')}
                            value={selectedHistory.deterministic_score ?? 0}
                            precision={2}
                          />
                        </Col>
                      </Row>
                      <Row gutter={16}>
                        <Col span={8}>
                          <Statistic
                            title={t('content_detail_rubric_score')}
                            value={selectedHistory.rubric_score ?? 0}
                            precision={2}
                          />
                        </Col>
                        <Col span={8}>
                          <Statistic title={t('content_detail_tokens_in')} value={selectedHistory.tokens_in ?? 0} />
                        </Col>
                        <Col span={8}>
                          <Statistic title={t('content_detail_tokens_out')} value={selectedHistory.tokens_out ?? 0} />
                        </Col>
                      </Row>
                      <div>
                        <span style={{ color: '#888' }}>{t('content_detail_latency')}: </span>
                        <span>{selectedHistory.latency_ms ? `${selectedHistory.latency_ms}ms` : 'N/A'}</span>
                      </div>
                      {selectedHistory.commit_hash && (
                        <div>
                          <span style={{ color: '#888' }}>{t('content_detail_snapshot')}: </span>
                          <code>{selectedHistory.commit_hash.slice(0, 8)}</code>
                        </div>
                      )}
                      {selectedHistory.author && (
                        <div>
                          <span style={{ color: '#888' }}>{t('content_detail_by')}: </span>
                          <span>{selectedHistory.author}</span>
                        </div>
                      )}
                      <Button
                        type="primary"
                        icon={<PlayCircleOutlined />}
                        onClick={() => navigate(`/assets/${id}/eval`)}
                      >
                        {t('content_detail_run_eval')}
                      </Button>
                    </Space>
                  ) : (
                    <div style={{ color: '#888', textAlign: 'center', padding: 40 }}>
                      {t('content_detail_select_execution')}
                    </div>
                  )}
                </Card>
              </div>
            ),
          },
        ]}
      />
    </div>
  )
}

export default ContentDetailView
