import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import type { Problem, WSEvent } from '../types'

async function fetchProblems(): Promise<Problem[]> {
  const res = await fetch('/api/problems')
  if (!res.ok) throw new Error('Failed to fetch problems')
  return res.json()
}

export function useProblems() {
  const queryClient = useQueryClient()

  const query = useQuery({
    queryKey: ['problems'],
    queryFn: fetchProblems,
    refetchInterval: 30_000,
    staleTime: 10_000,
  })

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws`)

    ws.onmessage = (e) => {
      try {
        const event: WSEvent = JSON.parse(e.data)
        if (event.type === 'cluster_status') {
          queryClient.invalidateQueries({ queryKey: ['problems'] })
        }
      } catch {}
    }

    ws.onerror = () => {}

    return () => ws.close()
  }, [queryClient])

  return query
}
