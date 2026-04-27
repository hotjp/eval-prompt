import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Form, Input, Select, Button, Space, message } from 'antd'
import { SaveOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { assetApi } from '../api/client'
import { getAssetTypes } from '../config/assetTypes'
import { getTags } from '../config/tags'
import { categoryOptions } from '../config/categories'

const { TextArea } = Input

const assetTypeOptions = getAssetTypes().map((b) => ({ label: b.name, value: b.name }))
const defaultAssetType = getAssetTypes()[0]?.name || 'prompt'
const tagOptions = getTags().map((t) => ({ label: t.name, value: t.name }))
const stateOptions = [
  { label: 'Active', value: 'active' },
  { label: 'Draft', value: 'draft' },
]

function CreateAssetView() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [saving, setSaving] = useState(false)
  const [selectedCategory, setSelectedCategory] = useState<string>('content')

  const handleCreate = async (values: { name: string; description?: string; asset_type: string; tags?: string[]; state?: string; category?: string; test_cases?: string; rubric?: string }) => {
    setSaving(true)
    try {
      const payload = {
        id: values.name,
        name: values.name,
        description: values.description,
        asset_type: values.asset_type,
        tags: values.tags,
        category: values.category,
      }
      await assetApi.create(payload)
      message.success(t('asset_create_success'))
      navigate(`/assets/${values.name}/edit`)
    } catch {
      message.error(t('asset_create_failed'))
    } finally {
      setSaving(false)
    }
  }

  const renderCategoryFields = () => {
    switch (selectedCategory) {
      case 'eval':
        return (
          <Form.Item label={t('create_test_cases')} name="test_cases">
            <TextArea rows={10} style={{ width: 600, fontFamily: 'monospace' }} placeholder="test_cases:&#10;  - id: case1&#10;    name: Test Case 1&#10;    input: |&#10;      ...&#10;    expected: |&#10;      ..." />
          </Form.Item>
        )
      case 'metric':
        return (
          <Form.Item label={t('create_rubric')} name="rubric">
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
          title={t('create_title')}
          extra={
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/assets')}>
              {t('create_back')}
            </Button>
          }
        >
          <Form
            form={form}
            layout="vertical"
            onFinish={handleCreate}
            initialValues={{ state: 'draft', asset_type: defaultAssetType, category: 'content' }}
          >
            <Form.Item
              label={t('create_category')}
              name="category"
              rules={[{ required: true, message: t('create_select_category') }]}
              tooltip={t('create_category_tooltip')}
            >
              <Select
                options={categoryOptions}
                style={{ width: 200 }}
                onChange={(value) => setSelectedCategory(value)}
              />
            </Form.Item>

            <Form.Item
              label={t('create_asset_name')}
              name="name"
              rules={[
                { required: true, message: t('create_enter_asset_name') },
                { pattern: /^[a-zA-Z0-9_-]+$/, message: t('create_asset_name_pattern') },
              ]}
              extra={t('create_asset_name_hint')}
            >
              <Input placeholder="e.g., customer-churn-scorer" style={{ width: 400 }} />
            </Form.Item>

            <Form.Item
              label={t('create_asset_type')}
              name="asset_type"
              rules={[{ required: true, message: t('create_select_biz_line') }]}
            >
              <Select options={assetTypeOptions} style={{ width: 200 }} />
            </Form.Item>

            <Form.Item label={t('create_tags')} name="tags">
              <Select mode="multiple" options={tagOptions} style={{ width: 400 }} placeholder={t('create_tags_placeholder')} />
            </Form.Item>

            <Form.Item label={t('create_state')} name="state">
              <Select options={stateOptions} style={{ width: 200 }} />
            </Form.Item>

            <Form.Item
              label={t('create_description')}
              name="description"
              tooltip={t('create_description_tooltip')}
            >
              <TextArea rows={3} style={{ width: 600 }} placeholder={t('create_description_placeholder')} />
            </Form.Item>

            {renderCategoryFields()}

            <Form.Item>
              <Space>
                <Button type="primary" icon={<SaveOutlined />} htmlType="submit" loading={saving}>
                  {t('create_submit')}
                </Button>
                <Button onClick={() => navigate('/assets')}>{t('common_cancel')}</Button>
              </Space>
            </Form.Item>
          </Form>
        </Card>
      </Space>
    </div>
  )
}

export default CreateAssetView
