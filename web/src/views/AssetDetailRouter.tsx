import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Spin } from 'antd'
import { useTranslation } from 'react-i18next'
import { assetApi } from '../api/client'
import type { AssetDetail } from '../api/client'
import ContentDetailView from './ContentDetailView'
import EvalCasesView from './EvalCasesView'
import MetricDetailView from './MetricDetailView'

function AssetDetailRouter() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const [asset, setAsset] = useState<AssetDetail | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (id) {
      loadAsset(id)
    }
  }, [id])

  const loadAsset = async (assetId: string) => {
    setLoading(true)
    try {
      const data = await assetApi.get(assetId)
      setAsset(data)
    } catch {
      setAsset(null)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  if (!asset) {
    return <div>{t('asset_detail_not_found')}</div>
  }

  const category = asset.category || 'content'

  switch (category) {
    case 'eval':
      return <EvalCasesView asset={asset} />
    case 'metric':
      return <MetricDetailView asset={asset} />
    case 'content':
    default:
      return <ContentDetailView asset={asset} />
  }
}

export default AssetDetailRouter
