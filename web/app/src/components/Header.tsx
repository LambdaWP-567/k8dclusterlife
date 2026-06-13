import { useProblems } from '../hooks/useProblems'
import type { User } from '../types'

interface Props {
  darkMode: boolean
  onToggleDark: () => void
  user?: User | null
}

export function Header({ darkMode, onToggleDark, user }: Props) {
  const { data: problems } = useProblems()
  const criticalCount = problems?.filter(p => p.severity === 'critical').length ?? 0
  const totalCount = problems?.length ?? 0

  return (
    <header className="sticky top-0 z-10 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
      <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-14 items-center justify-between">
          <div className="flex items-center gap-3">
            <svg className="h-7 w-7 text-violet-600" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" />
            </svg>
            <span className="text-lg font-semibold text-gray-900 dark:text-white">
              k8dclusterlife
            </span>
          </div>

          <div className="flex items-center gap-3">
            {/* Problem counter badge */}
            {totalCount > 0 && (
              <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-bold ${
                criticalCount > 0
                  ? 'bg-red-600 text-white'
                  : 'bg-yellow-500 text-white'
              }`}>
                {totalCount}
              </span>
            )}

            {/* Dark mode toggle */}
            <button
              onClick={onToggleDark}
              className="rounded-md p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              title={darkMode ? 'Light Mode' : 'Dark Mode'}
            >
              {darkMode ? (
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591-1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z" />
                </svg>
              ) : (
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z" />
                </svg>
              )}
            </button>

            {/* User avatar + logout */}
            {user && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-600 dark:text-gray-300 hidden sm:block">
                  {user.name || user.email}
                </span>
                <a
                  href="/auth/logout"
                  className="rounded-md px-2.5 py-1.5 text-xs font-medium text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400 border border-gray-200 dark:border-gray-700"
                >
                  Abmelden
                </a>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  )
}
