import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Tag, Button, Space, Select, Spin, message } from 'antd'
import { ReloadOutlined, EyeOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { executionApi } from '../api/client'
import type { Execution } from '../api/client'

function ExecutionListView() {
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
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 120,
      render: (id: string) => <code style={{ fontSize: 11 }}>{id.slice(-10)}</code>,
    },
    {
      title: 'Asset',
      dataIndex: 'asset_id',
      key: 'asset_id',
      width: 150,
      render: (assetId: string) => <span style={{ fontSize: 13 }}>{assetId}</span>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
    },
    {
      title: 'Progress',
      key: 'progress',
      width: 120,
      render: (_: unknown, record: Execution) => (
        <span>
          {record.completed_cases}/{record.total_cases} cases
        </span>
      ),
    },
    {
      title: 'Model',
      dataIndex: 'model',
      key: 'model',
      width: 100,
    },
    {
      title: 'Time',
      key: 'duration',
      width: 80,
      render: (_: unknown, record: Execution) => formatDuration(record),
    },
    {
      title: 'Action',
      key: 'action',
      width: 120,
      render: (_: unknown, record: Execution) => (
        <Button
          size="small"
          icon={<EyeOutlined />}
          onClick={() => navigate(`/executions/${record.id}/calls`)}
        >
          View Calls
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
        title="Executions"
        extra={
          <Space>
            <Select
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: 140 }}
              options={[
                { value: 'all', label: 'All Status' },
                { value: 'completed', label: 'Completed' },
                { value: 'running', label: 'Running' },
                { value: 'failed', label: 'Failed' },
                { value: 'pending', label: 'Pending' },
              ]}
            />
            <Button icon={<ReloadOutlined />} onClick={loadExecutions}>
              Refresh
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
