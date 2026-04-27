import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Tag, Button, Space, Select, Spin, message } from 'antd'
import { ReloadOutlined, EyeOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { executionApi } from '../api/client'
import type { Execution } from '../api/client'
import { useTranslation } from 'react-i18next'

function ExecutionListView() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [executions, setExecutions] = useState<Execution[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<string>('all')

  useEffect(() => {
    loadExecutions()
  }, [statusFilter])

  const loadExecutions = async () => {
    setLoading(true)
    try {
      const filters = statusFilter !== 'all' ? { status: statusFilter } : undefined
      const data = await executionApi.list(filters)
      setExecutions(data.executions)
    } catch {
      message.error('Failed to load executions')
    } finally {
      setLoading(false)
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'green'
      case 'running': return 'blue'
      case 'failed': return 'red'
      case 'pending': return 'orange'
      default: return 'default'
    }
  }

  const formatDuration = (exec: Execution) => {
    if (!exec.created_at) return '-'
    const start = new Date(exec.created_at).getTime()
    const end = exec.updated_at ? new Date(exec.updated_at).getTime() : Date.now()
    const diffMs = end - start
    const diffMins = Math.floor(diffMs / 60000)
    if (diffMins < 1) return '<1m'
    if (diffMins < 60) return `${diffMins}m`
    const diffHours = Math.floor(diffMins / 60)
    return `${diffHours}h ${diffMins % 60}m`
  }

  const columns: ColumnsType<Execution> = [
    {
      title: t('execution_list_col_id'),
      dataIndex: 'id',
      key: 'id',
      width: 120,
      render: (id: string) => <code style={{ fontSize: 11 }}>{id.slice(-10)}</code>,
    },
    {
      title: t('execution_list_col_asset'),
      dataIndex: 'asset_id',
      key: 'asset_id',
      width: 150,
      render: (assetId: string) => <span style={{ fontSize: 13 }}>{assetId}</span>,
    },
    {
      title: t('execution_list_col_status'),
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
    },
    {
      title: t('execution_list_col_progress'),
      key: 'progress',
      width: 120,
      render: (_: unknown, record: Execution) => (
        <span>
          {record.completed_cases}/{record.total_cases} {t('execution_list_cases')}
        </span>
      ),
    },
    {
      title: t('execution_list_col_model'),
      dataIndex: 'model',
      key: 'model',
      width: 100,
    },
    {
      title: t('execution_list_col_time'),
      key: 'duration',
      width: 80,
      render: (_: unknown, record: Execution) => formatDuration(record),
    },
    {
      title: t('execution_list_col_action'),
      key: 'action',
      width: 120,
      render: (_: unknown, record: Execution) => (
        <Button
          size="small"
          icon={<EyeOutlined />}
          onClick={() => navigate(`/executions/${record.id}/calls`)}
        >
          {t('execution_list_view_calls')}
        </Button>
      ),
    },
  ]

  if (loading && executions.length === 0) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  return (
    <div>
      <Card
        title={t('execution_list_title')}
        extra={
          <Space>
            <Select
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: 140 }}
              options={[
                { value: 'all', label: t('execution_list_all_status') },
                { value: 'completed', label: t('execution_list_completed') },
                { value: 'running', label: t('execution_list_running') },
                { value: 'failed', label: t('execution_list_failed') },
                { value: 'pending', label: t('execution_list_pending') },
              ]}
            />
            <Button icon={<ReloadOutlined />} onClick={loadExecutions}>
              {t('execution_list_refresh')}
            </Button>
          </Space>
        }
      >
        <Table
          dataSource={executions}
          columns={columns}
          rowKey="id"
          size="small"
          pagination={{ pageSize: 20 }}
        />
      </Card>
    </div>
  )
}

export default ExecutionListView
