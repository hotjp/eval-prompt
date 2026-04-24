import { useState } from 'react'
import { Card, Select, Row, Col, Statistic, Tag, Table, message, Button, Space } from 'antd'
import { ArrowUpOutlined, ArrowDownOutlined, SwapOutlined } from '@ant-design/icons'
import { assetApi, evalApi } from '../api/client'
import type { AssetSummary, CompareResult } from '../api/client'

function CompareView() {
  const [assets, setAssets] = useState<AssetSummary[]>([])
  const [selectedAsset, setSelectedAsset] = useState<string>('')
  const [version1, setVersion1] = useState<string>('')
  const [version2, setVersion2] = useState<string>('')
  const [compareResult, setCompareResult] = useState<CompareResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [assetLoading, setAssetLoading] = useState(false)

  useState(() => {
    loadAssets()
  })

  const loadAssets = async () => {
    setAssetLoading(true)
    try {
      const data = await assetApi.list()
      setAssets(data)
    } catch {
      message.error('Failed to load assets')
    } finally {
      setAssetLoading(false)
    }
  }

  const handleCompare = async () => {
    if (!selectedAsset || !version1 || !version2) {
      message.warning('Please select asset and both versions')
      return
    }
    setLoading(true)
    try {
      const result = await evalApi.compare(selectedAsset, version1, version2)
      setCompareResult(result)
    } catch {
      message.error('Compare failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card title="Compare Versions">
          <Space wrap>
            <Select
              placeholder="Select Asset"
              style={{ width: 200 }}
              loading={assetLoading}
              onChange={setSelectedAsset}
              options={assets.map((a) => ({ label: a.name, value: a.id }))}
            />
            <Select
              placeholder="Version 1"
              style={{ width: 150 }}
              onChange={setVersion1}
              options={[
                { label: 'v1.0.0', value: 'v1.0.0' },
                { label: 'v1.1.0', value: 'v1.1.0' },
                { label: 'v1.2.0', value: 'v1.2.0' },
              ]}
            />
            <Select
              placeholder="Version 2"
              style={{ width: 150 }}
              onChange={setVersion2}
              options={[
                { label: 'v1.0.0', value: 'v1.0.0' },
                { label: 'v1.1.0', value: 'v1.1.0' },
                { label: 'v1.2.0', value: 'v1.2.0' },
              ]}
            />
            <Button type="primary" icon={<SwapOutlined />} onClick={handleCompare} loading={loading}>
              Compare
            </Button>
          </Space>
        </Card>

        {compareResult && (
          <Card title="Comparison Result">
            <Row gutter={16}>
              <Col span={12}>
                <Card size="small">
                  <Statistic
                    title={`Score Delta (${version1} → ${version2})`}
                    value={compareResult.score_delta}
                    precision={3}
                    prefix={
                      compareResult.score_delta > 0 ? (
                        <ArrowUpOutlined style={{ color: 'green' }} />
                      ) : compareResult.score_delta < 0 ? (
                        <ArrowDownOutlined style={{ color: 'red' }} />
                      ) : null
                    }
                    valueStyle={{
                      color: compareResult.score_delta > 0 ? 'green' : compareResult.score_delta < 0 ? 'red' : 'black',
                    }}
                  />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic
                    title="Pass Rate Delta"
                    value={compareResult.passed_delta}
                    precision={0}
                    prefix={
                      compareResult.passed_delta > 0 ? (
                        <ArrowUpOutlined style={{ color: 'green' }} />
                      ) : compareResult.passed_delta < 0 ? (
                        <ArrowDownOutlined style={{ color: 'red' }} />
                      ) : null
                    }
                    suffix="checks"
                    valueStyle={{
                      color:
                        compareResult.passed_delta > 0 ? 'green' : compareResult.passed_delta < 0 ? 'red' : 'black',
                    }}
                  />
                </Card>
              </Col>
            </Row>
          </Card>
        )}

        {compareResult && (
          <Card title="Change Summary">
            <Table
              dataSource={[
                { metric: 'Overall Score', v1: '0.85', v2: (0.85 + compareResult.score_delta).toFixed(2), delta: compareResult.score_delta.toFixed(3) },
                { metric: 'Passed Checks', v1: '12', v2: (12 + compareResult.passed_delta).toString(), delta: compareResult.passed_delta > 0 ? `+${compareResult.passed_delta}` : compareResult.passed_delta.toString() },
              ]}
              rowKey="metric"
              size="small"
              pagination={false}
              columns={[
                { title: 'Metric', dataIndex: 'metric', key: 'metric' },
                { title: version1, dataIndex: 'v1', key: 'v1' },
                { title: version2, dataIndex: 'v2', key: 'v2' },
                {
                  title: 'Delta',
                  dataIndex: 'delta',
                  key: 'delta',
                  render: (delta) => (
                    <Tag color={parseFloat(delta) > 0 ? 'green' : parseFloat(delta) < 0 ? 'red' : 'gray'}>
                      {delta}
                    </Tag>
                  ),
                },
              ]}
            />
          </Card>
        )}
      </Space>
    </div>
  )
}

export default CompareView
