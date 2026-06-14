import type { Severity } from '../types'

interface Props {
  status: string
  severity: Severity
  description: string
}

const severityClasses: Record<Severity, string> = {
  critical: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  warning: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  info: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
}

export function StatusBadge({ status, severity, description }: Props) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium cursor-help ${severityClasses[severity]}`}
      title={description}
    >
      {status}
    </span>
  )
}
