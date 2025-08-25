import { useAuth } from '../auth/AuthProvider';

/**
 * Header
 *
 * Global site header containing the brand link and auth-aware navigation.
 *
 * - Always shows the brand ("Wordle") linking back to `#/`.
 * - If authenticated (`me` exists):
 *   - Greets the user with their username.
 *   - Provides links to Profile and a Log out button.
 * - If not authenticated:
 *   - Provides links to Sign in and Sign up pages.
 *
 * Uses `useAuth` to access the current user (`me`) and `logout` action.
 */
export default function Header() {
  const { me, logout } = useAuth();

  return (
    <header className="site-header">
      {/* Brand link */}
      <a href="#" className="brand">
        Wordle
      </a>

      {/* Navigation changes depending on authentication state */}
      <nav className="nav">
        {me ? (
          <>
            {/* Authenticated user view */}
            <span className="text-sm muted nowrap">
              Hello, <strong>@{me.username}</strong>{' '}
            </span>
            <a className="text-sm link-underline" href="#/profile">
              Profile
            </a>
            <button
              className="text-sm btn-link link-underline"
              onClick={logout}
            >
              Log out
            </button>
          </>
        ) : (
          <>
            {/* Guest view */}
            <a className="text-sm link-underline" href="#/auth">
              Sign in
            </a>
            <a className="text-sm link-underline" href="#/auth?mode=signup">
              Sign up
            </a>
          </>
        )}
      </nav>
    </header>
  );
}
