import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Timeline, Tag, Button, Space, Spin, message } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, SyncOutlined } from '@ant-design/icons'
import { assetApi } from '../api/client'
import type { AssetDetail } from '../api/client'

function VersionTreeView() {
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
    return <div>Asset not found</div>
  }

  const sortedSnapshots = [...(asset.snapshots || [])].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )

  return (
    <div>
      <Card
        title={`Version History: ${asset.name}`}
        extra={
          <Space>
            <Button type="primary" onClick={() => navigate(`/assets/${id}/eval`)}>
              Run Eval
            </Button>
            <Button onClick={() => navigate(`/assets/${id}/edit`)}>Edit</Button>
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
                    <Tag>{snapshot.commit_hash.slice(0, 8)}</Tag>
                    {snapshot.eval_score !== undefined && (
                      <Tag icon={snapshot.eval_score >= 0.8 ? <CheckCircleOutlined /> : <CloseCircleOutlined />}>
                        Score: {(snapshot.eval_score * 100).toFixed(1)}%
                      </Tag>
                    )}
                  </Space>
                  <div>
                    <strong>Author:</strong> {snapshot.author}
                  </div>
                  <div>
                    <strong>Reason:</strong> {snapshot.reason}
                  </div>
                  <div style={{ color: '#888', fontSize: 12 }}>
                    {new Date(snapshot.created_at).toLocaleString()}
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
