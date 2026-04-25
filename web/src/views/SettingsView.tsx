import { useState } from 'react'
import { Layout, Card, Table, Tag, Button, Space, Modal, Form, Input, message, Popconfirm, Select, Menu, Tooltip } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, TeamOutlined, RobotOutlined, LockOutlined, FolderOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getBizLines, saveBizLinesToAPI, BizLineConfig } from '../config/bizLines'
import { getTags, saveTagsToAPI, TagConfig } from '../config/tags'
import { getLLMConfigs, saveLLMConfigsToAPI, LLMConfig as LLMConfigType } from '../config/llmConfig'
import { adminApi } from '../api/client'
import ColorPicker from '../components/ColorPicker'

const { Sider, Content } = Layout

type BizLine = BizLineConfig & { assetCount?: number; built_in?: boolean }
type TagItem = TagConfig & { usageCount?: number; built_in?: boolean }
type LLMConfigItem = LLMConfigType & { key: string }

type SettingsSection = 'categories' | 'llm' | 'repo'

function SettingsView() {
  const [selectedSection, setSelectedSection] = useState<SettingsSection>('categories')
  const [bizLines, setBizLines] = useState<BizLine[]>(() => getBizLines().map((b, i) => ({ ...b, key: b.name || `biz-${i}`, assetCount: 0 })))
  const [tags, setTags] = useState<TagItem[]>(() => getTags().map((t, i) => ({ ...t, key: t.name || `tag-${i}`, usageCount: 0 })))
  const [llmConfigs, setLlmConfigs] = useState<LLMConfigItem[]>(() => getLLMConfigs().map((c, i) => ({ ...c, key: c.name || `llm-${i}` })))
  const [repoForm] = Form.useForm()
  const [bizLineModalOpen, setBizLineModalOpen] = useState(false)
  const [tagModalOpen, setTagModalOpen] = useState(false)
  const [llmModalOpen, setLlmModalOpen] = useState(false)
  const [editingBizLine, setEditingBizLine] = useState<BizLine | null>(null)
  const [editingTag, setEditingTag] = useState<TagItem | null>(null)
  const [editingLlm, setEditingLlm] = useState<LLMConfigItem | null>(null)
  const [form] = Form.useForm()

  const menuItems = [
    {
      key: 'categories',
      icon: <TeamOutlined />,
      label: 'Categories',
      children: [
        { key: 'bizlines', label: 'Business Lines' },
        { key: 'tags', label: 'Tags' },
      ],
    },
    {
      key: 'llm',
      icon: <RobotOutlined />,
      label: 'LLM',
      children: [
        { key: 'llm-configs', label: 'Configurations' },
      ],
    },
    {
      key: 'repo',
      icon: <FolderOutlined />,
      label: 'Repository',
    },
  ]

  const handleMenuClick = ({ key }: { key: string }) => {
    if (key === 'bizlines' || key === 'tags') {
      setSelectedSection('categories')
    } else if (key === 'llm-configs') {
      setSelectedSection('llm')
    } else if (key === 'repo') {
      setSelectedSection('repo')
      adminApi.getRepoConfig().then((config) => {
        repoForm.setFieldsValue(config)
      }).catch(() => {})
    }
  }

  const bizLineColumns: ColumnsType<BizLine> = [
    { title: 'Name', dataIndex: 'name', key: 'name', render: (name, record) => (
      <Space>
        <Tag key={name} color={record.color}>{name}</Tag>
        {record.built_in && <Tooltip title="Built-in"><LockOutlined style={{ color: '#999' }} /></Tooltip>}
      </Space>
    )},
    { title: 'Description', dataIndex: 'description', key: 'description' },
    { title: 'Assets', dataIndex: 'assetCount', key: 'assetCount', width: 80 },
    {
      title: 'Action',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Space>
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingBizLine(record)
              form.setFieldsValue(record)
              setBizLineModalOpen(true)
            }}
          />
          {!record.built_in && (
            <Popconfirm
              title="Delete this biz line?"
              onConfirm={() => {
                const updated = bizLines.filter((b) => b.name !== record.name)
                setBizLines(updated)
                saveBizLinesToAPI(updated.map(({ name, description, color }) => ({ name, description, color })))
                message.success('Deleted')
              }}
            >
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const tagColumns: ColumnsType<TagItem> = [
    {
      title: 'Tag',
      dataIndex: 'name',
      key: 'name',
      render: (name, record) => (
        <Space>
          <Tag key={name} color={record.color}>{name}</Tag>
          {record.built_in && <Tooltip title="Built-in"><LockOutlined style={{ color: '#999' }} /></Tooltip>}
        </Space>
      ),
    },
    { title: 'Usage', dataIndex: 'usageCount', key: 'usageCount', width: 80 },
    {
      title: 'Action',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Space>
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingTag(record)
              form.setFieldsValue(record)
              setTagModalOpen(true)
            }}
          />
          {!record.built_in && (
            <Popconfirm
              title="Delete this tag?"
              onConfirm={() => {
                const updated = tags.filter((t) => t.name !== record.name)
                setTags(updated)
                saveTagsToAPI(updated.map(({ name, color }) => ({ name, color })))
                message.success('Deleted')
              }}
            >
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const llmColumns: ColumnsType<LLMConfigItem> = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Provider', dataIndex: 'provider', key: 'provider', render: (provider) => <Tag color="blue">{provider}</Tag> },
    { title: 'Default Model', dataIndex: 'default_model', key: 'default_model' },
    { title: 'API Key', dataIndex: 'api_key', key: 'api_key', render: (key) => key ? '••••' + key.slice(-4) : '-' },
    {
      title: 'Action',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Space>
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingLlm(record)
              form.setFieldsValue(record)
              setLlmModalOpen(true)
            }}
          />
          <Popconfirm
            title="Delete this LLM config?"
            onConfirm={() => {
              const updated = llmConfigs.filter((c) => c.name !== record.name)
              setLlmConfigs(updated)
              saveLLMConfigsToAPI(updated.map(({ name, provider, api_key, endpoint, default_model }) => ({ name, provider, api_key, endpoint, default_model })))
              message.success('Deleted')
            }}
          >
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const handleBizLineSave = () => {
    form.validateFields().then((values) => {
      let updated: BizLine[]
      if (editingBizLine) {
        updated = bizLines.map((b) => (b.name === editingBizLine.name ? { ...b, ...values } : b))
        message.success('Updated')
      } else {
        updated = [...bizLines, { ...values, key: values.name, color: values.color || 'default', assetCount: 0 }]
        message.success('Added')
      }
      setBizLines(updated)
      saveBizLinesToAPI(updated.map(({ name, description, color }) => ({ name, description, color })))
      setBizLineModalOpen(false)
      form.resetFields()
      setEditingBizLine(null)
    })
  }

  const handleTagSave = () => {
    form.validateFields().then((values) => {
      let updated: TagItem[]
      if (editingTag) {
        updated = tags.map((t) => (t.name === editingTag.name ? { ...t, ...values } : t))
        message.success('Updated')
      } else {
        updated = [...tags, { ...values, color: values.color || 'blue', usageCount: 0 }]
        message.success('Added')
      }
      setTags(updated)
      saveTagsToAPI(updated.map(({ name, color }) => ({ name, color })))
      setTagModalOpen(false)
      form.resetFields()
      setEditingTag(null)
    })
  }

  const handleLlmSave = () => {
    form.validateFields().then((values) => {
      let updated: LLMConfigItem[]
      if (editingLlm) {
        updated = llmConfigs.map((c) => (c.name === editingLlm.name ? { ...c, ...values, key: values.name } : c))
        message.success('Updated')
      } else {
        updated = [...llmConfigs, { ...values, key: values.name }]
        message.success('Added')
      }
      setLlmConfigs(updated)
      saveLLMConfigsToAPI(updated.map(({ name, provider, api_key, endpoint, default_model }) => ({ name, provider, api_key, endpoint, default_model })))
      setLlmModalOpen(false)
      form.resetFields()
      setEditingLlm(null)
    })
  }

  const renderContent = () => {
    if (selectedSection === 'categories') {
      return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Card
            title="Business Lines"
            extra={
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditingBizLine(null)
                  form.resetFields()
                  setBizLineModalOpen(true)
                }}
              >
                Add
              </Button>
            }
          >
            <Table columns={bizLineColumns} dataSource={bizLines} rowKey="key" pagination={false} size="small" />
          </Card>

          <Card
            title="Tags"
            extra={
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditingTag(null)
                  form.resetFields()
                  setTagModalOpen(true)
                }}
              >
                Add
              </Button>
            }
          >
            <Table columns={tagColumns} dataSource={tags} rowKey="key" pagination={false} size="small" />
          </Card>
        </Space>
      )
    }

    if (selectedSection === 'llm') {
      return (
        <Card
          title="LLM Configurations"
          extra={
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setEditingLlm(null)
                form.resetFields()
                setLlmModalOpen(true)
              }}
            >
              Add LLM
            </Button>
          }
        >
          <Table columns={llmColumns} dataSource={llmConfigs} rowKey="key" pagination={false} size="small" />
        </Card>
      )
    }

    if (selectedSection === 'repo') {
      return (
        <Card title="Repository Settings">
          <Form form={repoForm} layout="vertical">
            <Form.Item
              name="repo_path"
              label="Repository Path"
              rules={[{ required: true }]}
              extra="Absolute path to the git repository root containing prompt assets"
            >
              <Input placeholder="/Users/name/prompts-repo" />
            </Form.Item>
            <Form.Item
              name="assets_dir"
              label="Assets Directory"
              rules={[{ required: true }]}
              extra="Directory where .md prompt files are stored, relative to repo root"
            >
              <Input placeholder="prompts" />
            </Form.Item>
            <Form.Item
              name="evals_dir"
              label="Evals Directory"
              rules={[{ required: true }]}
              extra="Directory for eval results and traces, relative to repo root"
            >
              <Input placeholder=".evals" />
            </Form.Item>
            <Button type="primary" onClick={() => {
              repoForm.validateFields().then((values) => {
                adminApi.saveRepoConfig(values).then(() => {
                  message.success('Saved')
                }).catch(() => {
                  message.error('Failed to save')
                })
              })
            }}>Save</Button>
          </Form>
        </Card>
      )
    }

    return null
  }

  return (
    <Layout style={{ minHeight: 'calc(100vh - 64px)', background: '#fff' }}>
      <Sider
        width={220}
        style={{ background: '#fff', borderRight: '1px solid #f0f0f0', padding: '16px 0' }}
      >
        <Menu
          mode="inline"
          selectedKeys={[selectedSection === 'categories' ? 'bizlines' : selectedSection === 'llm' ? 'llm-configs' : 'repo']}
          defaultOpenKeys={['taxonomy', 'llm']}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ border: 0 }}
        />
      </Sider>
      <Content style={{ padding: '24px', overflow: 'initial' }}>
        {renderContent()}
      </Content>

      <Modal
        title={editingBizLine ? 'Edit Biz Line' : 'Add Biz Line'}
        open={bizLineModalOpen}
        onOk={handleBizLineSave}
        onCancel={() => {
          setBizLineModalOpen(false)
          form.resetFields()
          setEditingBizLine(null)
        }}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input />
          </Form.Item>
          <Form.Item name="color" label="Color" valuePropName="color">
            <ColorPicker />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingTag ? 'Edit Tag' : 'Add Tag'}
        open={tagModalOpen}
        onOk={handleTagSave}
        onCancel={() => {
          setTagModalOpen(false)
          form.resetFields()
          setEditingTag(null)
        }}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="color" label="Color" valuePropName="color">
            <ColorPicker />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingLlm ? 'Edit LLM' : 'Add LLM'}
        open={llmModalOpen}
        onOk={handleLlmSave}
        onCancel={() => {
          setLlmModalOpen(false)
          form.resetFields()
          setEditingLlm(null)
        }}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]} extra="Unique identifier for this LLM config">
            <Input placeholder="e.g., openai-eval" />
          </Form.Item>
          <Form.Item name="provider" label="Provider" rules={[{ required: true }]}>
            <Select placeholder="Select provider">
              <Select.Option value="openai">OpenAI</Select.Option>
              <Select.Option value="claude">Claude</Select.Option>
              <Select.Option value="ollama">Ollama</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="api_key" label="API Key" rules={[{ required: true }]} extra="API key will be stored securely">
            <Input.Password placeholder="sk-..." />
          </Form.Item>
          <Form.Item name="endpoint" label="Endpoint" extra="Leave empty for default (optional for OpenAI/Claude)">
            <Input placeholder="https://api.openai.com/v1" />
          </Form.Item>
          <Form.Item name="default_model" label="Default Model" rules={[{ required: true }]}>
            <Input placeholder="gpt-4o" />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  )
}

export default SettingsView
