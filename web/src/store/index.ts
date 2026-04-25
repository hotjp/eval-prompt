import { create } from 'zustand'

export interface Asset {
  id: string
  name: string
  description: string
  bizLine: string
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

interface AppState {
  assets: Asset[]
  currentAsset: Asset | null
  snapshots: Snapshot[]
  evalRuns: EvalRun[]
  matchedPrompts: MatchedPrompt[]
  loading: boolean
  runningEval: { id: string; assetId: string; assetName: string } | null
  evalConcurrency: number
  showInitRepoModal: boolean

  setAssets: (assets: Asset[]) => void
  setCurrentAsset: (asset: Asset | null) => void
  setSnapshots: (snapshots: Snapshot[]) => void
  setEvalRuns: (runs: EvalRun[]) => void
  setMatchedPrompts: (prompts: MatchedPrompt[]) => void
  setLoading: (loading: boolean) => void
  setRunningEval: (data: { id: string; assetId: string; assetName: string } | null) => void
  setEvalConcurrency: (concurrency: number) => void
  setShowInitRepoModal: (show: boolean) => void
}

export const useStore = create<AppState>((set) => ({
  assets: [],
  currentAsset: null,
  snapshots: [],
  evalRuns: [],
  matchedPrompts: [],
  loading: false,
  runningEval: null,
  evalConcurrency: 1,
  showInitRepoModal: false,

  setAssets: (assets) => set({ assets }),
  setCurrentAsset: (currentAsset) => set({ currentAsset }),
  setSnapshots: (snapshots) => set({ snapshots }),
  setEvalRuns: (evalRuns) => set({ evalRuns }),
  setMatchedPrompts: (matchedPrompts) => set({ matchedPrompts }),
  setLoading: (loading) => set({ loading }),
  setRunningEval: (runningEval) => set({ runningEval }),
  setEvalConcurrency: (evalConcurrency) => set({ evalConcurrency }),
  setShowInitRepoModal: (showInitRepoModal) => set({ showInitRepoModal }),
}))
