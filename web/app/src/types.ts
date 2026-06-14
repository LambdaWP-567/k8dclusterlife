export type Severity = 'critical' | 'warning' | 'info'

export interface Problem {
  id: string
  cluster_id: string
  cluster_name: string
  kind: string
  name: string
  namespace: string
  status: string
  description: string
  cause: string
  severity: Severity
  detected_at: string
}

export interface WSEvent {
  type: 'problem_added' | 'problem_resolved' | 'cluster_status'
  cluster: string
  problem?: Problem
  at: string
}

export interface User {
  sub: string
  email: string
  name: string
  provider: string
}
