// Shared tag configuration
// Fetched from API on init, cached in localStorage for synchronous access

import axios from 'axios'

export interface TagConfig {
  name: string
  color: string
  built_in?: boolean
}

const STORAGE_KEY = 'ep_tags'

const DEFAULT_TAGS: TagConfig[] = [
  { name: 'prod', color: 'green', built_in: true },
  { name: 'draft', color: 'orange', built_in: true },
  { name: 'llm', color: 'blue', built_in: true },
  { name: 'rag', color: 'purple', built_in: true },
  { name: 'agent', color: 'cyan', built_in: true },
  { name: 'security', color: 'red', built_in: true },
  { name: 'ops', color: 'geekblue', built_in: true },
  { name: 'go', color: 'lime', built_in: true },
  { name: 'review', color: 'gold', built_in: true },
]

let cachedTags: TagConfig[] | null = null

export function getTags(): TagConfig[] {
  if (cachedTags) return cachedTags
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      cachedTags = JSON.parse(stored)
      return cachedTags!
    }
  } catch {}
  cachedTags = DEFAULT_TAGS
  return cachedTags
}

export async function loadTagsFromAPI(): Promise<void> {
  try {
    const { data } = await axios.get<{ tags: TagConfig[] }>('/api/v1/taxonomy')
    cachedTags = data.tags || DEFAULT_TAGS
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cachedTags))
  } catch {
    if (!cachedTags) {
      cachedTags = DEFAULT_TAGS
    }
  }
}

export async function saveTagsToAPI(tags: TagConfig[]): Promise<void> {
  cachedTags = tags
  localStorage.setItem(STORAGE_KEY, JSON.stringify(tags))
  await axios.put('/api/v1/taxonomy/tags', tags)
}

export function getTagColor(name: string): string {
  const tags = getTags()
  const found = tags.find((t) => t.name === name)
  return found?.color || 'blue'
}
