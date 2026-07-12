import { useState } from 'react'
import { clearToken } from './api'
import ActiveTrips from './ActiveTrips'
import Operators from './Operators'
import VerificationQueue from './VerificationQueue'

type Tab = 'verification' | 'trips' | 'operators'

type Props = {
  onLogout: () => void
}

export default function AdminShell({ onLogout }: Props) {
  const [tab, setTab] = useState<Tab>('verification')

  function logout() {
    clearToken()
    onLogout()
  }

  return (
    <div className="min-h-screen bg-cream">
      <header className="border-b border-kdborder bg-kdwhite px-6 py-4">
        <div className="mx-auto flex max-w-6xl items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-charcoal">Rozy Admin</h1>
            <p className="text-sm text-kdgrey">Mbarara operations</p>
          </div>
          <button type="button" onClick={logout} className="text-sm text-kdgrey underline">
            Log out
          </button>
        </div>
        <nav className="mx-auto mt-4 flex max-w-6xl gap-2">
          {([
            ['verification', 'Verification'],
            ['trips', 'Active trips'],
            ['operators', 'Operators'],
          ] as const).map(([id, label]) => (
            <button
              key={id}
              type="button"
              onClick={() => setTab(id)}
              className={`rounded-lg px-4 py-2 text-sm font-medium ${
                tab === id ? 'bg-gold text-charcoal' : 'border border-kdborder bg-kdwhite text-kdgrey'
              }`}
            >
              {label}
            </button>
          ))}
        </nav>
      </header>

      <main className="mx-auto max-w-6xl p-6">
        {tab === 'verification' && <VerificationQueue embedded />}
        {tab === 'trips' && <ActiveTrips />}
        {tab === 'operators' && <Operators />}
      </main>
    </div>
  )
}
