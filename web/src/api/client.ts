import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

export interface AssetSummary {
  id: string
  name: string
  description: string
  biz_line: string
  tags: string[]
  state: string
}

export interface AssetDetail extends AssetSummary {
  labels: Record<string, string>
  snapshots: Snapshot[]
}

export interface Snapshot {
  version: string
  commit_hash: string
  author: string
  reason: string
  eval_score?: number
  created_at: string
}

export interface EvalRun {
  id: string
  status: string
  deterministic_score: number
  rubric_score: number
  created_at: string
}

export interface EvalReport {
  run_id: string
  status: string
  overall_score: number
  deterministic_score: number
  rubric_score: number
  rubric_details: RubricDetail[]
  duration_ms: number
}

export interface RubricDetail {
  check_id: string
  passed: boolean
  score: number
  details: string
}

export interface CompareResult {
  asset_id: string
  version1: string
  version2: string
  score_delta: number
  passed_delta: number
}

export const assetApi = {
  list: async (filters?: { biz_line?: string; tag?: string }): Promise<AssetSummary[]> => {
    const params = new URLSearchParams()
    if (filters?.biz_line) params.append('biz_line', filters.biz_line)
    if (filters?.tag) params.append('tag', filters.tag)
    const { data } = await api.get(`/assets?${params}`)
    return data.assets
  },

  get: async (id: string): Promise<AssetDetail> => {
    const { data } = await api.get(`/assets/${id}`)
    return data
  },

  create: async (asset: { id: string; name: string; description?: string }): Promise<void> => {
    await api.post('/assets', asset)
  },

  update: async (id: string, updates: Partial<AssetDetail>): Promise<void> => {
    await api.put(`/assets/${id}`, updates)
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/assets/${id}`)
  },
}

export const evalApi = {
  run: async (assetId: string, snapshotVersion?: string, caseIds?: string[]): Promise<{ run_id: string; status: string }> => {
    const { data } = await api.post('/evals/run', {
      asset_id: assetId,
      snapshot_version: snapshotVersion || 'latest',
      eval_case_ids: caseIds,
    })
    return data
  },

  get: async (runId: string): Promise<EvalRun> => {
    const { data } = await api.get(`/evals/${runId}`)
    return data
  },

  report: async (runId: string): Promise<EvalReport> => {
    const { data } = await api.get(`/evals/${runId}/report`)
    return data
  },

  compare: async (assetId: string, v1: string, v2: string): Promise<CompareResult> => {
    const { data } = await api.post('/evals/compare', {
      asset_id: assetId,
      version1: v1,
      version2: v2,
    })
    return data
  },

  diagnose: async (runId: string): Promise<any> => {
    const { data } = await api.get(`/evals/${runId}/diagnose`)
    return data
  },

  list: async (assetId: string): Promise<EvalRun[]> => {
    const { data } = await api.get(`/evals?asset_id=${assetId}`)
    return data.runs
  },
}

export const triggerApi = {
  match: async (input: string, top = 5): Promise<{ matches: any[]; total: number }> => {
    const { data } = await api.post('/trigger/match', { input, top })
    return data
  },

  validate: async (prompt: string): Promise<{ valid: boolean; message?: string }> => {
    const { data } = await api.post('/trigger/validate', { prompt })
    return data
  },

  inject: async (prompt: string, variables: Record<string, string>): Promise<{ result: string }> => {
    const { data } = await api.post('/trigger/inject', { prompt, variables })
    return data
  },
}

export default api
