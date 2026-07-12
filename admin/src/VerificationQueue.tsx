import { useCallback, useEffect, useState } from 'react'
import {
  adminApi,
  clearToken,
  type QueueItem,
  type SubmissionDetail,
} from './api'

type Props = { onLogout?: () => void; embedded?: boolean }

export default function VerificationQueue({ onLogout, embedded = false }: Props) {
  const [pending, setPending] = useState(0)
  const [activeTrips, setActiveTrips] = useState(0)
  const [items, setItems] = useState<QueueItem[]>([])
  const [selected, setSelected] = useState<SubmissionDetail | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setError('')
    try {
      const [stats, queue] = await Promise.all([adminApi.stats(), adminApi.queue()])
      setPending(stats.pending_verifications)
      setActiveTrips(stats.active_trips ?? 0)
      setItems(queue.items)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  async function openDetail(id: string) {
    setLoading(true)
    try {
      setSelected(await adminApi.detail(id))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed')
    } finally {
      setLoading(false)
    }
  }

  async function approve(id: string) {
    setLoading(true)
    try {
      await adminApi.approve(id)
      setSelected(null)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Approve failed')
    } finally {
      setLoading(false)
    }
  }

  async function reject(id: string) {
    setLoading(true)
    try {
      await adminApi.reject(id, rejectReason || 'Rejected by admin')
      setSelected(null)
      setRejectReason('')
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Reject failed')
    } finally {
      setLoading(false)
    }
  }

  function logout() {
    clearToken()
    onLogout?.()
  }

  const content = (
    <div className={embedded ? 'grid gap-6 lg:grid-cols-2' : 'mx-auto grid max-w-6xl gap-6 p-6 lg:grid-cols-2'}>
      <section>
        <div className="mb-4 grid grid-cols-2 gap-3">
          <div className="rounded-2xl bg-kdblack p-5 text-kdwhite">
            <p className="text-sm text-kdgrey">Pending verifications</p>
            <p className="text-3xl font-bold text-gold">{pending}</p>
          </div>
          <div className="rounded-2xl border border-kdborder bg-kdwhite p-5">
            <p className="text-sm text-kdgrey">Active trips</p>
            <p className="text-3xl font-bold text-charcoal">{activeTrips}</p>
          </div>
        </div>

          {error && <p className="mb-3 text-sm text-red-600">{error}</p>}

          <div className="space-y-3">
            {items.length === 0 && (
              <p className="rounded-xl border border-kdborder bg-kdwhite p-4 text-kdgrey">
                No pending submissions.
              </p>
            )}
            {items.map((item) => (
              <button
                key={item.submission_id}
                type="button"
                onClick={() => openDetail(item.submission_id)}
                className="w-full rounded-xl border border-kdborder bg-kdwhite p-4 text-left hover:border-gold"
              >
                <p className="font-semibold">{item.legal_name || item.phone}</p>
                <p className="text-sm text-kdgrey">
                  {item.ride_type} · {item.plate} · {item.phone}
                </p>
                <p className="mt-1 text-xs text-kdgrey">{item.submitted_at}</p>
              </button>
            ))}
          </div>
        </section>

        <section className="rounded-2xl border border-kdborder bg-kdwhite p-6">
          {!selected ? (
            <p className="text-kdgrey">Select a submission to review documents.</p>
          ) : (
            <>
              <h2 className="text-lg font-semibold">{selected.legal_name}</h2>
              <p className="text-sm text-kdgrey">
                {selected.ride_type} · Plate {selected.plate} · NIN …{selected.nin_last4}
              </p>
              <p className="mt-1 text-sm">Permit: {selected.permit_number}</p>
              <p className="text-sm text-kdgrey">Phone: {selected.phone}</p>

              <div className="mt-4 grid grid-cols-2 gap-3">
                {selected.documents.map((doc) => (
                  <a
                    key={doc.id}
                    href={adminApi.fileUrl(doc.storage_key)}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-lg border border-kdborder p-2 text-center text-xs hover:border-gold"
                  >
                    <div className="font-medium">{doc.doc_type}</div>
                    {doc.mime_type.startsWith('image/') ? (
                      <img
                        src={adminApi.fileUrl(doc.storage_key)}
                        alt={doc.doc_type}
                        className="mt-2 max-h-24 w-full rounded object-cover"
                      />
                    ) : (
                      <span className="mt-2 block text-kdgrey">View file</span>
                    )}
                  </a>
                ))}
              </div>

              <textarea
                className="mt-4 w-full rounded-lg border border-kdborder p-2 text-sm"
                placeholder="Rejection reason (optional)"
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
              />

              <div className="mt-4 flex gap-3">
                <button
                  type="button"
                  disabled={loading}
                  onClick={() => approve(selected.submission_id)}
                  className="flex-1 rounded-xl bg-gold py-2.5 font-semibold text-charcoal disabled:opacity-50"
                >
                  Approve
                </button>
                <button
                  type="button"
                  disabled={loading}
                  onClick={() => reject(selected.submission_id)}
                  className="flex-1 rounded-xl border border-kdborder py-2.5 font-semibold disabled:opacity-50"
                >
                  Reject
                </button>
              </div>
            </>
          )}
        </section>
      </div>
  )

  if (embedded) return content

  return (
    <div className="min-h-screen bg-cream">
      <header className="flex items-center justify-between border-b border-kdborder bg-kdwhite px-6 py-4">
        <div>
          <h1 className="text-xl font-semibold text-charcoal">Rozy Admin</h1>
          <p className="text-sm text-kdgrey">Verification queue · Mbarara</p>
        </div>
        <button type="button" onClick={logout} className="text-sm text-kdgrey underline">
          Log out
        </button>
      </header>
      <main>{content}</main>
    </div>
  )
}
