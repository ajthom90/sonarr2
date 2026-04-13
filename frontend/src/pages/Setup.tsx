import { useState, FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import styles from './Login.module.css'  // Reuse the same styles

export function Setup() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [apiKey, setApiKey] = useState('')

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')

    if (password !== confirmPassword) {
      setError('Passwords do not match')
      return
    }
    if (password.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }

    setLoading(true)
    try {
      const res = await fetch('/api/v3/initialize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({ message: 'Setup failed' }))
        setError(body.message || 'Failed to create account')
        return
      }

      const data = await res.json()
      setApiKey(data.apiKey)
    } catch {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  // After setup, show the API key and a button to proceed to login.
  if (apiKey) {
    return (
      <div className={styles.container}>
        <div className={styles.card}>
          <h1 className={styles.title}>sonarr2</h1>
          <p className={styles.subtitle}>Setup complete!</p>
          <div style={{ marginBottom: 'var(--space-4)' }}>
            <p style={{ color: 'var(--color-text-secondary)', fontSize: '0.875rem', marginBottom: 'var(--space-2)' }}>
              Your API key for external integrations:
            </p>
            <code style={{
              display: 'block',
              background: 'var(--color-bg)',
              border: '1px solid var(--color-border)',
              borderRadius: 'var(--radius-sm)',
              padding: 'var(--space-2) var(--space-3)',
              color: 'var(--color-accent)',
              fontSize: '0.75rem',
              wordBreak: 'break-all',
            }}>
              {apiKey}
            </code>
          </div>
          <button
            className={styles.button}
            onClick={() => navigate('/login')}
            style={{ width: '100%' }}
          >
            Continue to Login
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.card}>
        <h1 className={styles.title}>sonarr2</h1>
        <p className={styles.subtitle}>Welcome! Create your account to get started.</p>
        <form onSubmit={handleSubmit} className={styles.form}>
          {error && <div className={styles.error}>{error}</div>}
          <label className={styles.label}>
            Username
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className={styles.input}
              autoFocus
              required
            />
          </label>
          <label className={styles.label}>
            Password
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={styles.input}
              required
              minLength={8}
            />
          </label>
          <label className={styles.label}>
            Confirm Password
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className={styles.input}
              required
            />
          </label>
          <button type="submit" className={styles.button} disabled={loading}>
            {loading ? 'Creating account...' : 'Create Account'}
          </button>
        </form>
      </div>
    </div>
  )
}
