import { useState, useEffect } from 'react'
import { Layout, Card, Table, Tag, Button, Space, Modal, Form, Input, message, Popconfirm, Select, Menu, Tooltip, List } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, TeamOutlined, RobotOutlined, LockOutlined, FolderOutlined, CheckCircleOutlined, ExclamationCircleOutlined, CloseCircleOutlined, SwapOutlined, RocketOutlined, SendOutlined, GlobalOutlined } from '@ant-design/icons'
import { useSearchParams } from 'react-router-dom'
import type { ColumnsType } from 'antd/es/table'
import { getAssetTypes, saveAssetTypesToAPI, AssetTypeConfig } from '../config/bizLines'
import { getTags, saveTagsToAPI, TagConfig } from '../config/tags'
import { getLLMConfigs, saveLLMConfigsToAPI, LLMConfig as LLMConfigType } from '../config/llmConfig'
import { adminApi, llmConfigApi, type RepoListResponse } from '../api/client'
import ColorPicker from '../components/ColorPicker'
import { useTranslation } from 'react-i18next'
import i18n from '../i18n'

const { Sider, Content } = Layout

type AssetType = AssetTypeConfig & { assetCount?: number; built_in?: boolean }
type TagItem = TagConfig & { usageCount?: number; built_in?: boolean }
type LLMConfigItem = LLMConfigType & { key: string; default?: boolean }

type SettingsSection = 'categories' | 'llm' | 'repo' | 'language'

function SettingsView() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const [currentLang, setCurrentLang] = useState(() => localStorage.getItem('lang') || 'en-US')
  const [selectedSection, setSelectedSection] = useState<SettingsSection>(() => {
    const section = searchParams.get('section')
    if (section === 'llm') return 'llm'
    if (section === 'repo') return 'repo'
    if (section === 'language') return 'language'
    return 'categories'
  })
  const [bizLines, setAssetTypes] = useState<AssetType[]>(() => getAssetTypes().map((b, i) => ({ ...b, key: b.name || `biz-${i}`, assetCount: 0 })))
  const [tags, setTags] = useState<TagItem[]>(() => getTags().map((t, i) => ({ ...t, key: t.name || `tag-${i}`, usageCount: 0 })))
  const [llmConfigs, setLlmConfigs] = useState<LLMConfigItem[]>(() => getLLMConfigs().map((c, i) => ({ ...c, key: c.name || `llm-${i}` })))
  const [repoForm] = Form.useForm()
  const [bizLineModalOpen, setAssetTypeModalOpen] = useState(false)
  const [tagModalOpen, setTagModalOpen] = useState(false)
  const [llmModalOpen, setLlmModalOpen] = useState(false)
  const [editingAssetType, setEditingAssetType] = useState<AssetType | null>(null)
  const [editingTag, setEditingTag] = useState<TagItem | null>(null)
  const [editingLlm, setEditingLlm] = useState<LLMConfigItem | null>(null)
  const [form] = Form.useForm()
  const [repoList, setRepoList] = useState<RepoListResponse | null>(null)
  const [repoListLoading] = useState(false)
  const [switchingRepo, setSwitchingRepo] = useState<string | null>(null)
  const [, setIsFirstUse] = useState<boolean | null>(null)
  const [firstUseModalOpen, setFirstUseModalOpen] = useState(false)
  const [firstUseLoading, setFirstUseLoading] = useState(false)
  const [initPath, setInitPath] = useState('')
  const [testModalOpen, setTestModalOpen] = useState(false)
  const [testConfigName, setTestConfigName] = useState<string | null>(null)
  const [testMessage, setTestMessage] = useState('')
  const [testResponse, setTestResponse] = useState<{ success: boolean; content?: string; error?: string } | null>(null)
  const [testLoading, setTestLoading] = useState(false)

  useEffect(() => {
    adminApi.getFirstUse().then((res) => {
      setIsFirstUse(res.first_use)
      if (res.first_use) {
        setFirstUseModalOpen(true)
      }
    }).catch(() => {
      setIsFirstUse(false)
    })
  }, [])

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
    {
      key: 'language',
      icon: <GlobalOutlined />,
      label: 'Language',
    },
  ]

  const handleMenuClick = ({ key }: { key: string }) => {
    if (key === 'bizlines' || key === 'tags') {
      setSelectedSection('categories')
      setSearchParams({})
    } else if (key === 'llm-configs') {
      setSelectedSection('llm')
      setSearchParams({ section: 'llm' })
    } else if (key === 'repo') {
      setSelectedSection('repo')
      setSearchParams({ section: 'repo' })
      adminApi.getRepoConfig().then((config) => {
        repoForm.setFieldsValue(config)
      }).catch(() => {})
      // Fetch repo list for multi-repo management
      adminApi.getRepoList().then((list) => {
        setRepoList(list)
      }).catch(() => {
        setRepoList(null)
      })
    } else if (key === 'language') {
      setSelectedSection('language')
      setSearchParams({ section: 'language' })
    }
  }

  const bizLineColumns: ColumnsType<AssetType> = [
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
              setEditingAssetType(record)
              form.setFieldsValue(record)
              setAssetTypeModalOpen(true)
            }}
          />
          {!record.built_in && (
            <Popconfirm
              title="Delete this biz line?"
              onConfirm={() => {
                const updated = bizLines.filter((b) => b.name !== record.name)
                setAssetTypes(updated)
                saveAssetTypesToAPI(updated.map(({ name, description, color }) => ({ name, description, color })))
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
    { title: 'Name', dataIndex: 'name', key: 'name', render: (name, record) => (
      <Space>
        {name}
        {record.default && <Tag color="green">Default</Tag>}
      </Space>
    )},
    { title: 'Provider', dataIndex: 'provider', key: 'provider', render: (provider) => <Tag color="blue">{provider}</Tag> },
    { title: 'Default Model', dataIndex: 'default_model', key: 'default_model' },
    { title: 'API Key', dataIndex: 'api_key', key: 'api_key', render: (key) => key ? '••••' + key.slice(-4) : '-' },
    {
      title: 'Action',
      key: 'action',
      width: 150,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="Test connection">
            <Button
              type="text"
              size="small"
              icon={<SendOutlined />}
              onClick={() => {
                setTestConfigName(record.name)
                setTestMessage('')
                setTestResponse(null)
                setTestModalOpen(true)
              }}
            />
          </Tooltip>
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

  const handleAssetTypeSave = () => {
    form.validateFields().then((values) => {
      let updated: AssetType[]
      if (editingAssetType) {
        updated = bizLines.map((b) => (b.name === editingAssetType.name ? { ...b, ...values } : b))
        message.success('Updated')
      } else {
        updated = [...bizLines, { ...values, key: values.name, color: values.color || 'default', assetCount: 0 }]
        message.success('Added')
      }
      setAssetTypes(updated)
      saveAssetTypesToAPI(updated.map(({ name, description, color }) => ({ name, description, color })))
      setAssetTypeModalOpen(false)
      form.resetFields()
      setEditingAssetType(null)
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
      const isDefault = values.default === true
      let updated: LLMConfigItem[]
      const hasExistingDefault = llmConfigs.some((c) => c.default)

      if (editingLlm) {
        // When editing, if this one is set as default, unset others
        updated = llmConfigs.map((c) => {
          if (c.name === editingLlm.name) {
            return { ...c, ...values, key: values.name, default: isDefault }
          }
          // If this one is set as default, unset others
          if (isDefault) {
            return { ...c, default: false }
          }
          return c
        })
        message.success('Updated')
      } else {
        // New config
        const newItem = { ...values, key: values.name } as LLMConfigItem
        // If no default exists or this one is set as default, unset others on other configs
        if (!hasExistingDefault || isDefault) {
          updated = [...llmConfigs.map((c) => ({ ...c, default: false })), newItem]
        } else {
          updated = [...llmConfigs, newItem]
        }
        message.success('Added')
      }

      setLlmConfigs(updated)
      saveLLMConfigsToAPI(updated.map(({ key, ...rest }) => rest))
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
                  setEditingAssetType(null)
                  form.resetFields()
                  setAssetTypeModalOpen(true)
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
      const getStatusIcon = (status: string) => {
        switch (status) {
          case 'valid':
            return <CheckCircleOutlined style={{ color: '#52c41a' }} />
          case 'notfound':
            return <ExclamationCircleOutlined style={{ color: '#fa8c16' }} />
          case 'notgit':
            return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
          default:
            return null
        }
      }

      const getStatusText = (status: string) => {
        switch (status) {
          case 'valid':
            return 'Valid'
          case 'notfound':
            return 'Not found'
          case 'notgit':
            return 'Not a git repo'
          default:
            return status
        }
      }

      const handleSwitchRepo = async (path: string) => {
        setSwitchingRepo(path)
        try {
          await adminApi.switchRepo(path)
          message.success('Switched to ' + path)
          // Refresh repo list
          const list = await adminApi.getRepoList()
          setRepoList(list)
          // Update repo config form
          const config = await adminApi.getRepoConfig()
          repoForm.setFieldsValue(config)
        } catch (err: any) {
          message.error(err?.response?.data?.message || 'Failed to switch repo')
        } finally {
          setSwitchingRepo(null)
        }
      }

      return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Card title="Multi-Repo Management" extra={
            <Button icon={<SwapOutlined />} onClick={() => {
              adminApi.getRepoList().then((list) => {
                setRepoList(list)
              }).catch(() => {})
            }}>Refresh</Button>
          }>
            {repoListLoading ? (
              <div style={{ padding: 20, textAlign: 'center' }}>Loading...</div>
            ) : repoList && repoList.repos.length > 0 ? (
              <List
                size="small"
                dataSource={repoList.repos}
                renderItem={(repo) => {
                  const isCurrent = repo.path === repoList.current
                  return (
                    <List.Item
                      actions={!isCurrent ? [
                        <Button
                          key="switch"
                          type="link"
                          size="small"
                          icon={<SwapOutlined />}
                          loading={switchingRepo === repo.path}
                          onClick={() => handleSwitchRepo(repo.path)}
                        >
                          Switch
                        </Button>
                      ] : []}
                    >
                      <List.Item.Meta
                        avatar={getStatusIcon(repo.status)}
                        title={
                          <Space>
                            <span>{repo.path}</span>
                            {isCurrent && <Tag color="blue">Current</Tag>}
                          </Space>
                        }
                        description={getStatusText(repo.status)}
                      />
                    </List.Item>
                  )
                }}
              />
            ) : (
              <div style={{ padding: 20, textAlign: 'center', color: '#999' }}>
                No repositories configured. Run <code>ep init &lt;path&gt;</code> to add one.
              </div>
            )}
          </Card>

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
        </Space>
      )
    }

    if (selectedSection === 'language') {
      const handleLangChange = async (lang: string) => {
        setCurrentLang(lang)
        localStorage.setItem('lang', lang)
        // Change i18next language
        await i18n.changeLanguage(lang)
        // Persist to server config
        try {
          await adminApi.saveConfig({ lang })
          message.success(t('common_success'))
        } catch (err) {
          // Config save failed, but local change is still valid
          console.warn('Failed to save lang to server config:', err)
        }
      }

      return (
        <Card title="Language / 语言">
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <div>
              <p style={{ color: '#666', marginBottom: 16 }}>
                Select your preferred language. This affects the UI display language.
              </p>
              <Select
                value={currentLang}
                onChange={handleLangChange}
                style={{ width: 200 }}
                options={[
                  { value: 'en-US', label: 'English' },
                  { value: 'zh-CN', label: '中文' },
                ]}
              />
            </div>
            <div style={{ color: '#999', fontSize: 12 }}>
              <p>Note: Server-side config requires restart to take effect.</p>
              <p>注意：服务器端配置需要重启服务才能生效。</p>
            </div>
          </Space>
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
          selectedKeys={[selectedSection === 'categories' ? 'bizlines' : selectedSection === 'llm' ? 'llm-configs' : selectedSection === 'repo' ? 'repo' : 'language']}
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
        title={editingAssetType ? 'Edit Biz Line' : 'Add Biz Line'}
        open={bizLineModalOpen}
        onOk={handleAssetTypeSave}
        onCancel={() => {
          setAssetTypeModalOpen(false)
          form.resetFields()
          setEditingAssetType(null)
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
        onCancel={() => {
          setLlmModalOpen(false)
          form.resetFields()
          setEditingLlm(null)
        }}
        footer={[
          <Button key="cancel" onClick={() => {
            setLlmModalOpen(false)
            form.resetFields()
            setEditingLlm(null)
          }}>
            Cancel
          </Button>,
          <Button key="save" type="primary" onClick={handleLlmSave}>
            Save
          </Button>,
        ]}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]} extra="Unique identifier for this LLM config">
            <Input placeholder="e.g., openai-eval" />
          </Form.Item>
          <Form.Item name="provider" label="Provider" rules={[{ required: true }]}>
            <Select placeholder="Select provider">
              <Select.Option value="openai">OpenAI</Select.Option>
              <Select.Option value="openai-compatible">OpenAI Compatible</Select.Option>
              <Select.Option value="claude">Claude</Select.Option>
              <Select.Option value="ollama">Ollama</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="api_key" label="API Key" extra="API key will be stored securely">
            <Input.Password placeholder="sk-..." />
          </Form.Item>
          <Form.Item name="endpoint" label="Endpoint" extra="Leave empty for default (optional for OpenAI/Claude)">
            <Input placeholder="https://api.openai.com/v1" />
          </Form.Item>
          <Form.Item name="default_model" label="Default Model" rules={[{ required: true }]}>
            <Input placeholder="gpt-4o" />
          </Form.Item>
          <Form.Item name="default" valuePropName="checked" extra="Set as the default LLM provider">
            <label><input type="checkbox" style={{ marginRight: 8 }} />Set as default</label>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={null}
        open={firstUseModalOpen}
        footer={null}
        closable={false}
        maskClosable={false}
        width={520}
      >
        <div style={{ textAlign: 'center', padding: '20px 0' }}>
          <RocketOutlined style={{ fontSize: 48, color: '#1890ff', marginBottom: 16 }} />
          <h2 style={{ marginBottom: 8 }}>Welcome to eval-prompt</h2>
          <p style={{ color: '#666', marginBottom: 24 }}>
            Let's set up your first prompt assets repository to get started.
          </p>

          <Form layout="vertical">
            <Form.Item
              label="Repository Path"
              extra="Choose a directory to store your prompt assets"
              required
            >
              <Input
                placeholder="./prompt-assets"
                value={initPath}
                onChange={(e) => setInitPath(e.target.value)}
              />
            </Form.Item>

            <Space direction="vertical" size="small" style={{ width: '100%', marginTop: 16 }}>
              <Button
                type="primary"
                block
                loading={firstUseLoading}
                onClick={async () => {
                  const path = initPath || './prompt-assets'
                  setFirstUseLoading(true)
                  try {
                    await adminApi.switchRepo(path)
                    setFirstUseModalOpen(false)
                    setIsFirstUse(false)
                    // Refresh repo list and config
                    const list = await adminApi.getRepoList()
                    setRepoList(list)
                    const config = await adminApi.getRepoConfig()
                    repoForm.setFieldsValue(config)
                    message.success('Repository initialized!')
                    // Trigger repo section to refresh
                    setSelectedSection('repo')
                  } catch (err: any) {
                    message.error(err?.response?.data?.message || 'Failed to initialize repository')
                  } finally {
                    setFirstUseLoading(false)
                  }
                }}
              >
                Initialize Repository
              </Button>
              <Button
                block
                onClick={() => {
                  setFirstUseModalOpen(false)
                  setIsFirstUse(false)
                }}
              >
                Skip for now
              </Button>
            </Space>
          </Form>

          <div style={{ marginTop: 24, padding: '16px', background: '#f5f5f5', borderRadius: 8, fontSize: 12, color: '#666' }}>
            <strong>Note:</strong> You can manage multiple repositories later using{' '}
            <code>ep init &lt;path&gt;</code> or via the Repository settings page.
          </div>
        </div>
      </Modal>

      <Modal
        title="Test LLM Connection"
        open={testModalOpen}
        onCancel={() => setTestModalOpen(false)}
        footer={null}
        width={520}
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <div style={{ background: '#f5f5f5', padding: 12, borderRadius: 6, fontSize: 13 }}>
            <div style={{ color: '#666' }}>Testing config: <strong>{testConfigName}</strong></div>
          </div>

          <Select
            placeholder="Select a saved config to test"
            value={testConfigName}
            onChange={(name) => setTestConfigName(name)}
            style={{ width: '100%' }}
          >
            {llmConfigs.map((c) => (
              <Select.Option key={c.name} value={c.name}>
                {c.name} ({c.provider} - {c.default_model})
              </Select.Option>
            ))}
          </Select>

          <Input.TextArea
            placeholder="Test message (leave empty for default)"
            value={testMessage}
            onChange={(e) => setTestMessage(e.target.value)}
            rows={2}
          />

          <Button
            type="primary"
            icon={<SendOutlined />}
            loading={testLoading}
            disabled={!testConfigName}
            onClick={async () => {
              if (!testConfigName) return
              setTestLoading(true)
              setTestResponse(null)
              try {
                const result = await llmConfigApi.testByName(testConfigName, testMessage || undefined)
                setTestResponse(result)
              } catch (err: any) {
                setTestResponse({ success: false, error: err?.message || 'Request failed' })
              } finally {
                setTestLoading(false)
              }
            }}
            block
          >
            Send Test Message
          </Button>

          {testResponse && (
            <div style={{
              padding: 12,
              borderRadius: 6,
              background: testResponse.success ? '#f6ffed' : '#fff2f0',
              border: `1px solid ${testResponse.success ? '#b7eb8f' : '#ffa39e'}`,
            }}>
              <div style={{ marginBottom: 4, fontWeight: 500, color: testResponse.success ? '#52c41a' : '#ff4d4f' }}>
                {testResponse.success ? 'Success!' : 'Failed'}
              </div>
              {testResponse.success && testResponse.content && (
                <div style={{ color: '#333', whiteSpace: 'pre-wrap' }}>{testResponse.content}</div>
              )}
              {testResponse.error && (
                <div style={{ color: '#ff4d4f', fontSize: 12, whiteSpace: 'pre-wrap' }}>{testResponse.error}</div>
              )}
            </div>
          )}
        </Space>
      </Modal>
    </Layout>
  )
}

export default SettingsView
