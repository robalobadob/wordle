import { useEffect, useState } from "react";
import { useAuth } from "../auth/AuthProvider";
import { apiGET } from "../lib/api";

type Stats = { id: string; gamesPlayed: number; wins: number; streak: number };
type GameRow = { id: string; status: string; guesses: number; startedAt: string; finishedAt?: string };

export default function ProfilePage() {
  const { me } = useAuth();
  const [stats, setStats] = useState<Stats | null>(null);
  const [games, setGames] = useState<GameRow[]>([]);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const run = async () => {
      try {
        const [s, g] = await Promise.all([
          apiGET<Stats>("/stats/me"),
          apiGET<GameRow[]>("/games/mine"),
        ]);
        if (!cancelled) { setStats(s); setGames(g); }
      } catch (e: any) {
        setErr(e?.message ?? "Failed to load profile");
      }
    };
    if (me) run();
    return () => { cancelled = true; };
  }, [me]);

  if (!me) return <div className="p-4">Please sign in to view your profile.</div>;
  if (err) return <div className="p-4 text-red-600">{err}</div>;

  return (
    <div className="profile">
      <h1 className="profile-title">@{me.username}</h1>

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

      <h2 className="section-title">Recent games</h2>

      <div className="stack">
        {games.map((g) => (
          <div key={g.id} className="card row-between text-sm">
            <div>
              #{g.id.slice(0, 8)} • {g.status} • {g.guesses} guesses
            </div>
            <div className="muted">
              {new Date(g.startedAt).toLocaleString()}
            </div>
          </div>
        ))}
        {games.length === 0 && (
          <div className="muted text-sm">No games yet.</div>
        )}
      </div>
    </div>
  );
}
