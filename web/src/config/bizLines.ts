// Shared biz_line configuration
// Fetched from API on init, cached in localStorage for synchronous access

import axios from 'axios'

export interface BizLineConfig {
  name: string
  description: string
  color: string
  built_in?: boolean
}

const STORAGE_KEY = 'ep_biz_lines'

const DEFAULT_BIZ_LINES: BizLineConfig[] = [
  { name: 'tech', description: '技术研发', color: 'blue', built_in: true },
  { name: 'opinion', description: '舆情业务', color: 'red', built_in: true },
  { name: 'marketing', description: '营销业务', color: 'green', built_in: true },
  { name: 'content', description: '内容创作', color: 'purple', built_in: true },
]

let cachedBizLines: BizLineConfig[] | null = null

export function getBizLines(): BizLineConfig[] {
  if (cachedBizLines) return cachedBizLines
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      cachedBizLines = JSON.parse(stored)
      return cachedBizLines!
    }
  } catch {}
  cachedBizLines = DEFAULT_BIZ_LINES
  return cachedBizLines
}

export async function loadBizLinesFromAPI(): Promise<void> {
  try {
    const { data } = await axios.get<{ biz_lines: BizLineConfig[] }>('/api/v1/taxonomy')
    cachedBizLines = data.biz_lines || DEFAULT_BIZ_LINES
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cachedBizLines))
  } catch {
    // fallback to cached or default
    if (!cachedBizLines) {
      cachedBizLines = DEFAULT_BIZ_LINES
    }
  }
}

export async function saveBizLinesToAPI(bizLines: BizLineConfig[]): Promise<void> {
  cachedBizLines = bizLines
  localStorage.setItem(STORAGE_KEY, JSON.stringify(bizLines))
  await axios.put('/api/v1/taxonomy/biz_lines', bizLines)
}

export function getBizLineColor(name: string): string {
  const bizLines = getBizLines()
  const found = bizLines.find((b) => b.name === name)
  return found?.color || 'default'
}
