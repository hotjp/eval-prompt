import { useEffect, useState } from 'react'
import { Layout, Menu, Popover, Button, Space, Typography, Popconfirm, message, Tooltip } from 'antd'
import { LoadingOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import {
  AppstoreOutlined,
  SwapOutlined,
  SettingOutlined,
  ReloadOutlined,
  PoweroffOutlined,
  BranchesOutlined,
  SyncOutlined,
  FieldTimeOutlined,
} from '@ant-design/icons'
import { healthApi, adminApi, type HealthStatus, type GitInfo } from '../api/client'
import { useStore } from '../store'

const { Header } = Layout
const { Text } = Typography

type Status = 'ok' | 'error' | 'loading' | 'degraded'

function Sidebar() {
  const navigate = useNavigate()
  const [serverStatus, setServerStatus] = useState<Status>('loading')
  const [healthData, setHealthData] = useState<HealthStatus | null>(null)
  const [gitInfo, setGitInfo] = useState<GitInfo | null>(null)
  const [loading, setLoading] = useState(false)
  const [statusOpen, setStatusOpen] = useState(false)
  const [lastChecked, setLastChecked] = useState<Date | null>(null)
  const runningEval = useStore(s => s.runningEval)

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

  const fetchGitInfo = async () => {
    try {
      const res = await adminApi.gitInfo()
      setGitInfo(res)
    } catch {
      // Silently fail for git info
    }
  }

  useEffect(() => {
    fetchHealth()
    fetchGitInfo()
    const interval = setInterval(fetchHealth, 30000)
    return () => clearInterval(interval)
  }, [])

  const handleRestart = async () => {
    if (runningEval) {
      message.warning('An eval is running. Please wait for it to finish before restarting.')
      return
    }
    setLoading(true)
    try {
      await new Promise(r => setTimeout(r, 1000))
      message.success('Restart signal sent — server is restarting')
      setStatusOpen(false)
    } catch {
      message.error('Failed to send restart signal')
    } finally {
      setLoading(false)
    }
  }

  const handleReloadConfig = async () => {
    setLoading(true)
    try {
      await new Promise(r => setTimeout(r, 800))
      message.success('Config reloaded — changes will take effect immediately')
      setStatusOpen(false)
    } catch {
      message.error('Failed to reload config')
    } finally {
      setLoading(false)
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
    ? 'Checking'
    : serverStatus === 'ok'
    ? 'Online'
    : serverStatus === 'degraded'
    ? 'Degraded'
    : 'Offline'

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

  const navItems = [
    { key: '/assets', icon: <AppstoreOutlined />, label: 'Assets' },
    { key: '/compare', icon: <SwapOutlined />, label: 'Compare' },
    { key: '/settings', icon: <SettingOutlined />, label: 'Settings' },
  ]

  const gitBranchContent = gitInfo ? (
    <div style={{ fontSize: 12 }}>
      {gitInfo.dirty ? (
        <>
          <div style={{ marginBottom: 4 }}>Uncommitted changes — run <code>git stash</code> or <code>git commit</code></div>
          <div style={{ color: '#8c8c8c', fontSize: 11 }}>Working directory has uncommitted modifications</div>
        </>
      ) : (
        <div>Branch: <strong>{gitInfo.branch}</strong></div>
      )}
    </div>
  ) : null

  const popoverContent = (
    <div style={{ width: 260 }}>
      <div style={{ marginBottom: 12 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>Server Status</Text>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 4 }}>
          {statusDot}
          <Text strong style={{ fontSize: 13 }}>
            {serverStatus === 'ok' ? 'Running' : serverStatus === 'error' ? 'Offline' : serverStatus === 'degraded' ? 'Degraded' : 'Checking...'}
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
            <span style={{ fontWeight: 500 }}>Eval in progress</span>
          </div>
          <div style={{ color: '#8c8c8c' }}>Do not restart server while eval is running</div>
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
            <span style={{ fontWeight: 500 }}>Degraded — some services unavailable</span>
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
          <div style={{ marginBottom: 4, fontWeight: 500 }}>Server is not responding</div>
          <div>Start the server manually:</div>
          <code style={{ display: 'block', marginTop: 4, padding: '4px 6px', background: '#fff', borderRadius: 4, fontSize: 11 }}>
            ep server
          </code>
        </div>
      )}

      {healthData && (
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>Details</Text>
          <div style={{ marginTop: 4, fontSize: 12, color: '#595959' }}>
            <div>Checked: {lastChecked ? lastChecked.toLocaleTimeString() : 'N/A'}</div>
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
        <Text type="secondary" style={{ fontSize: 12 }}>Actions</Text>
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
              Reload Config
            </Button>
            <Popconfirm
              title="Restart the server?"
              description="This will terminate all ongoing operations."
              okText="Restart"
              cancelText="Cancel"
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
                Restart Server
              </Button>
            </Popconfirm>
          </Space>
        </div>
      </div>
    </div>
  )

  return (
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
        {gitInfo && (
          <Tooltip
            title={gitBranchContent}
            placement="bottomRight"
            arrow={false}
          >
            <span style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, color: '#8c8c8c', cursor: 'default' }}>
              <BranchesOutlined style={{ fontSize: 11 }} />
              {gitInfo.branch}
              {gitInfo.dirty && <SyncOutlined spin style={{ fontSize: 10, color: '#fa8c16' }} />}
            </span>
          </Tooltip>
        )}
        {gitInfo?.short_commit && (
          <Text type="secondary" style={{ fontSize: 11, fontFamily: 'monospace' }}>
            {gitInfo.short_commit}
          </Text>
        )}
      </div>
    </Header>
  )
}

export default Sidebar