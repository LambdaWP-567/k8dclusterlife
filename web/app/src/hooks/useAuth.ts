import { useQuery } from '@tanstack/react-query'
import type { User } from '../types'

async function fetchMe(): Promise<User | null> {
  const res = await fetch('/api/me')
  if (res.status === 401) return null
  if (!res.ok) throw new Error('Failed to fetch user')
  return res.json()
}

export function useAuth() {
  const { data: user, isLoading } = useQuery<User | null>({
    queryKey: ['me'],
    queryFn: fetchMe,
    retry: false,
    staleTime: 5 * 60 * 1000,
  })
  return { user: user ?? null, isLoading }
}
