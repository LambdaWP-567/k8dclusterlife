import { useState } from 'react'
import { useProblems } from '../hooks/useProblems'
import { ProblemCard } from '../components/ProblemCard'
import { HealingStream } from '../components/HealingStream'
import type { Problem, Severity } from '../types'

const severityOrder: Record<Severity, number> = { critical: 0, warning: 1, info: 2 }

function sortProblems(problems: Problem[]): Problem[] {
  return [...problems].sort((a, b) => {
    const diff = severityOrder[a.severity] - severityOrder[b.severity]
    if (diff !== 0) return diff
    return new Date(b.detected_at).getTime() - new Date(a.detected_at).getTime()
  })
}

export function Dashboard() {
  const { data: problems, isLoading, isError } = useProblems()
  const [healingProblem, setHealingProblem] = useState<Problem | null>(null)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-violet-600 border-t-transparent" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="rounded-lg bg-red-50 dark:bg-red-900/20 p-4 text-red-700 dark:text-red-400">
        Fehler beim Laden der Cluster-Daten. Backend erreichbar?
      </div>
    )
  }

  const sorted = sortProblems(problems ?? [])
  const criticalCount = sorted.filter(p => p.severity === 'critical').length

  return (
    <div className="space-y-4">
      {/* Status Banner */}
      {sorted.length === 0 ? (
        <div className="rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 p-6 text-center">
          <svg className="mx-auto h-12 w-12 text-green-500" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <h2 className="mt-2 text-lg font-semibold text-green-800 dark:text-green-300">
            Alle Cluster gesund
          </h2>
          <p className="mt-1 text-sm text-green-600 dark:text-green-400">
            Keine Probleme erkannt.
          </p>
        </div>
      ) : (
        <div className="flex items-center justify-between rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 px-4 py-3">
          <div className="flex items-center gap-2">
            <svg className="h-5 w-5 text-red-500" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            <span className="font-semibold text-red-800 dark:text-red-300">
              {sorted.length} Problem{sorted.length !== 1 ? 'e' : ''} erkannt
              {criticalCount > 0 && ` (${criticalCount} kritisch)`}
            </span>
          </div>
        </div>
      )}

      {/* Problem List */}
      <div className="space-y-3">
        {sorted.map(problem => (
          <ProblemCard
            key={problem.id}
            problem={problem}
            onHeal={setHealingProblem}
          />
        ))}
      </div>

      {/* KI-Healing Modal */}
      {healingProblem && (
        <HealingStream
          problem={healingProblem}
          onClose={() => setHealingProblem(null)}
        />
      )}
    </div>
  )
}
