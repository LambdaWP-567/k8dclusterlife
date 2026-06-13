import { useState, useEffect, useRef } from 'react'
import type { Problem } from '../types'

interface StreamEvent {
  type: 'text' | 'tool_call' | 'tool_result' | 'approval_required' | 'done' | 'error'
  content?: string
  tool?: string
  input?: string
  output?: string
  action_id?: string
}

interface HealingSession {
  id: string
  status: 'running' | 'healed' | 'failed' | 'aborted'
  result?: string
  dialog: Array<{ role: string; content: string; at: string }>
  actions: Array<{ id: string; tool_name: string; tool_input: string; output: string }>
}

interface Props {
  problem: Problem
  onClose: () => void
}

export function HealingStream({ problem, onClose }: Props) {
  const [session, setSession] = useState<HealingSession | null>(null)
  const [events, setEvents] = useState<StreamEvent[]>([])
  const [pendingApproval, setPendingApproval] = useState<StreamEvent | null>(null)
  const [isStarting, setIsStarting] = useState(false)
  const [error, setError] = useState('')
  const eventSourceRef = useRef<EventSource | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [events])

  async function startHealing() {
    setIsStarting(true)
    setError('')
    try {
      const res = await fetch('/api/healing', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ problem, autonomy_mode: 'MANUAL' }),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: 'Unbekannter Fehler' }))
        throw new Error(body.error)
      }
      const sess: HealingSession = await res.json()
      setSession(sess)

      // Connect to SSE stream
      const es = new EventSource(`/api/healing/${sess.id}/stream`)
      eventSourceRef.current = es

      es.onmessage = (e) => {
        const ev: StreamEvent = JSON.parse(e.data)
        if (ev.type === 'approval_required') {
          setPendingApproval(ev)
        } else if (ev.type === 'done') {
          setSession(s => s ? { ...s, status: 'healed', result: ev.content } : s)
          es.close()
        } else if (ev.type === 'error') {
          setError(ev.content || 'Fehler')
          es.close()
        } else {
          setEvents(prev => [...prev, ev])
        }
      }
      es.onerror = () => es.close()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setIsStarting(false)
    }
  }

  async function handleApproval(approved: boolean) {
    if (!session) return
    setPendingApproval(null)
    await fetch(`/api/healing/${session.id}/approve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ approved }),
    })
  }

  useEffect(() => {
    return () => eventSourceRef.current?.close()
  }, [])

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-2xl mx-4 bg-white dark:bg-gray-900 rounded-2xl shadow-2xl flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-200 dark:border-gray-700">
          <div>
            <h2 className="font-semibold text-gray-900 dark:text-white">KI-Heilung</h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {problem.kind} · {problem.name}
              {problem.namespace && ` · ${problem.namespace}`}
            </p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200">
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-5 space-y-3 font-mono text-sm">
          {!session && !isStarting && (
            <div className="text-center space-y-3 py-6">
              <p className="text-gray-600 dark:text-gray-300">
                Claude wird das Problem analysieren und automatisch Heilungsschritte vorschlagen.
              </p>
              <p className="text-xs text-gray-500">Problem: {problem.description}</p>
              {error && <p className="text-red-500">{error}</p>}
            </div>
          )}

          {isStarting && (
            <div className="text-center py-6 text-gray-500">Claude analysiert…</div>
          )}

          {events.map((ev, i) => (
            <div key={i}>
              {ev.type === 'text' && (
                <p className="text-gray-800 dark:text-gray-200 whitespace-pre-wrap leading-relaxed">{ev.content}</p>
              )}
              {ev.type === 'tool_call' && (
                <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg px-3 py-2 text-blue-700 dark:text-blue-300">
                  ▶ {ev.tool}({ev.input})
                </div>
              )}
              {ev.type === 'tool_result' && (
                <div className="bg-gray-50 dark:bg-gray-800 rounded-lg px-3 py-2 text-gray-600 dark:text-gray-400 text-xs whitespace-pre-wrap">
                  {ev.output}
                </div>
              )}
            </div>
          ))}

          {session?.status === 'healed' && (
            <div className="bg-green-50 dark:bg-green-900/20 rounded-lg px-3 py-2 text-green-700 dark:text-green-300">
              ✓ {session.result || 'Heilung abgeschlossen'}
            </div>
          )}

          {session?.status === 'failed' && (
            <div className="bg-red-50 dark:bg-red-900/20 rounded-lg px-3 py-2 text-red-700 dark:text-red-300">
              ✗ {session.result}
            </div>
          )}

          <div ref={scrollRef} />
        </div>

        {/* Approval Banner */}
        {pendingApproval && (
          <div className="border-t border-yellow-200 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-900/20 px-5 py-4">
            <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200 mb-1">
              Aktion genehmigen?
            </p>
            <code className="text-xs text-yellow-700 dark:text-yellow-300 block mb-3">
              {pendingApproval.tool}({pendingApproval.input})
            </code>
            <div className="flex gap-2">
              <button
                onClick={() => handleApproval(true)}
                className="rounded-lg bg-green-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-green-700"
              >
                Genehmigen
              </button>
              <button
                onClick={() => handleApproval(false)}
                className="rounded-lg bg-gray-200 dark:bg-gray-700 px-4 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-300"
              >
                Ablehnen
              </button>
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="border-t border-gray-200 dark:border-gray-700 px-5 py-3 flex justify-between items-center">
          {!session && !isStarting && (
            <button
              onClick={startHealing}
              className="rounded-lg bg-violet-600 px-5 py-2 text-sm font-medium text-white hover:bg-violet-700"
            >
              KI-Heilung starten
            </button>
          )}
          {session?.status === 'running' && (
            <span className="text-sm text-gray-500 dark:text-gray-400 animate-pulse">Claude analysiert…</span>
          )}
          <button onClick={onClose} className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 ml-auto">
            Schließen
          </button>
        </div>
      </div>
    </div>
  )
}
