import { useCallback, useEffect, useState } from 'react'
import { adminApi, type OperatorItem } from './api'

const statuses = ['', 'available', 'busy', 'offline', 'wallet_blocked', 'pending_verification', 'suspended']

export default function Operators() {
  const [operators, setOperators] = useState<OperatorItem[]>([])
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await adminApi.operators(status || undefined)
      setOperators(data.operators)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [status])

  useEffect(() => {
    load()
  }, [load])

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">Operators</h2>
          <p className="text-sm text-kdgrey">Drivers registered in Mbarara</p>
        </div>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="rounded-lg border border-kdborder px-3 py-2 text-sm"
        >
          <option value="">All statuses</option>
          {statuses.filter(Boolean).map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      {loading && operators.length === 0 ? (
        <p className="text-kdgrey">Loading…</p>
      ) : operators.length === 0 ? (
        <p className="rounded-xl border border-kdborder bg-kdwhite p-4 text-kdgrey">No operators found.</p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-kdborder bg-kdwhite">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-kdborder bg-beige/40 text-kdgrey">
              <tr>
                <th className="px-4 py-3">Name</th>
                <th className="px-4 py-3">Phone</th>
                <th className="px-4 py-3">Type</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Wallet</th>
                <th className="px-4 py-3">Plate</th>
              </tr>
            </thead>
            <tbody>
              {operators.map((op) => (
                <tr key={op.id} className="border-b border-kdborder last:border-0">
                  <td className="px-4 py-3 font-medium">{op.name || '—'}</td>
                  <td className="px-4 py-3">{op.phone}</td>
                  <td className="px-4 py-3">{op.ride_type}</td>
                  <td className="px-4 py-3">
                    <span className="rounded-full bg-beige px-2 py-0.5 text-xs">{op.status}</span>
                    {op.verified && <span className="ml-2 text-xs text-green-700">verified</span>}
                  </td>
                  <td className="px-4 py-3">
                    UGX {op.wallet_balance}
                    <span className="block text-xs text-kdgrey">min {op.wallet_min_balance}</span>
                  </td>
                  <td className="px-4 py-3">{op.plate || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
