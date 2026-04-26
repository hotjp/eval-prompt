import { useEffect, useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Tag, Input, Button, Space, message, Spin, Row, Col, Dropdown, Modal } from 'antd'
import type { MenuProps } from 'antd'
import { PlusOutlined, ReloadOutlined, EditOutlined, HistoryOutlined, CheckCircleOutlined, MoreOutlined, RollbackOutlined, DeleteOutlined, InboxOutlined } from '@ant-design/icons'
import { assetApi, adminApi } from '../api/client'
import type { AssetSummary } from '../api/client'
import { useStore } from '../store'
import { getAssetTypes, getAssetTypeColor } from '../config/bizLines'
import { getTagColor } from '../config/tags'

const { Search } = Input

// Derive unique tags from loaded assets (excluding archived)
const getAllTags = (assets: AssetSummary[]) => {
  const tagSet = new Set<string>()
  assets.filter(a => a.state !== 'archived').forEach((a) => a.tags?.forEach((t) => tagSet.add(t)))
  return Array.from(tagSet).sort()
}

type ViewMode = 'active' | 'archived'

function AssetListView() {
  const navigate = useNavigate()
  const [assets, setAssets] = useState<AssetSummary[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedAssetType, setSelectedAssetType] = useState<string>('all')
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [searchText, setSearchText] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('active')
  const setShowInitRepoModal = useStore(s => s.setShowInitRepoModal)

  const initRef = useRef(false)

  useEffect(() => {
    if (initRef.current) return
    initRef.current = true
    loadAssets()
  }, [])

  const loadAssets = async () => {
    setLoading(true)
    try {
      const data = await assetApi.list()
      setAssets(data.assets)
    } catch {
      message.error('Failed to load assets')
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async () => {
    try {
      const status = await adminApi.getRepoStatus()
      if (!status.current?.valid) {
        setShowInitRepoModal(true, 'create')
        return
      }
      navigate('/assets/new')
    } catch {
      // API unavailable — treat as no valid repo, show init modal
      setShowInitRepoModal(true, 'api_error')
    }
  }

  const handleArchive = async (id: string) => {
    try {
      await assetApi.archive(id)
      message.success('Asset archived')
      loadAssets()
    } catch {
      message.error('Failed to archive asset')
    }
  }

  const handleRestore = async (id: string) => {
    try {
      await assetApi.restore(id)
      message.success('Asset restored')
      loadAssets()
    } catch {
      message.error('Failed to restore asset')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await assetApi.delete(id)
      message.success('Asset deleted')
      loadAssets()
    } catch {
      message.error('Failed to delete asset')
    }
  }

  const getMoreMenuItems = (asset: AssetSummary): MenuProps['items'] => {
    if (asset.state === 'archived') {
      return [
        {
          key: 'restore',
          label: 'Restore',
          icon: <RollbackOutlined />,
          onClick: () => handleRestore(asset.id),
        },
        {
          key: 'delete',
          label: 'Delete permanently',
          danger: true,
          icon: <DeleteOutlined />,
          onClick: () => {
            Modal.confirm({
              title: 'Delete asset?',
              content: 'This action cannot be undone.',
              okText: 'Delete',
              okType: 'danger',
              onOk: () => handleDelete(asset.id),
            })
          },
        },
      ]
    }
    return [
      {
        key: 'archive',
        label: 'Archive',
        danger: true,
        onClick: () => handleArchive(asset.id),
      },
    ]
  }

  const toggleTag = (tag: string) => {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag]
    )
  }

  const activeAssets = assets.filter(a => a.state !== 'archived')
  const archivedAssets = assets.filter(a => a.state === 'archived')

  const filteredAssets = (viewMode === 'archived' ? archivedAssets : activeAssets).filter((asset) => {
    const matchBiz = selectedAssetType === 'all' || asset.asset_type === selectedAssetType
    const matchTags = selectedTags.length === 0 || selectedTags.every((t) => asset.tags?.includes(t))
    const matchSearch =
      !searchText ||
      asset.name.toLowerCase().includes(searchText.toLowerCase()) ||
      asset.description?.toLowerCase().includes(searchText.toLowerCase())
    return matchBiz && matchTags && matchSearch
  })

  const getCountByAssetType = (biz: string) =>
    biz === 'all' ? activeAssets.length : activeAssets.filter((a) => a.asset_type === biz).length

  const getCountByTag = (tag: string) => activeAssets.filter((a) => a.tags?.includes(tag)).length

  if (loading && assets.length === 0) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }}>
        <Spin size="large" />
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', gap: 0, minHeight: 'calc(100vh - 100px)' }}>
      {/* Left Sidebar */}
      <div style={{ width: 200, borderRight: '1px solid #f0f0f0', paddingRight: 16, marginRight: 24 }}>
        <Search
          placeholder="Search..."
          onSearch={setSearchText}
          style={{ marginBottom: 16 }}
          allowClear
        />

        <div style={{ marginBottom: 16 }}>
          <div style={{ fontSize: 12, color: '#888', marginBottom: 8, fontWeight: 600 }}>STATUS</div>
          <div
            onClick={() => setViewMode('active')}
            style={{
              padding: '6px 8px',
              cursor: 'pointer',
              borderRadius: 4,
              background: viewMode === 'active' ? '#e6f7ff' : 'transparent',
              color: viewMode === 'active' ? '#1890ff' : '#333',
              fontWeight: viewMode === 'active' ? 600 : 400,
              display: 'flex',
              justifyContent: 'space-between',
            }}
          >
            <span>Active</span>
            <span style={{ fontSize: 12, color: '#888' }}>{activeAssets.length}</span>
          </div>
          <div
            onClick={() => setViewMode('archived')}
            style={{
              padding: '6px 8px',
              cursor: 'pointer',
              borderRadius: 4,
              background: viewMode === 'archived' ? '#fff7e6' : 'transparent',
              color: viewMode === 'archived' ? '#fa8c16' : '#333',
              fontWeight: viewMode === 'archived' ? 600 : 400,
              display: 'flex',
              justifyContent: 'space-between',
            }}
          >
            <span><InboxOutlined style={{ marginRight: 6 }} />Archived</span>
            <span style={{ fontSize: 12, color: '#888' }}>{archivedAssets.length}</span>
          </div>
        </div>

        {viewMode === 'active' && (
          <>
            <div style={{ marginBottom: 24 }}>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 8, fontWeight: 600 }}>BIZ LINE</div>
              {['all', ...getAssetTypes().map((b) => b.name)].map((biz) => (
                <div
                  key={biz}
                  onClick={() => setSelectedAssetType(biz)}
                  style={{
                    padding: '6px 8px',
                    cursor: 'pointer',
                    borderRadius: 4,
                    background: selectedAssetType === biz ? '#e6f7ff' : 'transparent',
                    color: selectedAssetType === biz ? '#1890ff' : '#333',
                    fontWeight: selectedAssetType === biz ? 600 : 400,
                    display: 'flex',
                    justifyContent: 'space-between',
                  }}
                >
                  <span>{biz === 'all' ? 'All Assets' : biz}</span>
                  <span style={{ fontSize: 12, color: '#888' }}>{getCountByAssetType(biz)}</span>
                </div>
              ))}
            </div>

            <div>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 8, fontWeight: 600 }}>TAGS</div>
              {getAllTags(activeAssets).map((tag) => (
                <div
                  key={tag}
                  onClick={() => toggleTag(tag)}
                  style={{
                    padding: '6px 8px',
                    cursor: 'pointer',
                    borderRadius: 4,
                    background: selectedTags.includes(tag) ? '#fff7e6' : 'transparent',
                    color: selectedTags.includes(tag) ? '#fa8c16' : '#333',
                    display: 'flex',
                    justifyContent: 'space-between',
                  }}
                >
                  <span>{tag}</span>
                  <span style={{ fontSize: 12, color: '#888' }}>{getCountByTag(tag)}</span>
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {/* Right Content - Card Grid */}
      <div style={{ flex: 1 }}>
        <Space style={{ marginBottom: 16 }}>
          <Button icon={<ReloadOutlined />} onClick={loadAssets}>Refresh</Button>
          {viewMode === 'active' && (
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              Create
            </Button>
          )}
          <span style={{ color: '#888', fontSize: 12 }}>
            {filteredAssets.length} asset{filteredAssets.length !== 1 ? 's' : ''}
          </span>
        </Space>

        {filteredAssets.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 60, color: '#888' }}>
            {viewMode === 'archived' ? 'No archived assets' : 'No assets found'}
          </div>
        ) : (
          <Row gutter={[16, 16]}>
            {filteredAssets.map((asset) => (
              <Col key={asset.id} xs={24} sm={12} md={8} lg={6}>
                <Card
                  bodyStyle={{ padding: 16, position: 'relative' }}
                  actions={[
                    viewMode === 'active' && (
                      <Button key="edit" type="text" size="small" icon={<EditOutlined />} onClick={() => navigate(`/assets/${asset.id}/edit`)}>
                        Edit
                      </Button>
                    ),
                    viewMode === 'active' && (
                      <Button key="versions" type="text" size="small" icon={<HistoryOutlined />} onClick={() => navigate(`/assets/${asset.id}/versions`)}>
                        History
                      </Button>
                    ),
                    viewMode === 'active' && (
                      <Button key="eval" type="text" size="small" icon={<CheckCircleOutlined />} onClick={() => navigate(`/assets/${asset.id}/eval`)}>
                        Eval
                      </Button>
                    ),
                  ].filter(Boolean)}
                  extra={
                    <Dropdown menu={{ items: getMoreMenuItems(asset) }} trigger={['click']}>
                      <Button type="text" size="small" icon={<MoreOutlined />} />
                    </Dropdown>
                  }
                >
                  {/* Biz Line corner badge */}
                  <div style={{ position: 'absolute', top: 8, left: 8 }}>
                    <Tag color={getAssetTypeColor(asset.asset_type)} style={{ margin: 0 }}>
                      {asset.asset_type}
                    </Tag>
                  </div>
                  <div style={{ fontWeight: 600, fontSize: 14, cursor: 'pointer', paddingTop: 20, marginBottom: 6 }} onClick={() => navigate(`/assets/${asset.id}/edit`)}>
                    {asset.name}
                  </div>
                  {/* Tags below title */}
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginBottom: 8 }}>
                    {asset.tags?.map((tag) => (
                      <Tag key={tag} color={getTagColor(tag)} style={{ margin: 0 }}>
                        {tag}
                      </Tag>
                    ))}
                  </div>
                  <div style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>
                    {asset.description?.slice(0, 50) || 'No description'}
                    {(asset.description?.length || 0) > 50 ? '...' : ''}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    {asset.latest_score !== undefined && asset.latest_score !== null ? (
                      <Tag
                        color={
                          asset.latest_score >= 0.8 ? 'green' :
                          asset.latest_score >= 0.6 ? 'orange' : 'red'
                        }
                        style={{ margin: 0 }}
                      >
                        {(asset.latest_score * 100).toFixed(0)}%
                      </Tag>
                    ) : (
                      <span style={{ fontSize: 11, color: '#bbb' }}>No eval</span>
                    )}
                    <Tag color={asset.state === 'active' ? 'green' : asset.state === 'draft' ? 'orange' : 'default'}>
                      {asset.state}
                    </Tag>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        )}
      </div>
    </div>
  )
}

export default AssetListView
