import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Table, Card, Tag, Input, Select, Button, Space, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { assetApi } from '../api/client'
import type { AssetSummary } from '../api/client'

const { Search } = Input

function AssetListView() {
  const navigate = useNavigate()
  const [assets, setAssets] = useState<AssetSummary[]>([])
  const [loading, setLoading] = useState(false)
  const [bizLineFilter, setBizLineFilter] = useState<string>('')
  const [tagFilter, setTagFilter] = useState<string>('')
  const [searchText, setSearchText] = useState('')

  useEffect(() => {
    loadAssets()
  }, [bizLineFilter, tagFilter])

  const loadAssets = async () => {
    setLoading(true)
    try {
      const data = await assetApi.list({ biz_line: bizLineFilter || undefined, tag: tagFilter || undefined })
      setAssets(data)
    } catch {
      message.error('Failed to load assets')
    } finally {
      setLoading(false)
    }
  }

  const filteredAssets = assets.filter((asset) =>
    asset.name.toLowerCase().includes(searchText.toLowerCase()) ||
    asset.description.toLowerCase().includes(searchText.toLowerCase())
  )

  const columns: ColumnsType<AssetSummary> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (name, record) => (
        <a onClick={() => navigate(`/assets/${record.id}/edit`)}>{name}</a>
      ),
    },
    { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: 'Biz Line', dataIndex: 'biz_line', key: 'biz_line' },
    {
      title: 'Tags',
      dataIndex: 'tags',
      key: 'tags',
      render: (tags: string[]) => (
        <>
          {tags.map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </>
      ),
    },
    {
      title: 'State',
      dataIndex: 'state',
      key: 'state',
      render: (state) => {
        const color = state === 'active' ? 'green' : state === 'draft' ? 'orange' : 'gray'
        return <Tag color={color}>{state}</Tag>
      },
    },
  ]

  return (
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card>
          <Space wrap>
            <Search
              placeholder="Search assets..."
              onSearch={setSearchText}
              style={{ width: 200 }}
            />
            <Select
              placeholder="Biz Line"
              allowClear
              style={{ width: 150 }}
              onChange={setBizLineFilter}
              options={[
                { label: 'Search', value: 'search' },
                { label: 'Ads', value: 'ads' },
                { label: 'Rec', value: 'rec' },
                { label: 'Feed', value: 'feed' },
              ]}
            />
            <Select
              placeholder="Tag"
              allowClear
              style={{ width: 150 }}
              onChange={setTagFilter}
              options={[
                { label: 'llm', value: 'llm' },
                { label: 'rag', value: 'rag' },
                { label: 'agent', value: 'agent' },
              ]}
            />
            <Button onClick={loadAssets}>Refresh</Button>
          </Space>
        </Card>
        <Table
          columns={columns}
          dataSource={filteredAssets}
          rowKey="id"
          loading={loading}
        />
      </Space>
    </div>
  )
}

export default AssetListView
