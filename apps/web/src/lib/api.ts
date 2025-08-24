const API = import.meta.env.VITE_API_URL as string;

export type State = 'playing' | 'won' | 'lost';
export type Mark = 'hit' | 'present' | 'miss';

export async function newGame(opts?: { mode?: 'normal' | 'cheat'; answer?: string }) {
  const res = await fetch(`${API}/game/new`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(opts ?? {}),
  });
  if (!res.ok) throw new Error(`newGame failed: ${res.status}`);
  return (await res.json()) as { gameId: string };
}

export async function sendGuess(gameId: string, guess: string) {
  const res = await fetch(`${API}/game/guess`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ gameId, guess }),
  });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`guess failed: ${res.status} ${text}`);
  }
  return (await res.json()) as { marks: Mark[]; state: State };
}

export async function health() {
  const res = await fetch(`${API}/health`);
  return res.ok;
}
