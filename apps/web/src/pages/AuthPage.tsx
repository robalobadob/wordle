import { useMemo, useState } from 'react';
import { useAuth } from '../auth/AuthProvider';
import Header from '../components/Header';

export default function AuthPage() {
  const { signup, login } = useAuth();
  const mode = useMemo(
    () =>
      new URLSearchParams(location.hash.split('?')[1]).get('mode') === 'signup'
        ? 'signup'
        : 'login',
    [],
  );
  const [username, setU] = useState('');
  const [password, setP] = useState('');
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr(null);
    setLoading(true);
    try {
      if (mode === 'signup') await signup(username, password);
      else await login(username, password);
      location.hash = '#/profile'; // after auth, show profile (stats)
    } catch (e: any) {
      setErr(e?.message ?? 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Header />
      <main className="page">
        <div className="auth">
          <h1 className="auth-title">
            {mode === 'signup' ? 'Create account' : 'Sign in'}
          </h1>

          {err && <div className="alert error">{err}</div>}

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
            <button disabled={loading} className="btn btn-primary">
              {loading ? '…' : mode === 'signup' ? 'Sign up' : 'Sign in'}
            </button>
          </form>

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
