import { useEffect, useState, useRef } from 'react'
import { Layout, Menu, Popover, Button, Space, Typography, Popconfirm, message, Dropdown, Input, Modal, Select } from 'antd'
import { LoadingOutlined } from '@ant-design/icons'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  AppstoreOutlined,
  SwapOutlined,
  SettingOutlined,
  ReloadOutlined,
  PoweroffOutlined,
  BranchesOutlined,
  SyncOutlined,
  FieldTimeOutlined,
  FolderOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { healthApi, adminApi, assetApi, type HealthStatus } from '../api/client'
import { useStore } from '../store'

interface RepoStatus {
  path?: string
  valid?: boolean
  branch?: string
  dirty?: boolean
  short_commit?: string
  error?: string
  outside_home?: boolean
}

interface RepoEntry {
  path: string
  status: string
}

const { Header } = Layout
const { Text } = Typography

type Status = 'ok' | 'error' | 'loading' | 'degraded'

function Sidebar() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [serverStatus, setServerStatus] = useState<Status>('loading')
  const [healthData, setHealthData] = useState<HealthStatus | null>(null)
  const [repoStatus, setRepoStatus] = useState<{ current?: RepoStatus; repos: RepoEntry[]; is_first_use: boolean } | null>(null)
  const [assetCount, setAssetCount] = useState(0)
  const [categoryCounts, setCategoryCounts] = useState({ content: 0, eval: 0, metric: 0 })
  const [loading, setLoading] = useState(false)
  const [statusOpen, setStatusOpen] = useState(false)
  const [lastChecked, setLastChecked] = useState<Date | null>(null)
  const [initPath, setInitPath] = useState('')
  const [initLoading, setInitLoading] = useState(false)
  const [gitSyncing, setGitSyncing] = useState(false)
  const runningEval = useStore(s => s.runningEval)
  const showInitRepoModal = useStore(s => s.showInitRepoModal)
  const initRepoModalReason = useStore(s => s.initRepoModalReason)
  const setShowInitRepoModal = useStore(s => s.setShowInitRepoModal)

  useEffect(() => {
    if (document.getElementById('sidebar-animations')) return
    const s = document.createElement('style')
    s.id = 'sidebar-animations'
    s.textContent = `
      @keyframes pulse {
        0%, 100% { opacity: 1; transform: scale(1); }
        50% { opacity: 0.7; transform: scale(1.15); }
      }
    `
    document.head.appendChild(s)
  }, [])

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(key)
  }

  const handleStatusItemClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    setStatusOpen(o => !o)
  }

  const fetchHealth = async () => {
    try {
      const res = await healthApi.check()
      setHealthData(res)
      setLastChecked(new Date())
      setServerStatus(res.status === 'ok' ? 'ok' : res.status === 'degraded' ? 'degraded' : 'error')
    } catch {
      setServerStatus('error')
      setHealthData(null)
    }
  }

  const fetchRepoStatus = async () => {
    try {
      const res = await adminApi.getRepoStatus()
      setRepoStatus(res)
    } catch {
      setRepoStatus(null)
    }
  }

  const fetchAssetCount = async () => {
    try {
      const res = await assetApi.list()
      setAssetCount(res.total)
      // Compute category counts
      const counts = { content: 0, eval: 0, metric: 0 }
      res.assets.forEach(a => {
        if (a.category === 'content') counts.content++
        else if (a.category === 'eval') counts.eval++
        else if (a.category === 'metric') counts.metric++
      })
      setCategoryCounts(counts)
    } catch {
      setAssetCount(0)
      setCategoryCounts({ content: 0, eval: 0, metric: 0 })
    }
  }

  const initRef = useRef(false)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    if (initRef.current) return
    initRef.current = true

    fetchHealth()
    fetchRepoStatus()
    fetchAssetCount()

    intervalRef.current = setInterval(fetchHealth, 30000)
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
      initRef.current = false
    }
  }, [])

  const handleRestart = async () => {
    if (runningEval) {
      message.warning(t('sidebar_eval_running_warning'))
      return
    }
    setLoading(true)
    try {
      await new Promise(r => setTimeout(r, 1000))
      message.success(t('sidebar_restart_sent'))
      setStatusOpen(false)
    } catch {
      message.error(t('sidebar_restart_failed'))
    } finally {
      setLoading(false)
    }
  }

  const handleReloadConfig = async () => {
    setLoading(true)
    try {
      await new Promise(r => setTimeout(r, 800))
      message.success(t('sidebar_config_reloaded'))
      setStatusOpen(false)
    } catch {
      message.error(t('sidebar_config_reload_failed'))
    } finally {
      setLoading(false)
    }
  }

  const handleGoToLLMSettings = () => {
    setStatusOpen(false)
    navigate('/settings?section=llm')
  }

  const handleCommitAll = async () => {
    setGitSyncing(true)
    try {
      // Get all asset IDs and commit them
      const res = await assetApi.list()
      const ids = res.assets.map(a => a.id)
      if (ids.length === 0) {
        message.warning(t('sidebar_no_assets_to_commit'))
        return
      }
      const result = await assetApi.commitBatch(ids, 'Batch commit all assets')
      message.success(t('sidebar_committed_count', { count: Object.keys(result.commits).length }))
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('sidebar_commit_failed'))
    } finally {
      setGitSyncing(false)
    }
  }

  const handleReconcile = async () => {
    setGitSyncing(true)
    try {
      const report = await adminApi.reconcile()
      message.success(t('sidebar_sync_complete', { added: report.added, updated: report.updated, deleted: report.deleted }))
      fetchRepoStatus()
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('sidebar_sync_failed'))
    } finally {
      setGitSyncing(false)
    }
  }

  const handleGitPull = async () => {
    setGitSyncing(true)
    try {
      await adminApi.gitPull()
      message.success(t('sidebar_git_pull_success'))
      fetchRepoStatus()
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('sidebar_git_pull_failed'))
    } finally {
      setGitSyncing(false)
    }
  }

  const handleSwitchRepo = async (path: string) => {
    setGitSyncing(true)
    try {
      await adminApi.switchRepo(path)
      message.success(t('sidebar_repo_switched'))
      fetchRepoStatus()
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('sidebar_switch_repo_failed'))
    } finally {
      setGitSyncing(false)
    }
  }

  const handleInitRepo = async () => {
    if (!initPath.trim()) return
    setInitLoading(true)
    try {
      await adminApi.switchRepo(initPath.trim())
      message.success(t('sidebar_repo_initialized'))
      setShowInitRepoModal(false)
      setInitPath('')
      fetchRepoStatus()
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('sidebar_init_repo_failed'))
    } finally {
      setInitLoading(false)
    }
  }

  const statusDot = (
    <span
      style={{
        width: 7,
        height: 7,
        borderRadius: '50%',
        background:
          serverStatus === 'ok' ? '#52c41a' :
          serverStatus === 'error' ? '#ff4d4f' :
          serverStatus === 'degraded' ? '#fa8c16' : '#d9d9d9',
        boxShadow:
          serverStatus === 'ok' ? '0 0 6px rgba(82, 196, 26, 0.6)' :
          serverStatus === 'error' ? '0 0 6px rgba(255, 77, 79, 0.6)' :
          serverStatus === 'degraded' ? '0 0 6px rgba(250, 140, 22, 0.5)' :
          'none',
        animation: serverStatus === 'ok' ? 'pulse 2s infinite' : 'none',
        display: 'inline-block',
        flexShrink: 0,
      }}
    />
  )

  const statusLabel = serverStatus === 'loading'
    ? t('sidebar_status_checking')
    : serverStatus === 'ok'
    ? t('sidebar_status_online')
    : serverStatus === 'degraded'
    ? t('sidebar_status_degraded')
    : t('sidebar_status_offline')

  const statusMenuItem = {
    key: '__status__',
    label: (
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }} onClick={handleStatusItemClick}>
        {statusDot}
        {serverStatus === 'loading' && <LoadingOutlined style={{ fontSize: 12, color: '#8c8c8c' }} />}
        <span style={{ fontSize: 13 }}>{statusLabel}</span>
      </div>
    ),
  }

  const currentCategory = searchParams.get('category') || ''

  const handleCategoryChange = (value: string) => {
    if (value) {
      setSearchParams({ category: value })
      navigate('/assets')
    } else {
      setSearchParams({})
      navigate('/assets')
    }
  }

  const navItems = [
    {
      key: '/assets',
      icon: <AppstoreOutlined />,
      label: (
        <Select
          value={currentCategory}
          onChange={handleCategoryChange}
          placeholder={t('sidebar_all_assets')}
          style={{ minWidth: 140 }}
          options={[
            { value: '', label: `${t('sidebar_all_assets')} (${assetCount})` },
            { value: 'content', label: `${t('sidebar_prompts')} (${categoryCounts.content})` },
            { value: 'eval', label: `${t('sidebar_eval_cases')} (${categoryCounts.eval})` },
            { value: 'metric', label: `${t('sidebar_metrics')} (${categoryCounts.metric})` },
          ]}
        />
      ),
    },
    { key: '/compare', icon: <SwapOutlined />, label: t('sidebar_nav_compare') },
    { key: '/settings', icon: <SettingOutlined />, label: t('sidebar_settings') },
  ]

  const repoStatusIcon = (entry: RepoEntry) => {
    if (entry.status === 'valid') {
      return <span style={{ fontSize: 10, color: '#52c41a' }}>✓</span>
    }
    if (entry.status === 'notfound') {
      return <span style={{ fontSize: 10, color: '#ff4d4f' }}>✗</span>
    }
    return <span style={{ fontSize: 10, color: '#fa8c16' }}>⚠</span>
  }

  const popoverContent = (
    <div style={{ width: 260 }}>
      <div style={{ marginBottom: 12 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>{t('sidebar_server_status')}</Text>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 4 }}>
          {statusDot}
          <Text strong style={{ fontSize: 13 }}>
            {serverStatus === 'ok' ? t('sidebar_repo_running') : serverStatus === 'error' ? t('sidebar_status_offline') : serverStatus === 'degraded' ? t('sidebar_status_degraded') : t('sidebar_status_checking')}
          </Text>
        </div>
      </div>

      {runningEval && (
        <div
          style={{
            marginBottom: 12,
            padding: '8px 10px',
            background: '#e6f7ff',
            borderRadius: 6,
            fontSize: 12,
            color: '#595959',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
            <FieldTimeOutlined style={{ color: '#1890ff' }} />
            <span style={{ fontWeight: 500 }}>{t('sidebar_eval_in_progress')}</span>
          </div>
          <div style={{ color: '#8c8c8c' }}>{t('sidebar_eval_warning')}</div>
        </div>
      )}

      {serverStatus === 'degraded' && healthData?.checks && (
        <div
          style={{
            marginBottom: 12,
            padding: '8px 10px',
            background: '#fff7e6',
            borderRadius: 6,
            fontSize: 12,
            color: '#595959',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
            <span style={{ fontSize: 14 }}>⚠️</span>
            <span style={{ fontWeight: 500 }}>{t('sidebar_degraded_services')}</span>
          </div>
          {Object.entries(healthData?.checks || {}).map(([k, v]) => (
            v?.status !== 'ok' && (
              <div key={k} style={{ color: '#8c8c8c', marginTop: 2 }}>
                {k}: {v?.message || v?.status || 'unknown'}
              </div>
            )
          ))}
          {healthData?.checks?.llm?.providers && Object.entries(healthData.checks.llm.providers || {}).map(([name, prov]) => (
            prov?.status !== 'ok' && (
              <div key={`llm-${name}`} style={{ color: '#8c8c8c', marginTop: 2 }}>
                {name}: {prov?.message || prov?.status || 'unknown'}
              </div>
            )
          ))}
        </div>
      )}

      {serverStatus === 'error' && (
        <div
          style={{
            marginBottom: 12,
            padding: '8px 10px',
            background: '#fff2e8',
            borderRadius: 6,
            fontSize: 12,
            color: '#595959',
          }}
        >
          <div style={{ marginBottom: 4, fontWeight: 500 }}>{t('sidebar_server_not_responding')}</div>
          <div>{t('sidebar_start_server_manually')}</div>
          <code style={{ display: 'block', marginTop: 4, padding: '4px 6px', background: '#fff', borderRadius: 4, fontSize: 11 }}>
            ep server
          </code>
        </div>
      )}

      {healthData && healthData?.checks?.llm?.status !== 'ok' && (
        <div style={{ marginBottom: 12 }}>
          <Button
            type="link"
            size="small"
            onClick={handleGoToLLMSettings}
            style={{ padding: 0, fontSize: 12 }}
          >
            {t('sidebar_configure_llm')} →
          </Button>
        </div>
      )}

      {healthData && (
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>{t('sidebar_details')}</Text>
          <div style={{ marginTop: 4, fontSize: 12, color: '#595959' }}>
            <div>{t('sidebar_checked')}: {lastChecked ? lastChecked.toLocaleTimeString() : 'N/A'}</div>
            {healthData?.checks && Object.entries(healthData.checks || {}).map(([k, v]) => (
              <div key={k}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 2 }}>
                  <span>{k}</span>
                  <span style={{ color: v.status === 'ok' ? '#52c41a' : v.status === 'degraded' ? '#fa8c16' : '#ff4d4f' }}>
                    {v.status}
                  </span>
                </div>
                {k === 'llm' && v?.providers && (
                  <div style={{ marginLeft: 12, marginTop: 2 }}>
                    {Object.entries(v.providers || {}).map(([name, prov]) => (
                      <div key={name} style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <span style={{ color: '#8c8c8c', fontSize: 11 }}>{name}</span>
                        <span style={{ color: prov?.status === 'ok' ? '#52c41a' : '#ff4d4f', fontSize: 11 }}>
                          {prov?.status === 'ok' ? `${prov?.latency_ms}ms` : prov?.message}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 12 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>{t('sidebar_actions')}</Text>
        <div style={{ marginTop: 8 }}>
          <Space direction="vertical" style={{ width: '100%' }} size={8}>
            <Button
              icon={<ReloadOutlined />}
              size="small"
              block
              loading={loading}
              disabled={serverStatus === 'error' || !!runningEval}
              onClick={handleReloadConfig}
            >
              {t('sidebar_reload_config')}
            </Button>
            <Popconfirm
              title={t('sidebar_restart_confirm_title')}
              description={t('sidebar_restart_confirm_desc')}
              okText={t('sidebar_restart')}
              cancelText={t('common_cancel')}
              okButtonProps={{ danger: true, loading }}
              onConfirm={handleRestart}
            >
              <Button
                icon={<PoweroffOutlined />}
                size="small"
                block
                danger
                disabled={serverStatus === 'error' || !!runningEval}
              >
                {t('sidebar_restart_server')}
              </Button>
            </Popconfirm>
          </Space>
        </div>
      </div>
    </div>
  )

  const branchLabel = () => {
    if (gitSyncing) {
      return (
        <span style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, color: '#8c8c8c', cursor: 'not-allowed', opacity: 0.6 }}>
          <SyncOutlined spin style={{ fontSize: 11 }} />
          {t('sidebar_syncing')}
        </span>
      )
    }
    return (
      <span style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, color: '#8c8c8c', cursor: 'pointer' }}>
        <BranchesOutlined style={{ fontSize: 11 }} />
        {repoStatus?.current?.branch || t('sidebar_no_branch')}
        {repoStatus?.current?.dirty && (
          <span style={{ fontSize: 8, color: '#fa8c16', lineHeight: 1 }}>●</span>
        )}
      </span>
    )
  }

  return (
    <>
    <Header
      style={{
        background: '#fff',
        padding: '0 24px',
        borderBottom: '1px solid #f0f0f0',
        display: 'flex',
        alignItems: 'center',
      }}
    >
      <Popover
        content={popoverContent}
        title={null}
        trigger="click"
        open={statusOpen}
        onOpenChange={setStatusOpen}
        placement="bottomLeft"
        arrow={false}
        styles={{ body: { padding: '12px 16px' } }}
      >
        <Menu
          mode="horizontal"
          selectedKeys={[]}
          items={[statusMenuItem]}
          style={{ borderBottom: 0, flexShrink: 0, minWidth: 90 }}
        />
      </Popover>

      <Menu
        mode="horizontal"
        selectedKeys={[]}
        items={navItems}
        onClick={handleMenuClick}
        style={{ borderBottom: 0, marginLeft: 0, flex: 1 }}
      />

      <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 6 }}>
        {repoStatus?.current?.valid && (
          <Button
            size="small"
            icon={<BranchesOutlined />}
            onClick={handleCommitAll}
            loading={gitSyncing}
            style={{ fontSize: 12 }}
          >
            {t('sidebar_commit_all')}
          </Button>
        )}
        {repoStatus?.current?.valid ? (
          <Dropdown
            menu={{
              items: [
                {
                  key: 'current',
                  label: <span style={{ fontSize: 11, color: '#8c8c8c' }}>{t('sidebar_current')}: {repoStatus.current?.path}</span>,
                  disabled: true,
                },
                ...(repoStatus.current?.outside_home ? [{
                  key: 'path-warning',
                  label: (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: '#fa8c16', fontSize: 12 }}>
                      <WarningOutlined />
                      <span>{t('sidebar_path_warning')}</span>
                    </div>
                  ),
                  disabled: true,
                }] : []),
                { type: 'divider' as const },
                ...(repoStatus.repos.length > 0 ? [
                  {
                    key: 'switch',
                    label: t('sidebar_switch_repo'),
                    icon: <SwapOutlined />,
                    children: repoStatus.repos.map(r => ({
                      key: `repo-${r.path}`,
                      label: (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <span style={{ fontSize: 12 }}>{r.path.split('/').pop()}</span>
                          {repoStatusIcon(r)}
                        </div>
                      ),
                      onClick: () => handleSwitchRepo(r.path),
                    })),
                  },
                  { type: 'divider' as const },
                ] : []),
                {
                  key: 'init',
                  label: t('sidebar_init_repo'),
                  icon: <FolderOutlined />,
                  onClick: () => setShowInitRepoModal(true, 'manual'),
                },
                { type: 'divider' as const },
                {
                  key: 'reconcile',
                  label: t('sidebar_reconcile'),
                  icon: <SyncOutlined />,
                  onClick: handleReconcile,
                  disabled: gitSyncing,
                },
                {
                  key: 'gitpull',
                  label: t('sidebar_git_pull'),
                  icon: <BranchesOutlined />,
                  onClick: handleGitPull,
                  disabled: gitSyncing,
                },
                {
                  key: 'openfolder',
                  label: t('sidebar_open_finder'),
                  icon: <FolderOutlined />,
                  onClick: async () => {
                    try {
                      await adminApi.openFolder()
                    } catch (e) {
                      message.error(e instanceof Error ? e.message : t('sidebar_open_folder_failed'))
                    }
                  },
                },
                { type: 'divider' as const },
                {
                  key: 'refresh',
                  label: t('sidebar_refresh'),
                  icon: <ReloadOutlined />,
                  onClick: fetchRepoStatus,
                  disabled: gitSyncing,
                },
              ]
            }}
            trigger={['click']}
          >
            {branchLabel()}
          </Dropdown>
        ) : (
          <Dropdown
            menu={{
              items: [
                {
                  key: 'warning',
                  label: (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: '#ff4d4f' }}>
                      <WarningOutlined />
                      <span>{t('sidebar_no_git_repo')}</span>
                    </div>
                  ),
                  disabled: true,
                },
                {
                  key: 'hint',
                  label: <span style={{ fontSize: 11, color: '#8c8c8c' }}>{t('sidebar_init_hint')}</span>,
                  disabled: true,
                },
                { type: 'divider' as const },
                {
                  key: 'init',
                  label: t('sidebar_init_repo'),
                  icon: <FolderOutlined />,
                  onClick: () => setShowInitRepoModal(true, 'manual'),
                },
              ],
            }}
            trigger={['click']}
          >
            <span style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, color: '#ff4d4f', cursor: 'pointer' }}>
              <WarningOutlined style={{ fontSize: 11 }} />
              {t('sidebar_no_repo')}
            </span>
          </Dropdown>
        )}
        {repoStatus?.current?.short_commit && !gitSyncing && (
          <Text type="secondary" style={{ fontSize: 11, fontFamily: 'monospace' }}>
            {repoStatus.current.short_commit}
          </Text>
        )}
      </div>
    </Header>

    <Modal
      title={initRepoModalReason === 'api_error' ? t('modal_init_api_error_title') : t('modal_init_repo_title')}
      open={showInitRepoModal}
      onOk={handleInitRepo}
      onCancel={() => { setShowInitRepoModal(false); setInitPath('') }}
      okText={t('modal_init_ok')}
      confirmLoading={initLoading}
    >
      {initRepoModalReason === 'api_error' ? (
        <div style={{ fontSize: 13, color: '#595959' }}>
          <p style={{ marginBottom: 12 }}>{t('modal_init_api_error_desc1')}</p>
          <p>{t('modal_init_api_error_desc2')}</p>
        </div>
      ) : (
        <div style={{ marginBottom: 12, fontSize: 13, color: '#595959' }}>
          {t('modal_init_path_hint')}
        </div>
      )}
      <Input
        placeholder="$HOME/path/to/repo"
        value={initPath}
        onChange={e => setInitPath(e.target.value)}
        onPressEnter={handleInitRepo}
      />
    </Modal>
    </>
  )
}

export default Sidebar
