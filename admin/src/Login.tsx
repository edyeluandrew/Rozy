import { useState } from 'react'
import { authApi, setToken } from './api'

type Props = { onLoggedIn: () => void }

export default function Login({ onLoggedIn }: Props) {
  const [phone, setPhone] = useState('+256700000000')
  const [code, setCode] = useState('')
  const [sent, setSent] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function sendOtp() {
    setLoading(true)
    setError('')
    try {
      await authApi.requestOtp(phone)
      setSent(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed')
    } finally {
      setLoading(false)
    }
  }

  async function verify() {
    setLoading(true)
    setError('')
    try {
      const res = await authApi.verifyOtp(phone, code)
      if (res.user.role !== 'admin') {
        setError('This account is not an admin')
        return
      }
      setToken(res.token)
      onLoggedIn()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto mt-24 max-w-md rounded-2xl border border-kdborder bg-kdwhite p-8">
      <h1 className="text-xl font-semibold">Rozy Admin</h1>
      <p className="mt-1 text-sm text-kdgrey">Mbarara operations login</p>
      <p className="mt-4 text-xs text-kdgrey">
        Any admin phone in the database works · seeded: +256700000000 · OTP in Render logs
      </p>
      <label className="mt-4 block text-sm">Phone</label>
      <input
        className="mt-1 w-full rounded-lg border border-kdborder px-3 py-2"
        value={phone}
        onChange={(e) => setPhone(e.target.value)}
      />
      {!sent ? (
        <button
          type="button"
          className="mt-4 w-full rounded-xl bg-gold py-2.5 font-semibold text-charcoal disabled:opacity-50"
          onClick={sendOtp}
          disabled={loading}
        >
          Send OTP
        </button>
      ) : (
        <>
          <label className="mt-4 block text-sm">OTP code</label>
          <input
            className="mt-1 w-full rounded-lg border border-kdborder px-3 py-2"
            value={code}
            onChange={(e) => setCode(e.target.value)}
          />
          <button
            type="button"
            className="mt-4 w-full rounded-xl bg-gold py-2.5 font-semibold text-charcoal disabled:opacity-50"
            onClick={verify}
            disabled={loading}
          >
            Verify & enter
          </button>
        </>
      )}
      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
    </div>
  )
}
