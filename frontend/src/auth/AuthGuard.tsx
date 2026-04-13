import { useEffect, useState, type ReactNode } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'

type AuthState = 'loading' | 'needs-setup' | 'needs-login' | 'authenticated'

export function AuthGuard({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>('loading')
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    // Skip auth checks on login and setup pages.
    if (location.pathname === '/login' || location.pathname === '/setup') {
      setState('authenticated') // Let these pages render without redirect
      return
    }

    async function check() {
      try {
        // Check if initialized.
        const initRes = await fetch('/api/v3/initialize')
        const initData = await initRes.json()

        if (!initData.initialized) {
          setState('needs-setup')
          return
        }

        // Check if authenticated by trying a simple API call.
        const authRes = await fetch('/api/v6/system/status', {
          credentials: 'include',
        })

        if (authRes.ok) {
          setState('authenticated')
        } else {
          setState('needs-login')
        }
      } catch {
        setState('needs-login')
      }
    }

    check()
  }, [location.pathname])

  useEffect(() => {
    if (state === 'needs-setup' && location.pathname !== '/setup') {
      navigate('/setup', { replace: true })
    } else if (state === 'needs-login' && location.pathname !== '/login') {
      navigate('/login', { replace: true })
    }
  }, [state, location.pathname, navigate])

  if (state === 'loading') {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        background: 'var(--color-bg)',
        color: 'var(--color-text-secondary)',
      }}>
        Loading...
      </div>
    )
  }

  return <>{children}</>
}
