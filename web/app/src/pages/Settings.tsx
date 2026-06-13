import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

interface Settings {
  autonomy_mode: string
  countdown_seconds: string
  refresh_interval_s: string
  teams_webhook_url: string
  claude_model: string
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

export function Settings() {
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['settings'], queryFn: fetchSettings })
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
    </div>
  )
}
