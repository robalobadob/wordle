/**
 * ProfilePage.tsx
 *
 * Shows the authenticated user's profile:
 * - Summary stats (games played, wins, streak)
 * - Recent games list
 *
 * Data flow:
 *   mount → if authenticated (`me`) → fetch /stats/me and /games/mine in parallel
 *   → store in local state → render cards
 *
 * Notes:
 * - Uses a cancellation flag in the effect to avoid setting state after unmount.
 * - Minimal error surface shown as an inline message.
 */

import { useEffect, useState } from 'react';
import { useAuth } from '../auth/AuthProvider';
import { apiGET } from '../lib/api';
import Header from '../components/Header';

/** Aggregate stats for the current user. */
type Stats = { id: string; gamesPlayed: number; wins: number; streak: number };

/** Row for a single past game. */
type GameRow = {
  id: string;
  status: string;        // e.g., "won" | "lost" | "in_progress"
  guesses: number;       // number of submitted guesses
  startedAt: string;     // ISO timestamp
  finishedAt?: string;   // ISO timestamp (optional when incomplete)
};

/**
 * Profile page component.
 * Requires an authenticated user; otherwise prompts to sign in.
 */
export default function ProfilePage() {
  const { me } = useAuth();

  // Data states
  const [stats, setStats] = useState<Stats | null>(null);
  const [games, setGames] = useState<GameRow[]>([]);
  const [err, setErr] = useState<string | null>(null);

  /**
   * On auth change: fetch profile stats + game history in parallel.
   * Uses a `cancelled` guard to prevent state updates after unmount.
   */
  useEffect(() => {
    let cancelled = false;
    const run = async () => {
      try {
        const [s, g] = await Promise.all([
          apiGET<Stats>('/stats/me'),
          apiGET<GameRow[]>('/games/mine'),
        ]);
        if (!cancelled) {
          setStats(s);
          setGames(g);
        }
      } catch (e: any) {
        if (!cancelled) setErr(e?.message ?? 'Failed to load profile');
      }
    };
    if (me) run();
    return () => {
      cancelled = true;
    };
  }, [me]);

  // --- Render branches -------------------------------------------------------

  // If user is not authenticated, prompt to sign in.
  if (!me)
    return <div className="p-4">Please sign in to view your profile.</div>;

  // Error state
  if (err) return <div className="p-4 text-red-600">{err}</div>;

  return (
    <>
      <Header />
      <main className="page">
        <div className="profile">
          {/* Username header */}
          <h1 className="profile-title">@{me.username}</h1>

          {/* Stats summary card (render only when loaded) */}
          {stats && (
            <div className="card stats">
              <div className="stat">
                <b>Games</b>
                <div>{stats.gamesPlayed}</div>
              </div>
              <div className="stat">
                <b>Wins</b>
                <div>{stats.wins}</div>
              </div>
              <div className="stat">
                <b>Streak</b>
                <div>{stats.streak}</div>
              </div>
            </div>
          )}

          {/* Recent games list */}
          <h2 className="section-title">Recent games</h2>

          <div className="stack">
            {games.map((g) => (
              <div key={g.id} className="card row-between text-sm">
                <div>
                  #{g.id.slice(0, 8)} • {g.status} • {g.guesses} guesses
                </div>
                <div className="muted">
                  {/* Human-friendly timestamp */}
                  {new Date(g.startedAt).toLocaleString()}
                </div>
              </div>
            ))}

            {/* Empty-state message */}
            {games.length === 0 && (
              <div className="muted text-sm">No games yet.</div>
            )}
          </div>
        </div>
      </main>
    </>
  );
}
