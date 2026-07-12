import { useCallback, useEffect, useState } from 'react'
import { adminApi, type ActiveTrip } from './api'

export default function ActiveTrips() {
  const [trips, setTrips] = useState<ActiveTrip[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await adminApi.activeTrips()
      setTrips(data.trips)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
    const id = window.setInterval(load, 8000)
    return () => window.clearInterval(id)
  }, [load])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Active trips</h2>
          <p className="text-sm text-kdgrey">Live monitor · refreshes every 8s</p>
        </div>
        <button
          type="button"
          onClick={load}
          disabled={loading}
          className="rounded-lg border border-kdborder px-3 py-1.5 text-sm disabled:opacity-50"
        >
          Refresh
        </button>
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      {trips.length === 0 ? (
        <p className="rounded-xl border border-kdborder bg-kdwhite p-4 text-kdgrey">No active trips right now.</p>
      ) : (
        <div className="space-y-3">
          {trips.map((trip) => (
            <div key={trip.id} className="rounded-xl border border-kdborder bg-kdwhite p-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="font-semibold">{trip.status.replace(/_/g, ' ')}</p>
                  <p className="text-sm text-kdgrey">
                    {trip.ride_type} · {trip.passenger_phone}
                  </p>
                  {trip.driver_name && (
                    <p className="text-sm">
                      Driver: {trip.driver_name}
                      {trip.driver_plate ? ` · ${trip.driver_plate}` : ''}
                    </p>
                  )}
                </div>
                <div className="text-right text-sm">
                  {trip.estimated_fare != null && (
                    <p className="font-semibold text-gold">UGX {trip.estimated_fare}</p>
                  )}
                  <p className="text-xs text-kdgrey">{trip.id.slice(0, 8)}…</p>
                </div>
              </div>
              <p className="mt-2 text-xs text-kdgrey">
                Pickup {trip.pickup_lat.toFixed(4)}, {trip.pickup_lng.toFixed(4)}
                {trip.driver_lat != null && trip.driver_lng != null
                  ? ` · Driver ${trip.driver_lat.toFixed(4)}, ${trip.driver_lng.toFixed(4)}`
                  : ''}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
