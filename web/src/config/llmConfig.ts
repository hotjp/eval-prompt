// LLM provider configuration
// Fetched from API on init, cached in localStorage for offline access

import axios from 'axios'

export interface LLMConfig {
  name: string
  provider: string // openai | claude | ollama
  api_key: string
  endpoint?: string
  default_model: string
}

const STORAGE_KEY = 'ep_llm_configs'

let cachedLLMConfigs: LLMConfig[] | null = null

export function getLLMConfigs(): LLMConfig[] {
  if (cachedLLMConfigs) return cachedLLMConfigs
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      cachedLLMConfigs = JSON.parse(stored)
      return cachedLLMConfigs!
    }
  } catch {}
  cachedLLMConfigs = []
  return cachedLLMConfigs
}

export async function loadLLMConfigsFromAPI(): Promise<void> {
  try {
    const { data } = await axios.get<LLMConfig[]>('/api/v1/llm-config')
    cachedLLMConfigs = Array.isArray(data) ? data : []
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cachedLLMConfigs))
  } catch {
    // fallback to cached or empty
    if (!cachedLLMConfigs) {
      cachedLLMConfigs = []
    }
  }
}

export async function saveLLMConfigsToAPI(configs: LLMConfig[]): Promise<void> {
  cachedLLMConfigs = configs
  localStorage.setItem(STORAGE_KEY, JSON.stringify(configs))
  await axios.put('/api/v1/llm-config', configs)
}
