import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

interface Settings {
  autonomy_mode: string
  countdown_seconds: string
  refresh_interval_s: string
  teams_webhook_url: string
  claude_model: string
}

interface SSOProvider {
  provider: string
  enabled: boolean
  client_id: string
  client_secret: string
  tenant_id?: string
}

const SECRET_MASK = '••••••••'

const PROVIDER_LABELS: Record<string, string> = {
  entra: 'Microsoft Entra ID (Azure AD)',
  github: 'GitHub',
  google: 'Google',
}

async function fetchSettings(): Promise<Settings> {
  const res = await fetch('/api/settings')
  if (!res.ok) throw new Error('Failed to load settings')
  return res.json()
}

async function saveSettings(s: Partial<Settings>): Promise<void> {
  const res = await fetch('/api/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(s),
  })
  if (!res.ok) throw new Error('Failed to save')
}

async function fetchSSO(): Promise<SSOProvider[]> {
  const res = await fetch('/api/sso')
  if (!res.ok) {
    if (res.status === 404) return []
    throw new Error('Failed to load SSO config')
  }
  return res.json()
}

async function saveSSO(provider: string, config: Partial<SSOProvider>): Promise<void> {
  const res = await fetch(`/api/sso/${provider}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error((err as { error?: string }).error ?? 'Failed to save SSO config')
  }
}

function SSOProviderCard({ provider, onSaved }: { provider: SSOProvider; onSaved: () => void }) {
  const [form, setForm] = useState<SSOProvider>({ ...provider })
  const [saved, setSaved] = useState(false)

  const mutation = useMutation({
    mutationFn: () => saveSSO(provider.provider, form),
    onSuccess: () => {
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      onSaved()
    },
  })

  function handleChange(key: keyof SSOProvider, value: string | boolean) {
    setForm(f => ({ ...f, [key]: value }))
    setSaved(false)
  }

  const isEntra = provider.provider === 'entra'

  return (
    <div className="p-5 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-medium text-gray-900 dark:text-white">
          {PROVIDER_LABELS[provider.provider] ?? provider.provider}
        </h3>
        <label className="flex items-center gap-2 cursor-pointer select-none">
          <span className="text-sm text-gray-600 dark:text-gray-400">
            {form.enabled ? 'Aktiviert' : 'Deaktiviert'}
          </span>
          <button
            type="button"
            onClick={() => handleChange('enabled', !form.enabled)}
            className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
              form.enabled ? 'bg-violet-600' : 'bg-gray-300 dark:bg-gray-600'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
                form.enabled ? 'translate-x-4' : 'translate-x-0.5'
              }`}
            />
          </button>
        </label>
      </div>

      <div className="grid gap-3">
        <div>
          <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1 uppercase tracking-wide">
            Client ID
          </label>
          <input
            type="text"
            placeholder="Application / Client ID"
            value={form.client_id}
            onChange={e => handleChange('client_id', e.target.value)}
            className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 font-mono"
          />
        </div>

        <div>
          <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1 uppercase tracking-wide">
            Client Secret
          </label>
          <input
            type="password"
            placeholder={form.client_secret === SECRET_MASK ? 'Leer lassen um unverändert zu lassen' : 'Client Secret eingeben'}
            value={form.client_secret === SECRET_MASK ? '' : form.client_secret}
            onChange={e => handleChange('client_secret', e.target.value || SECRET_MASK)}
            onFocus={e => {
              if (form.client_secret === SECRET_MASK) {
                handleChange('client_secret', '')
                e.target.placeholder = 'Neues Secret eingeben'
              }
            }}
            className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 font-mono"
          />
          {provider.client_secret === SECRET_MASK && (
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Secret ist gesetzt — leer lassen um es beizubehalten
            </p>
          )}
        </div>

        {isEntra && (
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1 uppercase tracking-wide">
              Tenant ID
            </label>
            <input
              type="text"
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
              value={form.tenant_id ?? ''}
              onChange={e => handleChange('tenant_id', e.target.value)}
              className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 font-mono"
            />
          </div>
        )}
      </div>

      <div className="flex items-center gap-3 pt-1">
        <button
          type="button"
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending}
          className="rounded-lg bg-violet-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-50"
        >
          {mutation.isPending ? 'Speichern…' : 'Speichern'}
        </button>
        {saved && (
          <span className="text-sm text-green-600 dark:text-green-400">✓ Gespeichert</span>
        )}
        {mutation.isError && (
          <span className="text-sm text-red-600 dark:text-red-400">
            {mutation.error instanceof Error ? mutation.error.message : 'Fehler'}
          </span>
        )}
      </div>
    </div>
  )
}

export function Settings() {
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['settings'], queryFn: fetchSettings })
  const { data: ssoProviders } = useQuery({ queryKey: ['sso'], queryFn: fetchSSO })
  const [form, setForm] = useState<Partial<Settings>>({})
  const [saved, setSaved] = useState(false)

  const mutation = useMutation({
    mutationFn: saveSettings,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['settings'] })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    },
  })

  const merged = { ...data, ...form }

  function handleChange(key: keyof Settings, value: string) {
    setForm(f => ({ ...f, [key]: value }))
    setSaved(false)
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    mutation.mutate(merged)
  }

  if (isLoading) return <div className="text-sm text-gray-500 dark:text-gray-400">Laden…</div>

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Einstellungen</h1>

      <form onSubmit={handleSubmit} className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">

        {/* KI Autonomy */}
        <div className="p-5 space-y-4">
          <h2 className="font-medium text-gray-900 dark:text-white">KI-Heilung</h2>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Autonomie-Modus
            </label>
            <select
              value={merged.autonomy_mode ?? 'MANUAL'}
              onChange={e => handleChange('autonomy_mode', e.target.value)}
              className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            >
              <option value="MANUAL">MANUAL — Jede Aktion manuell bestätigen</option>
              <option value="COUNTDOWN">COUNTDOWN — Automatisch nach Timer</option>
              <option value="FULL_AUTO">FULL_AUTO — Vollautomatisch</option>
            </select>
          </div>

          {merged.autonomy_mode === 'COUNTDOWN' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Countdown-Dauer (Sekunden)
              </label>
              <input
                type="number"
                min={10}
                max={300}
                value={merged.countdown_seconds ?? '60'}
                onChange={e => handleChange('countdown_seconds', e.target.value)}
                className="w-32 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
              />
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Claude Modell
            </label>
            <select
              value={merged.claude_model ?? 'claude-sonnet-4-6'}
              onChange={e => handleChange('claude_model', e.target.value)}
              className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            >
              <option value="claude-sonnet-4-6">claude-sonnet-4-6 (Empfohlen)</option>
              <option value="claude-opus-4-8">claude-opus-4-8 (Leistungsstärker)</option>
              <option value="claude-haiku-4-5-20251001">claude-haiku-4-5-20251001 (Schneller)</option>
            </select>
          </div>
        </div>

        {/* Monitoring */}
        <div className="p-5 space-y-4">
          <h2 className="font-medium text-gray-900 dark:text-white">Überwachung</h2>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Abfrageintervall (Sekunden)
            </label>
            <input
              type="number"
              min={10}
              max={300}
              value={merged.refresh_interval_s ?? '30'}
              onChange={e => handleChange('refresh_interval_s', e.target.value)}
              className="w-32 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            />
          </div>
        </div>

        {/* Notifications */}
        <div className="p-5 space-y-4">
          <h2 className="font-medium text-gray-900 dark:text-white">Benachrichtigungen</h2>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Microsoft Teams Webhook URL
            </label>
            <input
              type="url"
              placeholder="https://outlook.office.com/webhook/…"
              value={merged.teams_webhook_url ?? ''}
              onChange={e => handleChange('teams_webhook_url', e.target.value)}
              className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            />
          </div>
        </div>

        {/* Save */}
        <div className="p-5 flex items-center gap-4">
          <button
            type="submit"
            disabled={mutation.isPending}
            className="rounded-lg bg-violet-600 px-5 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-50"
          >
            {mutation.isPending ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && (
            <span className="text-sm text-green-600 dark:text-green-400">✓ Gespeichert</span>
          )}
          {mutation.isError && (
            <span className="text-sm text-red-600 dark:text-red-400">Fehler beim Speichern</span>
          )}
        </div>
      </form>

      {/* SSO Configuration */}
      {ssoProviders && ssoProviders.length > 0 && (
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
            SSO-Anmeldung
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
            Konfiguriere externe Anmeldeanbieter. Aktivierte Anbieter erscheinen auf der Login-Seite.
            Die Änderungen werden sofort ohne Neustart wirksam.
          </p>
          <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
            {ssoProviders.map(p => (
              <SSOProviderCard
                key={p.provider}
                provider={p}
                onSaved={() => qc.invalidateQueries({ queryKey: ['sso'] })}
              />
            ))}
          </div>
        </div>
      )}

      {!ssoProviders || ssoProviders.length === 0 && (
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 p-5 text-sm text-gray-500 dark:text-gray-400">
          SSO-Konfiguration erfordert eine Datenbankverbindung (DATABASE_URL).
        </div>
      )}
    </div>
  )
}
