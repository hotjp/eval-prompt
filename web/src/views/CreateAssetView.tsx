import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Form, Input, Select, Button, Space, message } from 'antd'
import { SaveOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import { assetApi } from '../api/client'
import { getBizLines } from '../config/bizLines'
import { getTags } from '../config/tags'

const { TextArea } = Input

const bizLineOptions = getBizLines().map((b) => ({ label: b.name, value: b.name }))
const tagOptions = getTags().map((t) => ({ label: t.name, value: t.name }))
const stateOptions = [
  { label: 'Active', value: 'active' },
  { label: 'Draft', value: 'draft' },
]

function CreateAssetView() {
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [saving, setSaving] = useState(false)

  const handleCreate = async (values: { name: string; description?: string; biz_line: string; tags?: string[]; state?: string }) => {
    setSaving(true)
    try {
      await assetApi.create({
        id: values.name,
        name: values.name,
        description: values.description,
        biz_line: values.biz_line,
        tags: values.tags,
      })
      message.success('Asset created successfully')
      navigate(`/assets/${values.name}/edit`)
    } catch {
      message.error('Failed to create asset')
    } finally {
      setSaving(false)
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
            initialValues={{ state: 'draft', biz_line: 'common' }}
          >
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
              name="biz_line"
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
              tooltip="Describe the purpose, use cases, and expected behavior of this prompt"
            >
              <TextArea rows={3} style={{ width: 600 }} placeholder="Describe this prompt asset..." />
            </Form.Item>

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
