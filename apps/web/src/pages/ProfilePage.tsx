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
    <div className="p-4">
      <h1 className="text-xl font-bold mb-2">@{me.username}</h1>
      {stats && (
        <div className="border rounded p-3 mb-4 text-sm flex gap-6">
          <div><b>Games</b><div>{stats.gamesPlayed}</div></div>
          <div><b>Wins</b><div>{stats.wins}</div></div>
          <div><b>Streak</b><div>{stats.streak}</div></div>
        </div>
      )}
      <h2 className="font-semibold mb-2">Recent games</h2>
      <div className="space-y-2">
        {games.map(g => (
          <div key={g.id} className="border rounded p-2 text-sm flex justify-between">
            <div>#{g.id.slice(0,8)} • {g.status} • {g.guesses} guesses</div>
            <div className="opacity-60">{new Date(g.startedAt).toLocaleString()}</div>
          </div>
        ))}
        {games.length === 0 && <div className="opacity-60 text-sm">No games yet.</div>}
      </div>
    </div>
  );
}
