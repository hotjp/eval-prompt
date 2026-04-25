// Shared asset_type configuration
// Fetched from API on init, cached in localStorage for synchronous access

import axios from 'axios'

export interface AssetTypeConfig {
  name: string
  description: string
  color: string
  built_in?: boolean
}

const STORAGE_KEY = 'ep_asset_types'

const DEFAULT_BIZ_LINES: AssetTypeConfig[] = [
  { name: 'prompt', description: '提示词', color: 'blue', built_in: true },
  { name: 'agent', description: 'Agent 描述文件', color: 'purple', built_in: true },
  { name: 'skill', description: 'Skill', color: 'green', built_in: true },
  { name: 'knowledge', description: '知识库', color: 'orange', built_in: true },
  { name: 'system', description: '系统配置', color: 'red', built_in: true },
  { name: 'workflow', description: '工作流', color: 'cyan', built_in: true },
  { name: 'tool', description: '工具描述', color: 'geekblue', built_in: true },
]

let cachedAssetTypes: AssetTypeConfig[] | null = null
let loadingAssetTypes = false

export function getAssetTypes(): AssetTypeConfig[] {
  if (cachedAssetTypes) return cachedAssetTypes
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      cachedAssetTypes = JSON.parse(stored)
      return cachedAssetTypes!
    }
  } catch {}
  cachedAssetTypes = DEFAULT_BIZ_LINES
  return cachedAssetTypes
}

export async function loadAssetTypesFromAPI(): Promise<void> {
  if (loadingAssetTypes) return
  loadingAssetTypes = true
  try {
    const { data } = await axios.get<{ asset_types: AssetTypeConfig[] }>('/api/v1/taxonomy')
    cachedAssetTypes = data.asset_types || DEFAULT_BIZ_LINES
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cachedAssetTypes))
  } catch {
    // fallback to cached or default
    if (!cachedAssetTypes) {
      cachedAssetTypes = DEFAULT_BIZ_LINES
    }
  } finally {
    loadingAssetTypes = false
  }
}

export async function saveAssetTypesToAPI(bizLines: AssetTypeConfig[]): Promise<void> {
  cachedAssetTypes = bizLines
  localStorage.setItem(STORAGE_KEY, JSON.stringify(bizLines))
  await axios.put('/api/v1/taxonomy/asset_types', bizLines)
}

export function getAssetTypeColor(name: string): string {
  const bizLines = getAssetTypes()
  const found = bizLines.find((b) => b.name === name)
  return found?.color || 'default'
}
