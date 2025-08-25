/**
 * AuthPage.tsx
 *
 * A simple authentication page that handles both login and signup flows.
 * - Uses hash-based routing (#/auth and #/auth?mode=signup).
 * - Delegates actual auth logic to AuthProvider via `useAuth`.
 * - After successful auth, redirects to "#/profile".
 */

import { useMemo, useState } from 'react';
import { useAuth } from '../auth/AuthProvider';
import Header from '../components/Header';

/**
 * Authentication page component.
 * Supports "login" and "signup" modes depending on the hash query param.
 */
export default function AuthPage() {
  const { signup, login } = useAuth();

  /**
   * Determine which mode to render based on location.hash:
   * - "#/auth" → login
   * - "#/auth?mode=signup" → signup
   */
  const mode = useMemo(
    () =>
      new URLSearchParams(location.hash.split('?')[1]).get('mode') === 'signup'
        ? 'signup'
        : 'login',
    [],
  );

  // --- Form state ------------------------------------------------------------
  const [username, setU] = useState('');
  const [password, setP] = useState('');
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  /**
   * Handle form submission for login/signup.
   * - Calls the relevant auth function (login/signup).
   * - Redirects to profile page on success.
   * - Displays error message on failure.
   */
  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr(null);
    setLoading(true);
    try {
      if (mode === 'signup') await signup(username, password);
      else await login(username, password);

      // After authentication, redirect to profile
      location.hash = '#/profile';
    } catch (e: any) {
      setErr(e?.message ?? 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      {/* Global site header with navigation/auth routes */}
      <Header />

      <main className="page">
        <div className="auth">
          {/* Page title depends on auth mode */}
          <h1 className="auth-title">
            {mode === 'signup' ? 'Create account' : 'Sign in'}
          </h1>

          {/* Error alert (shown if login/signup fails) */}
          {err && <div className="alert error">{err}</div>}

          {/* Auth form: username + password */}
          <form onSubmit={onSubmit} className="form-vert">
            <input
              className="input"
              placeholder="username"
              value={username}
              onChange={(e) => setU(e.target.value)}
              autoComplete="username"
            />
            <input
              className="input"
              placeholder="password"
              type="password"
              value={password}
              onChange={(e) => setP(e.target.value)}
              autoComplete={
                mode === 'signup' ? 'new-password' : 'current-password'
              }
            />

            {/* Submit button switches label based on mode */}
            <button disabled={loading} className="btn btn-primary">
              {loading ? '…' : mode === 'signup' ? 'Sign up' : 'Sign in'}
            </button>
          </form>

          {/* Footer link: toggle between login/signup modes */}
          <div className="auth-foot">
            {mode === 'signup' ? (
              <a href="#/auth" className="link">
                Already have an account? Sign in
              </a>
            ) : (
              <a href="#/auth?mode=signup" className="link">
                Create an account
              </a>
            )}
          </div>
        </div>
      </main>
    </>
  );
}
