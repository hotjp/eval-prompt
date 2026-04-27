import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Timeline, Tag, Button, Space, Spin, message } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, SyncOutlined } from '@ant-design/icons'
import { assetApi } from '../api/client'
import type { AssetDetail } from '../api/client'
import { useTranslation } from 'react-i18next'

function VersionTreeView() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)

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
    return <div>{t('version_tree_asset_not_found')}</div>
  }

  const sortedSnapshots = [...(asset.snapshots || [])].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )

  return (
    <div>
      <Card
        title={`${t('version_tree_title')}: ${asset.name}`}
        extra={
          <Space>
            <Button type="primary" onClick={() => navigate(`/assets/${id}/eval`)}>
              {t('version_tree_run_eval')}
            </Button>
            <Button onClick={() => navigate(`/assets/${id}/edit`)}>{t('version_tree_edit')}</Button>
          </Space>
        }
      >
        <Timeline
          items={sortedSnapshots.map((snapshot) => ({
            color: snapshot.eval_score !== undefined && snapshot.eval_score >= 0.8
              ? 'green'
              : snapshot.eval_score !== undefined
              ? 'red'
              : 'blue',
            children: (
              <Card size="small" style={{ marginBottom: 8 }}>
                <Space direction="vertical" size="small" style={{ width: '100%' }}>
                  <Space>
                    <Tag icon={<SyncOutlined spin />}>{snapshot.version}</Tag>
                    <Tag>{(snapshot.commit_hash || '').slice(0, 8)}</Tag>
                    {snapshot.eval_score !== undefined && (
                      <Tag icon={snapshot.eval_score >= 0.8 ? <CheckCircleOutlined /> : <CloseCircleOutlined />}>
                        {t('version_tree_score')}: {(snapshot.eval_score * 100).toFixed(1)}%
                      </Tag>
                    )}
                  </Space>
                  <div>
                    <strong>{t('version_tree_author')}:</strong> {snapshot.author || t('version_tree_unknown')}
                  </div>
                  <div>
                    <strong>{t('version_tree_reason')}:</strong> {snapshot.reason || t('version_tree_no_reason')}
                  </div>
                  <div style={{ color: '#888', fontSize: 12 }}>
                    {snapshot.created_at ? new Date(snapshot.created_at).toLocaleString() : t('version_tree_unknown')}
                  </div>
                </Space>
              </Card>
            ),
          }))}
        />
      </Card>
    </div>
  )
}

export default VersionTreeView
