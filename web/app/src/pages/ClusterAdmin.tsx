import { useState, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

interface Cluster {
  id: string
  name: string
  reachable: boolean
  created_at: string
  last_seen_at?: string
}

async function fetchClusters(): Promise<Cluster[]> {
  const res = await fetch('/api/clusters')
  if (!res.ok) throw new Error('Failed to load clusters')
  return res.json()
}

async function deleteCluster(id: string): Promise<void> {
  const res = await fetch(`/api/clusters/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete failed')
}

async function createCluster(name: string, file: File): Promise<void> {
  const form = new FormData()
  form.append('name', name)
  form.append('kubeconfig', file)
  const res = await fetch('/api/clusters', { method: 'POST', body: form })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(body.error || 'Create failed')
  }
}

export function ClusterAdmin() {
  const qc = useQueryClient()
  const { data: clusters = [], isLoading } = useQuery({ queryKey: ['clusters'], queryFn: fetchClusters })
  const [name, setName] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [formError, setFormError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)

  const addMutation = useMutation({
    mutationFn: () => createCluster(name, file!),
    onSuccess: () => {
      setName('')
      setFile(null)
      setFormError('')
      if (fileRef.current) fileRef.current.value = ''
      qc.invalidateQueries({ queryKey: ['clusters'] })
    },
    onError: (e: Error) => setFormError(e.message),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteCluster,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['clusters'] }),
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) { setFormError('Name erforderlich'); return }
    if (!file) { setFormError('Kubeconfig-Datei erforderlich'); return }
    setFormError('')
    addMutation.mutate()
  }

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Cluster-Verwaltung</h1>

      {/* Add cluster form */}
      <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
        <h2 className="font-medium text-gray-900 dark:text-white mb-4">Cluster hinzufügen</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="z.B. production"
              className="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              kubeconfig Datei
            </label>
            <input
              ref={fileRef}
              type="file"
              accept=".yaml,.yml,.json,.conf"
              onChange={e => setFile(e.target.files?.[0] ?? null)}
              className="block w-full text-sm text-gray-500 dark:text-gray-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-violet-50 file:text-violet-700 dark:file:bg-violet-900/30 dark:file:text-violet-300"
            />
          </div>
          {formError && (
            <p className="text-sm text-red-600 dark:text-red-400">{formError}</p>
          )}
          <button
            type="submit"
            disabled={addMutation.isPending}
            className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-50"
          >
            {addMutation.isPending ? 'Wird hinzugefügt…' : 'Cluster hinzufügen'}
          </button>
        </form>
      </div>

      {/* Cluster list */}
      <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
        {isLoading ? (
          <div className="p-5 text-sm text-gray-500 dark:text-gray-400">Laden…</div>
        ) : clusters.length === 0 ? (
          <div className="p-5 text-sm text-gray-500 dark:text-gray-400">
            Noch kein Cluster hinzugefügt.
          </div>
        ) : (
          clusters.map(c => (
            <div key={c.id} className="flex items-center justify-between p-4">
              <div className="flex items-center gap-3">
                <span className={`h-2.5 w-2.5 rounded-full flex-shrink-0 ${c.reachable ? 'bg-green-500' : 'bg-gray-400'}`} />
                <div>
                  <p className="font-medium text-gray-900 dark:text-white">{c.name}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {c.reachable ? 'Verbunden' : 'Nicht erreichbar'} · Hinzugefügt {new Date(c.created_at).toLocaleDateString('de-DE')}
                  </p>
                </div>
              </div>
              <button
                onClick={() => deleteMutation.mutate(c.id)}
                disabled={deleteMutation.isPending}
                className="text-sm text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 disabled:opacity-50"
              >
                Entfernen
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
