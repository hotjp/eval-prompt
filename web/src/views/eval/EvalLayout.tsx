import { useEffect, useState } from 'react'
import { useParams, useNavigate, useLocation, Outlet } from 'react-router-dom'
import { Card, Tag, Button, Space, Spin, message, Tabs } from 'antd'
import { useTranslation } from 'react-i18next'
import { assetApi } from '../../api/client'
import type { AssetDetail } from '../../api/client'

const tabItems = [
  { key: 'design', label: 'Design' },
  { key: 'run', label: 'Run' },
  { key: 'history', label: 'History' },
]

function EvalLayout() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)

  const activeTab = location.pathname.split('/').pop() || 'run'

  useEffect(() => {
    if (id) loadAsset(id)
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
    return <div>{t('eval_panel_asset_not_found')}</div>
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Prompt Info Card */}
      <Card
        size="small"
        title={asset.name}
        extra={
          <Space>
            <Tag>{asset.category || 'content'}</Tag>
            <Tag color={asset.state === 'active' ? 'green' : 'orange'}>{asset.state}</Tag>
            <Button size="small" onClick={() => navigate(`/assets/${id}/edit-v2`)}>
              {t('editor_v2_version_tree')}
            </Button>
          </Space>
        }
      >
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {asset.tags?.map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </div>
      </Card>

      {/* Stage Navigation */}
      <Tabs
        activeKey={activeTab}
        onChange={(key) => navigate(`/assets/${id}/eval/${key}`)}
        items={tabItems}
      />

      {/* Content */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        <Outlet />
      </div>
    </div>
  )
}

export default EvalLayout
