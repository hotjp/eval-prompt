import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Form, Input, Select, Button, Space, message } from 'antd'
import { SaveOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import { assetApi } from '../api/client'
import { getAssetTypes } from '../config/bizLines'
import { getTags } from '../config/tags'

const { TextArea } = Input

const bizLineOptions = getAssetTypes().map((b) => ({ label: b.name, value: b.name }))
const tagOptions = getTags().map((t) => ({ label: t.name, value: t.name }))
const stateOptions = [
  { label: 'Active', value: 'active' },
  { label: 'Draft', value: 'draft' },
]
const categoryOptions = [
  { label: 'Prompt', value: 'content' },
  { label: 'Eval Case', value: 'eval' },
  { label: 'Metric', value: 'metric' },
]

function CreateAssetView() {
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [saving, setSaving] = useState(false)
  const [selectedCategory, setSelectedCategory] = useState<string>('content')

  const handleCreate = async (values: { name: string; description?: string; asset_type: string; tags?: string[]; state?: string; category?: string; content?: string; test_cases?: string; rubric?: string }) => {
    setSaving(true)
    try {
      const payload: { id: string; name: string; description?: string; asset_type?: string; tags?: string[]; category?: string; content?: string; test_cases?: string; rubric?: string } = {
        id: values.name,
        name: values.name,
        description: values.description,
        asset_type: values.asset_type,
        tags: values.tags,
        category: values.category,
      }
      if (values.category === 'content' && values.content) {
        payload.content = values.content
      } else if (values.category === 'eval' && values.test_cases) {
        payload.test_cases = values.test_cases
      } else if (values.category === 'metric' && values.rubric) {
        payload.rubric = values.rubric
      }
      await assetApi.create(payload)
      message.success('Asset created successfully')
      navigate(`/assets/${values.name}/edit`)
    } catch {
      message.error('Failed to create asset')
    } finally {
      setSaving(false)
    }
  }

  const renderCategoryFields = () => {
    switch (selectedCategory) {
      case 'content':
        return (
          <Form.Item label="Content" name="content">
            <TextArea rows={10} style={{ width: 600 }} placeholder="Enter prompt content..." />
          </Form.Item>
        )
      case 'eval':
        return (
          <Form.Item label="Test Cases (YAML)" name="test_cases">
            <TextArea rows={10} style={{ width: 600, fontFamily: 'monospace' }} placeholder="test_cases:&#10;  - id: case1&#10;    name: Test Case 1&#10;    input: |&#10;      ...&#10;    expected: |&#10;      ..." />
          </Form.Item>
        )
      case 'metric':
        return (
          <Form.Item label="Rubric (YAML)" name="rubric">
            <TextArea rows={10} style={{ width: 600, fontFamily: 'monospace' }} placeholder="rubric:&#10;  - check: correctness&#10;    weight: 0.4&#10;    criteria: |&#10;      ..." />
          </Form.Item>
        )
      default:
        return null
    }
  }

  return (
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card
          title="Create New Asset"
          extra={
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/assets')}>
              Back
            </Button>
          }
        >
          <Form
            form={form}
            layout="vertical"
            onFinish={handleCreate}
            initialValues={{ state: 'draft', asset_type: 'common', category: 'content' }}
          >
            <Form.Item
              label="Category"
              name="category"
              rules={[{ required: true, message: 'Please select category' }]}
              tooltip="Category determines the asset type. This cannot be changed after creation."
            >
              <Select
                options={categoryOptions}
                style={{ width: 200 }}
                onChange={(value) => setSelectedCategory(value)}
              />
            </Form.Item>

            <Form.Item
              label="Asset Name"
              name="name"
              rules={[
                { required: true, message: 'Please enter asset name' },
                { pattern: /^[a-zA-Z0-9_-]+$/, message: 'Only letters, numbers, - and _ are allowed' },
              ]}
              extra="This will be used as the asset ID"
            >
              <Input placeholder="e.g., customer-churn-scorer" style={{ width: 400 }} />
            </Form.Item>

            <Form.Item
              label="Business Line"
              name="asset_type"
              rules={[{ required: true, message: 'Please select business line' }]}
            >
              <Select options={bizLineOptions} style={{ width: 200 }} />
            </Form.Item>

            <Form.Item label="Tags" name="tags">
              <Select mode="multiple" options={tagOptions} style={{ width: 400 }} placeholder="Select one or more tags (e.g., prod, llm)" />
            </Form.Item>

            <Form.Item label="State" name="state">
              <Select options={stateOptions} style={{ width: 200 }} />
            </Form.Item>

            <Form.Item
              label="Description"
              name="description"
              tooltip="Describe the purpose, use cases, and expected behavior"
            >
              <TextArea rows={3} style={{ width: 600 }} placeholder="Describe this asset..." />
            </Form.Item>

            {renderCategoryFields()}

            <Form.Item>
              <Space>
                <Button type="primary" icon={<SaveOutlined />} htmlType="submit" loading={saving}>
                  Create
                </Button>
                <Button onClick={() => navigate('/assets')}>Cancel</Button>
              </Space>
            </Form.Item>
          </Form>
        </Card>
      </Space>
    </div>
  )
}

export default CreateAssetView
