import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

const noPrefixApi = axios.create({
  timeout: 10000,
})

// LLM API uses longer timeout (60s) for streaming/complex operations
const llmAxios = axios.create({
  baseURL: '/api/v1',
  timeout: 60000,
})

export type AssetCategory = 'content' | 'eval' | 'metric'

export interface AssetSummary {
  id: string
  name: string
  description: string
  asset_type: string
  tags: string[]
  state: string
  latest_score?: number
  category?: AssetCategory
  test_cases?: TestCase[]
  rubric?: RubricItem[]
}

export interface AssetDetail extends AssetSummary {
  labels: Record<string, string>
  snapshots: Snapshot[]
  category?: AssetCategory
  eval_history?: EvalHistoryEntry[]
  eval_stats?: Record<string, EvalStats>
  triggers?: TriggerEntry[]
  test_cases?: TestCase[]
  recommended_snapshot_id?: string
  rubric?: RubricItem[]
  metric_refs?: string[]
  used_by?: string[]
}

export interface Snapshot {
  version: string
  commit_hash: string
  author: string
  reason: string
  eval_score?: number
  created_at: string
}

export interface EvalHistoryEntry {
  run_id: string
  snapshot_id: string
  score?: number
  deterministic_score?: number
  rubric_score?: number
  status: string
  model?: string
  tokens_in?: number
  tokens_out?: number
  latency_ms?: number
  created_at: string
  commit_hash?: string
  author?: string
}

export interface EvalStats {
  count: number
  mean: number
  stddev: number
  min: number
  max: number
}

export interface TriggerEntry {
  id: string
  pattern: string
  description?: string
}

export interface TestCase {
  id: string
  name: string
  description?: string
  input: string
  expected?: string
  rubric?: string
}

export interface RubricItem {
  check: string
  weight: number
  criteria: string
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
  diff_output?: string
}

export interface HealthStatus {
  status: string
  checks?: Record<string, { status: string; message?: string; providers?: Record<string, { status: string; latency_ms?: number; message?: string }> }>
}

export interface GitInfo {
  branch: string
  dirty: boolean
  short_commit: string
  remote: string
}

export interface LLMConfig {
  name: string
  provider: string
  api_key: string
  endpoint?: string
  default_model: string
  default?: boolean
}

export interface ExecuteEvalRequest {
  asset_id: string
  case_ids?: string[]
  mode?: string
  runs_per_case?: number
  concurrency?: number
  model?: string
  temperature?: number
}

export interface Execution {
  id: string
  asset_id: string
  status: string
  concurrency: number
  model: string
  temperature: number
  runs_per_case: number
  total_cases: number
  completed_cases: number
  created_at: string
  updated_at: string
}

export interface LLMCall {
  id: string
  run_id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  model: string
  temperature: number
  prompt_content: string
  response_content: string
  raw_json?: string
  tokens_in?: number
  tokens_out?: number
  latency_ms?: number
  error?: string
  created_at: string
}

export interface ExecutionListResponse {
  executions: Execution[]
  total: number
}

export interface RepoConfig {
  repo_path: string
  assets_dir: string
  evals_dir: string
}

export interface RepoInfo {
  path: string
  status: string // "valid" | "notfound" | "notgit"
}

export interface RepoListResponse {
  repos: RepoInfo[]
  current: string
}

export const healthApi = {
  check: async (): Promise<HealthStatus> => {
    const { data } = await noPrefixApi.get('/readyz')
    return data
  },
}

export const adminApi = {
  gitInfo: async (): Promise<GitInfo> => {
    const { data } = await api.get('/admin/git-info')
    return data
  },
  getRepoConfig: async (): Promise<RepoConfig> => {
    const { data } = await api.get('/admin/repo-config')
    return data
  },
  saveRepoConfig: async (config: RepoConfig): Promise<void> => {
    await api.put('/admin/repo-config', config)
  },
  getRepoList: async (): Promise<RepoListResponse> => {
    const { data } = await api.get('/admin/repo-list')
    return data
  },
  switchRepo: async (path: string): Promise<{ status: string; path: string }> => {
    const { data } = await api.put('/admin/repo-switch', { path })
    return data
  },
  getFirstUse: async (): Promise<{ first_use: boolean }> => {
    const { data } = await api.get('/admin/first-use')
    return data
  },
  getRepoStatus: async (): Promise<{
    current?: { path: string; valid: boolean; branch?: string; dirty?: boolean; short_commit?: string; error?: string; outside_home?: boolean }
    repos: { path: string; status: string }[]
    is_first_use: boolean
  }> => {
    const { data } = await api.get('/admin/repo-status')
    return data
  },
  reconcile: async (): Promise<{ added: number; updated: number; deleted: number; errors: string[] }> => {
    const { data } = await api.post('/admin/reconcile')
    return data
  },
  gitPull: async (): Promise<{ status: string; message: string }> => {
    const { data } = await api.post('/admin/git-pull')
    return data
  },
  openFolder: async (): Promise<{ status: string; message: string }> => {
    const { data } = await api.post('/admin/open-folder')
    return data
  },
  saveConfig: async (config: Record<string, any>): Promise<void> => {
    await api.put('/admin/config', config)
  },
}

export interface AssetListResponse {
  assets: AssetSummary[]
  total: number
}

export const assetApi = {
  list: async (filters?: { asset_type?: string; tag?: string; category?: string }): Promise<AssetListResponse> => {
    const params = new URLSearchParams()
    if (filters?.asset_type) params.append('asset_type', filters.asset_type)
    if (filters?.tag) params.append('tag', filters.tag)
    if (filters?.category) params.append('category', filters.category)
    const { data } = await api.get(`/assets?${params}`)
    return { assets: data.assets || [], total: data.total || 0 }
  },

  get: async (id: string): Promise<AssetDetail> => {
    const { data } = await api.get(`/assets/${id}`)
    return data
  },

  create: async (asset: { id: string; name: string; description?: string; asset_type?: string; tags?: string[]; content?: string; test_cases?: string; rubric?: string; category?: string }): Promise<void> => {
    await api.post('/assets', asset)
  },

  update: async (id: string, updates: Partial<AssetDetail>): Promise<void> => {
    await api.put(`/assets/${id}`, updates)
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/assets/${id}`)
  },

  archive: async (id: string): Promise<void> => {
    await api.post(`/assets/${id}/archive`)
  },

  restore: async (id: string): Promise<void> => {
    await api.post(`/assets/${id}/restore`)
  },

  getContent: async (id: string): Promise<{ id: string; content: string; content_hash: string; updated_at: string }> => {
    const { data } = await api.get(`/assets/${id}/content`)
    return data
  },

  saveContent: async (id: string, content: string, commitMessage?: string, contentHash?: string): Promise<{ id: string; content: string; content_hash: string; updated_at: string; message: string }> => {
    const { data } = await api.put(`/assets/${id}/content`, { content, commit_message: commitMessage, content_hash: contentHash })
    return data
  },

  commit: async (id: string, message?: string): Promise<{ id: string; commit: string; message: string }> => {
    const params = message ? `?message=${encodeURIComponent(message)}` : ''
    const { data } = await api.post(`/assets/${id}/commit${params}`)
    return data
  },

  commitBatch: async (ids: string[], message?: string): Promise<{ commits: Record<string, string>; message: string }> => {
    const params = message ? `?message=${encodeURIComponent(message)}` : ''
    const { data } = await api.post(`/assets/commit${params}`, { ids })
    return data
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

  execute: async (request: ExecuteEvalRequest): Promise<{ execution_id: string; status: string }> => {
    const { data } = await api.post('/evals/execute', request)
    return data
  },

  getExecution: async (executionId: string): Promise<Execution> => {
    const { data } = await api.get(`/executions/${executionId}`)
    return data
  },

  cancelExecution: async (executionId: string): Promise<void> => {
    await api.post(`/executions/${executionId}/cancel`)
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
    return data.runs || []
  },
}

export const executionApi = {
  list: async (filters?: { status?: string }): Promise<ExecutionListResponse> => {
    const params = new URLSearchParams()
    if (filters?.status) params.append('status', filters.status)
    const { data } = await api.get(`/executions?${params}`)
    return { executions: data.executions || [], total: data.total || 0 }
  },

  get: async (executionId: string): Promise<Execution> => {
    const { data } = await api.get(`/executions/${executionId}`)
    return data
  },

  getCalls: async (executionId: string): Promise<LLMCall[]> => {
    const { data } = await api.get(`/executions/${executionId}/calls`)
    return data.calls || []
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

export const llmConfigApi = {
  get: async (): Promise<LLMConfig[]> => {
    const { data } = await api.get('/llm-config')
    return Array.isArray(data) ? data : (data?.configs || [])
  },

  save: async (configs: LLMConfig[]): Promise<void> => {
    await api.put('/llm-config', configs)
  },

  testByName: async (name: string, message?: string): Promise<{ success: boolean; content?: string; error?: string }> => {
    const { data } = await api.post('/llm-config/test-by-name', { name, message })
    return data
  },
}

export const llmApi = {
  rewrite: async (content: string, instruction: string, modelName?: string) => {
    const { data } = await llmAxios.post('/rewrite', { content, instruction, model_name: modelName })
    return data
  },
  diff: async (oldContent: string, newContent: string, oldVersion: string, newVersion: string) => {
    const { data } = await llmAxios.post('/eval/diff', { old_content: oldContent, new_content: newContent, old_version: oldVersion, new_version: newVersion })
    return data
  },
}

export default api
