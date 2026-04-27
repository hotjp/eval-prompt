import { useEffect, useRef } from 'react'
import { useStore } from '../../../store'
import { executionApi } from '../../../api/client'

const POLL_INTERVAL = 2000

export function useExecutionPolling() {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isMountedRef = useRef(true)

  useEffect(() => {
    isMountedRef.current = true

    const tick = async () => {
      if (!isMountedRef.current) return

      const runningEvals = useStore.getState().runningEvals
      const activeEvals = runningEvals.filter(
        (e) => e.status === 'pending' || e.status === 'running' || e.status === 'cancelling'
      )

      if (activeEvals.length > 0) {
        await Promise.all(
          activeEvals.map(async (re) => {
            try {
              const exec = await executionApi.get(re.id)
              const progress =
                exec.total_cases > 0
                  ? { completed: exec.completed_cases, total: exec.total_cases }
                  : undefined

              const statusMap: Record<string, 'pending' | 'running' | 'completed' | 'failed' | 'cancelling'> = {
                pending: 'pending',
                initializing: 'pending',
                running: 'running',
                completed: 'completed',
                passed: 'completed',
                failed: 'failed',
                cancelled: 'failed',
                cancelling: 'cancelling',
              }

              const mappedStatus = statusMap[exec.status] || 'running'

              useStore.getState().updateRunningEval(re.id, {
                status: mappedStatus,
                progress,
              })

              if (mappedStatus === 'completed' || mappedStatus === 'failed') {
                setTimeout(() => {
                  useStore.getState().removeRunningEval(re.id)
                }, 5000)
              }
            } catch {
              useStore.getState().updateRunningEval(re.id, { status: 'failed' })
            }
          })
        )
      }

      timerRef.current = setTimeout(tick, POLL_INTERVAL)
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
