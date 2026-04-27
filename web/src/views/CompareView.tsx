import { useState, useEffect } from 'react'
import { Card, Select, Row, Col, Statistic, Tag, Table, message, Button, Space, Modal } from 'antd'
import { ArrowUpOutlined, ArrowDownOutlined, SwapOutlined, DiffOutlined } from '@ant-design/icons'
import { assetApi, evalApi, llmApi } from '../api/client'
import type { AssetSummary, AssetDetail, CompareResult, Snapshot } from '../api/client'
import { useTranslation } from 'react-i18next'

function CompareView() {
  const { t } = useTranslation()
  const [assets, setAssets] = useState<AssetSummary[]>([])
  const [selectedAsset, setSelectedAsset] = useState<string>('')
  const [assetDetail, setAssetDetail] = useState<AssetDetail | null>(null)
  const [version1, setVersion1] = useState<string>('')
  const [version2, setVersion2] = useState<string>('')
  const [compareResult, setCompareResult] = useState<CompareResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [assetLoading, setAssetLoading] = useState(false)
  const [semanticDiffResult, setSemanticDiffResult] = useState<{ summary: string; changes: string; impact: string } | null>(null)
  const [semanticDiffLoading, setSemanticDiffLoading] = useState(false)
  const [showSemanticDiffModal, setShowSemanticDiffModal] = useState(false)

  const loadAssets = async () => {
    setAssetLoading(true)
    try {
      const data = await assetApi.list()
      setAssets(data.assets)
    } catch {
      message.error('Failed to load assets')
    } finally {
      setAssetLoading(false)
    }
  }

  useEffect(() => {
    loadAssets()
  }, [])

  // Load asset detail when asset is selected to get snapshots
  useEffect(() => {
    if (!selectedAsset) {
      setAssetDetail(null)
      setVersion1('')
      setVersion2('')
      return
    }
    assetApi.get(selectedAsset).then(setAssetDetail).catch(() => {
      message.error('Failed to load asset details')
      setAssetDetail(null)
    })
  }, [selectedAsset])

  const handleCompare = async () => {
    if (!selectedAsset || !version1 || !version2) {
      message.warning('Please select asset and both versions')
      return
    }
    if (version1 === version2) {
      message.warning('Please select two different versions')
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

  const handleSemanticDiff = async () => {
    if (!selectedAsset || !version1 || !version2) {
      message.warning('Please select asset and both versions')
      return
    }
    if (version1 === version2) {
      message.warning('Please select two different versions')
      return
    }
    if (!compareResult?.diff_output) {
      message.warning('Please run Compare first to get the git diff')
      return
    }
    setSemanticDiffLoading(true)
    try {
      // Use git diff as old_content; semantic diff will analyze the textual changes
      const result = await llmApi.diff(compareResult.diff_output, '', version1, version2)
      setSemanticDiffResult(result)
      setShowSemanticDiffModal(true)
    } catch (err: any) {
      if (err?.response?.status === 503) {
        message.warning(t('compare_llm_not_configured'))
      } else {
        message.error(t('compare_semantic_diff_failed'))
      }
    } finally {
      setSemanticDiffLoading(false)
    }
  }

  // Get version options from snapshots
  const versionOptions = (assetDetail?.snapshots || [])
    .map((s: Snapshot) => ({ label: s.version, value: s.version }))
    .sort((a, b) => a.label.localeCompare(b.label))

  // Get snapshot data for display
  const getSnapshotData = (version: string): Snapshot | undefined => {
    return (assetDetail?.snapshots || []).find((s: Snapshot) => s.version === version)
  }

  return (
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card title={t('compare_title')}>
          <Space wrap>
            <Select
              placeholder={t('compare_select_asset')}
              style={{ width: 200 }}
              loading={assetLoading}
              value={selectedAsset || undefined}
              onChange={(val) => {
                setSelectedAsset(val)
                setCompareResult(null)
              }}
              options={assets.map((a) => ({ label: a.name, value: a.id }))}
            />
            <Select
              placeholder={t('compare_version_1')}
              style={{ width: 150 }}
              value={version1 || undefined}
              onChange={setVersion1}
              options={versionOptions}
              disabled={versionOptions.length === 0}
            />
            <Select
              placeholder={t('compare_version_2')}
              style={{ width: 150 }}
              value={version2 || undefined}
              onChange={setVersion2}
              options={versionOptions}
              disabled={versionOptions.length === 0}
            />
            <Button
              type="primary"
              icon={<SwapOutlined />}
              onClick={handleCompare}
              loading={loading}
              disabled={!selectedAsset || versionOptions.length < 2}
            >
              {t('compare_compare')}
            </Button>
            <Button
              icon={<DiffOutlined />}
              onClick={handleSemanticDiff}
              loading={semanticDiffLoading}
              disabled={!selectedAsset || versionOptions.length < 2}
            >
              {t('compare_semantic_diff')}
            </Button>
          </Space>
          {versionOptions.length === 0 && selectedAsset && (
            <div style={{ marginTop: 8, color: '#888' }}>
              {t('compare_no_history')}
            </div>
          )}
        </Card>

        {compareResult && (
          <Card title={t('compare_result')}>
            <Row gutter={16}>
              <Col span={12}>
                <Card size="small">
                  <Statistic
                    title={`${t('compare_score_delta')} (${version1} → ${version2})`}
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
                    title={t('compare_pass_rate_delta')}
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

        {compareResult && assetDetail && (
          <Card title={t('compare_change_summary')}>
            <Table
              dataSource={[
                {
                  metric: t('compare_eval_score'),
                  v1: getSnapshotData(version1)?.eval_score?.toFixed(2) ?? 'N/A',
                  v2: getSnapshotData(version2)?.eval_score?.toFixed(2) ?? 'N/A',
                  delta: compareResult.score_delta.toFixed(3),
                },
              ]}
              rowKey="metric"
              size="small"
              pagination={false}
              columns={[
                { title: t('compare_metric'), dataIndex: 'metric', key: 'metric' },
                { title: version1, dataIndex: 'v1', key: 'v1' },
                { title: version2, dataIndex: 'v2', key: 'v2' },
                {
                  title: t('compare_delta'),
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

      <Modal
        title={t('compare_semantic_diff_modal')}
        open={showSemanticDiffModal}
        onCancel={() => setShowSemanticDiffModal(false)}
        footer={null}
        width={600}
      >
        {semanticDiffResult && (
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <Card size="small" title={t('compare_summary')}>
              <p>{semanticDiffResult.summary}</p>
            </Card>
            <Card size="small" title={t('compare_changes')}>
              <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{semanticDiffResult.changes}</pre>
            </Card>
            <Card size="small" title={t('compare_impact')}>
              <p>{semanticDiffResult.impact}</p>
            </Card>
          </Space>
        )}
      </Modal>
    </div>
  )
}

export default CompareView
