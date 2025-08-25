import { useAuth } from '../auth/AuthProvider';

/**
 * SaveProgressBanner
 *
 * A guest-only call-to-action banner encouraging players
 * to sign in or create an account.
 *
 * - Shown only when `me` (the authenticated user) is null/undefined.
 * - Provides links to login (`#/auth`) and signup (`#/auth?mode=signup`).
 * - Highlights benefits: saving streaks, viewing history, and challenging friends.
 *
 * Usage:
 * ```tsx
 * {!me && state === 'playing' && <SaveProgressBanner />}
 * ```
 */
export default function SaveProgressBanner() {
  const { me } = useAuth();

  // Do not render banner for authenticated users
  if (me) return null;

  return (
    <div className="card card-warn mt-3">
      <div className="text-sm">
        <strong>Sign in</strong> to save your streak, view history, and
        challenge friends.
      </div>
      <div className="hstack mt-2">
        <a href="#/auth" className="btn btn-primary text-sm">
          Sign in
        </a>
        <a href="#/auth?mode=signup" className="btn btn-outline text-sm">
          Create account
        </a>
      </div>
    </div>
  );
}
