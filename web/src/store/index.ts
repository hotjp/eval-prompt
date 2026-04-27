import { create } from 'zustand'

export interface Asset {
  id: string
  name: string
  description: string
  assetType: string
  tags: string[]
  state: string
}

export interface Snapshot {
  version: string
  commitHash: string
  author: string
  reason: string
  evalScore?: number
  createdAt: string
}

export interface EvalRun {
  id: string
  status: 'pending' | 'running' | 'passed' | 'failed'
  deterministicScore: number
  rubricScore: number
  createdAt: string
}

export interface MatchedPrompt {
  assetId: string
  name: string
  description: string
  relevance: number
}

export interface RunningEval {
  id: string
  assetId: string
  assetName: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelling'
  progress?: { completed: number; total: number }
  startedAt: number
}

interface AppState {
  assets: Asset[]
  currentAsset: Asset | null
  snapshots: Snapshot[]
  evalRuns: EvalRun[]
  matchedPrompts: MatchedPrompt[]
  loading: boolean
  runningEvals: RunningEval[]
  evalConcurrency: number
  showInitRepoModal: boolean
  initRepoModalReason: 'create' | 'api_error' | 'manual' | null

  setAssets: (assets: Asset[]) => void
  setCurrentAsset: (asset: Asset | null) => void
  setSnapshots: (snapshots: Snapshot[]) => void
  setEvalRuns: (runs: EvalRun[]) => void
  setMatchedPrompts: (prompts: MatchedPrompt[]) => void
  setLoading: (loading: boolean) => void
  addRunningEval: (evalItem: RunningEval) => void
  updateRunningEval: (id: string, patch: Partial<RunningEval>) => void
  removeRunningEval: (id: string) => void
  setEvalConcurrency: (concurrency: number) => void
  setShowInitRepoModal: (show: boolean, reason?: 'create' | 'api_error' | 'manual') => void
}

export const useStore = create<AppState>((set) => ({
  assets: [],
  currentAsset: null,
  snapshots: [],
  evalRuns: [],
  matchedPrompts: [],
  loading: false,
  runningEvals: [],
  evalConcurrency: 1,
  showInitRepoModal: false,
  initRepoModalReason: null,

  setAssets: (assets) => set({ assets }),
  setCurrentAsset: (currentAsset) => set({ currentAsset }),
  setSnapshots: (snapshots) => set({ snapshots }),
  setEvalRuns: (evalRuns) => set({ evalRuns }),
  setMatchedPrompts: (matchedPrompts) => set({ matchedPrompts }),
  setLoading: (loading) => set({ loading }),
  addRunningEval: (evalItem) =>
    set((state) => ({ runningEvals: [...state.runningEvals, evalItem] })),
  updateRunningEval: (id, patch) =>
    set((state) => ({
      runningEvals: state.runningEvals.map((e) => (e.id === id ? { ...e, ...patch } : e)),
    })),
  removeRunningEval: (id) =>
    set((state) => ({ runningEvals: state.runningEvals.filter((e) => e.id !== id) })),
  setEvalConcurrency: (evalConcurrency) => set({ evalConcurrency }),
  setShowInitRepoModal: (show, reason) => set({ showInitRepoModal: show, initRepoModalReason: reason ?? null }),
}))
