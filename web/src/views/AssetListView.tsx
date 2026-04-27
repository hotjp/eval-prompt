import { useEffect, useState, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Card, Tag, Input, Button, Space, message, Spin, Dropdown, Modal } from 'antd'
import type { MenuProps } from 'antd'
import { PlusOutlined, ReloadOutlined, EditOutlined, HistoryOutlined, CheckCircleOutlined, MoreOutlined, RollbackOutlined, DeleteOutlined, InboxOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { assetApi, adminApi } from '../api/client'
import type { AssetSummary } from '../api/client'
import { useStore } from '../store'
import { getAssetTypes } from '../config/assetTypes'
import { getTagColor } from '../config/tags'
import { categoryLabels, categoryColors } from '../config/categories'

const { Search } = Input

// Derive unique tags from loaded assets (excluding archived)
const getAllTags = (assets: AssetSummary[]) => {
  const tagSet = new Set<string>()
  assets.filter(a => a.state !== 'archived').forEach((a) => a.tags?.forEach((t) => tagSet.add(t)))
  return Array.from(tagSet).sort()
}

type ViewMode = 'active' | 'archived'

function AssetListView() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [assets, setAssets] = useState<AssetSummary[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedAssetType, setSelectedAssetType] = useState<string>('all')
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [searchText, setSearchText] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('active')
  const setShowInitRepoModal = useStore(s => s.setShowInitRepoModal)

  const selectedCategory = searchParams.get('category') || ''

  const initRef = useRef(false)

  useEffect(() => {
    if (initRef.current) return
    initRef.current = true
    loadAssets()
  }, [])

  const loadAssets = async () => {
    setLoading(true)
    try {
      const data = await assetApi.list({ category: selectedCategory || undefined })
      setAssets(data.assets)
    } catch {
      message.error(t('asset_list_load_failed'))
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
      message.success(t('asset_archive_success'))
      loadAssets()
    } catch {
      message.error(t('asset_archive_failed'))
    }
  }

  const handleRestore = async (id: string) => {
    try {
      await assetApi.restore(id)
      message.success(t('asset_restore_success'))
      loadAssets()
    } catch {
      message.error(t('asset_restore_failed'))
    }
  }

  const handleMarkDraft = async (id: string) => {
    try {
      await assetApi.update(id, { state: 'draft' })
      message.success(t('asset_mark_draft_success'))
      loadAssets()
    } catch {
      message.error(t('asset_mark_draft_failed'))
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await assetApi.delete(id)
      message.success(t('asset_delete_success'))
      loadAssets()
    } catch {
      message.error(t('asset_delete_failed'))
    }
  }

  const getMoreMenuItems = (asset: AssetSummary): MenuProps['items'] => {
    if (asset.state === 'archived') {
      return [
        {
          key: 'restore',
          label: t('asset_menu_restore'),
          icon: <RollbackOutlined />,
          onClick: () => handleRestore(asset.id),
        },
        {
          key: 'markDraft',
          label: t('asset_menu_mark_draft'),
          icon: <EditOutlined />,
          onClick: () => handleMarkDraft(asset.id),
        },
        {
          key: 'delete',
          label: t('asset_menu_delete'),
          danger: true,
          icon: <DeleteOutlined />,
          onClick: () => {
            Modal.confirm({
              title: t('asset_menu_confirm_delete_title'),
              content: t('asset_menu_confirm_delete_content'),
              okText: t('asset_menu_confirm_delete_ok'),
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
    const matchCategory = !selectedCategory || asset.category === selectedCategory
    const matchSearch =
      !searchText ||
      asset.name.toLowerCase().includes(searchText.toLowerCase()) ||
      asset.description?.toLowerCase().includes(searchText.toLowerCase())
    return matchBiz && matchTags && matchSearch && matchCategory
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
          placeholder={t('asset_list_search')}
          onSearch={setSearchText}
          style={{ marginBottom: 16 }}
          allowClear
        />

        <div style={{ marginBottom: 16 }}>
          <div style={{ fontSize: 12, color: '#888', marginBottom: 8, fontWeight: 600 }}>{t('asset_list_status')}</div>
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
            <span>{t('asset_list_active')}</span>
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
            <span><InboxOutlined style={{ marginRight: 6 }} />{t('asset_list_archived')}</span>
            <span style={{ fontSize: 12, color: '#888' }}>{archivedAssets.length}</span>
          </div>
        </div>

        {viewMode === 'active' && (
          <>
            <div style={{ marginBottom: 24 }}>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 8, fontWeight: 600 }}>{t('asset_list_biz_line')}</div>
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
                  <span>{biz === 'all' ? t('asset_list_all_assets') : biz}</span>
                  <span style={{ fontSize: 12, color: '#888' }}>{getCountByAssetType(biz)}</span>
                </div>
              ))}
            </div>

            <div>
              <div style={{ fontSize: 12, color: '#888', marginBottom: 8, fontWeight: 600 }}>{t('asset_list_tags')}</div>
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
          <Button icon={<ReloadOutlined />} onClick={loadAssets}>{t('asset_list_refresh')}</Button>
          {viewMode === 'active' && (
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              {t('asset_list_create')}
            </Button>
          )}
          <span style={{ color: '#888', fontSize: 12 }}>
            {filteredAssets.length} {filteredAssets.length !== 1 ? t('asset_list_assets') : t('asset_list_asset')}
          </span>
        </Space>

        {filteredAssets.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 60, color: '#888' }}>
            {viewMode === 'archived' ? t('asset_list_no_archived') : t('asset_list_no_assets')}
          </div>
        ) : (
          <div style={{ display: 'flow-root', clear: 'both' }}>
            {filteredAssets.map((asset) => (
              <div key={asset.id} style={{ float: 'left', width: 280, marginRight: 16, marginBottom: 16 }}>
                <Card
                  headStyle={{ padding: '8px 12px', minHeight: 40 }}
                  bodyStyle={{ padding: 12, position: 'relative' }}
                  title={
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Tag color={categoryColors[asset.category || 'content'] || 'default'} style={{ margin: 0 }}>
                        {categoryLabels[asset.category || 'content'] || asset.category}
                      </Tag>
                      <span
                        style={{ fontWeight: 600, fontSize: 14, cursor: 'pointer', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                        onClick={(e) => { e.stopPropagation(); navigate(`/assets/${asset.id}/edit`) }}
                      >
                        {asset.name}
                      </span>
                    </div>
                  }
                  actions={[
                    viewMode === 'active' && (
                      <Button key="edit" type="text" size="small" icon={<EditOutlined />} onClick={() => navigate(`/assets/${asset.id}/edit`)}>
                        {t('asset_card_edit')}
                      </Button>
                    ),
                    viewMode === 'active' && (
                      <Button key="versions" type="text" size="small" icon={<HistoryOutlined />} onClick={() => navigate(`/assets/${asset.id}/versions`)}>
                        {t('asset_card_history')}
                      </Button>
                    ),
                    viewMode === 'active' && (
                      <Button key="eval" type="text" size="small" icon={<CheckCircleOutlined />} onClick={() => navigate(`/assets/${asset.id}/eval`)}>
                        {t('asset_card_eval')}
                      </Button>
                    ),
                  ].filter(Boolean)}
                  extra={
                    <Dropdown menu={{ items: getMoreMenuItems(asset) }} trigger={['click']}>
                      <Button type="text" size="small" icon={<MoreOutlined />} />
                    </Dropdown>
                  }
                >
                  {/* Tags below title */}
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginBottom: 8 }}>
                    {asset.tags?.map((tag) => (
                      <Tag key={tag} color={getTagColor(tag)} style={{ margin: 0 }}>
                        {tag}
                      </Tag>
                    ))}
                  </div>
                  <div style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>
                    {asset.description?.slice(0, 50) || t('asset_card_no_description')}
                    {(asset.description?.length || 0) > 50 ? '...' : ''}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    {asset.category === 'content' && (
                      asset.latest_score !== undefined && asset.latest_score !== null ? (
                        <span style={{ fontSize: 11, color: '#888' }}>
                          {t('asset_card_latest')}: {(asset.latest_score * 100).toFixed(0)}%
                        </span>
                      ) : (
                        <span style={{ fontSize: 11, color: '#bbb' }}>{t('asset_card_no_eval')}</span>
                      )
                    )}
                    {asset.category === 'eval' && (
                      <span style={{ fontSize: 11, color: '#888' }}>
                        {asset.test_cases?.length || 0} {t('asset_card_test_cases')}
                      </span>
                    )}
                    {asset.category === 'metric' && (
                      <span style={{ fontSize: 11, color: '#888' }}>
                        {asset.rubric?.length || 0} {t('asset_card_rubric_checks')}
                      </span>
                    )}
                    {(!asset.category || asset.category === 'content') && asset.latest_score !== undefined && asset.latest_score !== null && (
                      <Tag
                        color={
                          asset.latest_score >= 0.8 ? 'green' :
                          asset.latest_score >= 0.6 ? 'orange' : 'red'
                        }
                        style={{ margin: 0 }}
                      >
                        {(asset.latest_score * 100).toFixed(0)}%
                      </Tag>
                    )}
                    <Tag color={asset.state === 'active' ? 'green' : asset.state === 'draft' ? 'orange' : 'default'}>
                      {asset.state}
                    </Tag>
                  </div>
                </Card>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default AssetListView
