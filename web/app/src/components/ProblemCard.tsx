import type { Problem } from '../types'
import { StatusBadge } from './StatusBadge'
import { SeverityIcon } from './SeverityIcon'

interface Props {
  problem: Problem
  onHeal?: (problem: Problem) => void
}

const kindIcons: Record<string, string> = {
  Pod: '📦',
  Node: '🖥️',
  Deployment: '🚀',
  StatefulSet: '🗄️',
  DaemonSet: '👾',
  Job: '⚙️',
  CronJob: '🕐',
  PVC: '💾',
  Certificate: '🔐',
  HelmRelease: '⛵',
  LonghornVolume: '💿',
  CephCluster: '🪨',
  CephOSD: '🪨',
  VolumeAttachment: '🔗',
  CSIDriver: '💽',
  LinstorController: '🔷',
  LinstorSatellite: '🔷',
}

export function ProblemCard({ problem, onHeal }: Props) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 shadow-sm hover:shadow-md transition-shadow">
      <div className="flex items-start gap-3">
        <SeverityIcon severity={problem.severity} />

        <div className="flex-1 min-w-0">
          {/* Header row */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm text-gray-500 dark:text-gray-400">
              {kindIcons[problem.kind] ?? '❓'} {problem.kind}
            </span>
            <span className="font-semibold text-gray-900 dark:text-gray-100 truncate">
              {problem.name}
            </span>
            {problem.namespace && (
              <span className="text-xs text-gray-400 dark:text-gray-500">
                ns/{problem.namespace}
              </span>
            )}
            <StatusBadge
              status={problem.status}
              severity={problem.severity}
              description={problem.description}
            />
          </div>

          {/* Cluster badge */}
          <div className="mt-1">
            <span className="inline-flex items-center text-xs text-gray-500 dark:text-gray-400">
              <svg className="mr-1 h-3 w-3" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" />
              </svg>
              {problem.cluster_name}
            </span>
          </div>

          {/* Description */}
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-300">
            {problem.description}
          </p>

          {/* Cause */}
          <div className="mt-1 flex items-start gap-1">
            <span className="text-xs text-gray-400 dark:text-gray-500 shrink-0 mt-0.5">Ursache:</span>
            <span className="text-xs text-gray-500 dark:text-gray-400">{problem.cause}</span>
          </div>
        </div>

        {/* KI-Heilung button (placeholder until Commit 9) */}
        {onHeal && (
          <button
            onClick={() => onHeal(problem)}
            className="shrink-0 rounded-md bg-violet-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-violet-500 focus:outline-none focus:ring-2 focus:ring-violet-600"
          >
            KI-Heilung
          </button>
        )}
      </div>
    </div>
  )
}
