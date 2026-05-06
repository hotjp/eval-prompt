import { useEffect, useRef } from 'react'
import { useStore } from '../../../store'
import { executionApi } from '../../../api/client'

const ACTIVE_POLL_INTERVAL = 2000
const IDLE_POLL_INTERVAL = 10000

export function useExecutionPolling() {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isMountedRef = useRef(true)
  const isPollingRef = useRef(false)

  useEffect(() => {
    isMountedRef.current = true

    const tick = async () => {
      if (!isMountedRef.current || isPollingRef.current) return

      const runningEvals = useStore.getState().runningEvals
      const activeEvals = runningEvals.filter(
        (e) => e.status === 'pending' || e.status === 'running' || e.status === 'cancelling'
      )

      if (activeEvals.length > 0) {
        isPollingRef.current = true
        await Promise.all(
          activeEvals.map(async (re) => {
            try {
              const exec = await executionApi.get(re.id)
              const progress =
                exec.total_cases > 0
                  ? { completed: exec.completed_cases, total: exec.total_cases }
                  : undefined

              const statusMap: Record<string, 'pending' | 'running' | 'completed' | 'failed' | 'cancelling' | 'cancelled'> = {
                pending: 'pending',
                initializing: 'pending',
                running: 'running',
                completed: 'completed',
                passed: 'completed',
                failed: 'failed',
                cancelled: 'cancelled',
                cancelling: 'cancelling',
              }

              const mappedStatus = statusMap[exec.status] || 'running'

              useStore.getState().updateRunningEval(re.id, {
                status: mappedStatus,
                progress,
              })
            } catch {
              useStore.getState().updateRunningEval(re.id, { status: 'failed' })
            }
          })
        )
        isPollingRef.current = false
      }

      // Schedule next tick: fast if there are active evals, slower if idle
      const nextInterval = activeEvals.length > 0 ? ACTIVE_POLL_INTERVAL : IDLE_POLL_INTERVAL
      timerRef.current = setTimeout(tick, nextInterval)
    }

    tick()

    return () => {
      isMountedRef.current = false
      if (timerRef.current) {
        clearTimeout(timerRef.current)
        timerRef.current = null
      }
    }
  }, [])
}
