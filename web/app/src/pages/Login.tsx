interface LoginPageProps {
  availableProviders: string[]
}

const providerMeta: Record<string, { label: string; icon: string; color: string }> = {
  entra: {
    label: 'Anmelden mit Microsoft',
    icon: '🏢',
    color: 'bg-blue-600 hover:bg-blue-700',
  },
  github: {
    label: 'Anmelden mit GitHub',
    icon: '🐙',
    color: 'bg-gray-800 hover:bg-gray-900 dark:bg-gray-700 dark:hover:bg-gray-600',
  },
  google: {
    label: 'Anmelden mit Google',
    icon: '🔍',
    color: 'bg-red-600 hover:bg-red-700',
  },
}

export function Login({ availableProviders }: LoginPageProps) {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="w-full max-w-sm bg-white dark:bg-gray-800 rounded-2xl shadow-lg p-8 space-y-6">
        <div className="text-center space-y-2">
          <div className="text-4xl">☸️</div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">k8dclusterlife</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Kubernetes Cluster Überwachung &amp; KI-Heilung
          </p>
        </div>

        <div className="space-y-3">
          {availableProviders.length === 0 ? (
            <p className="text-center text-sm text-gray-500 dark:text-gray-400">
              Keine Auth-Provider konfiguriert — bitte Umgebungsvariablen setzen.
            </p>
          ) : (
            availableProviders.map((provider) => {
              const meta = providerMeta[provider]
              if (!meta) return null
              return (
                <a
                  key={provider}
                  href={`/auth/${provider}/login`}
                  className={`flex items-center justify-center gap-3 w-full py-3 px-4 rounded-lg text-white font-medium transition-colors ${meta.color}`}
                >
                  <span className="text-xl">{meta.icon}</span>
                  {meta.label}
                </a>
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}
