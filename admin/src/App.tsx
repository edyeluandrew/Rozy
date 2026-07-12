import { useState } from 'react'
import Login from './Login'
import AdminShell from './AdminShell'
import { getToken } from './api'

export default function App() {
  const [authed, setAuthed] = useState(!!getToken())

  if (!authed) {
    return (
      <div className="min-h-screen bg-cream">
        <Login onLoggedIn={() => setAuthed(true)} />
      </div>
    )
  }

  return <AdminShell onLogout={() => setAuthed(false)} />
}
