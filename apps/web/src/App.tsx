import { useEffect, useState } from 'react';
import type { Mark } from '@wordle/game-core';

const API = import.meta.env.VITE_API_URL as string;

export default function App() {
  const [gameId, setGameId] = useState<string | null>(null);
  const [guess, setGuess] = useState('');
  const [rows, setRows] = useState<{ guess: string; marks: Mark[] }[]>([]);
  const [state, setState] = useState<'idle'|'playing'|'won'|'lost'>('idle');

  useEffect(() => {
    (async () => {
      const r = await fetch(`${API}/api/new`, {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({ mode: 'normal', maxRounds: 6 })
      });
      const data = await r.json();
      setGameId(data.gameId);
      setState('playing');
    })();
  }, []);

  async function submitGuess() {
    if (!gameId || guess.length !== 5) return;
    const r = await fetch(`${API}/api/guess`, {
      method:'POST', headers:{'Content-Type':'application/json'},
      body: JSON.stringify({ gameId, guess })
    });
    const data = await r.json();
    setRows(rows => [...rows, { guess: guess.toUpperCase(), marks: data.marks }]);
    setGuess('');
    setState(data.state);
  }

  return (
    <div style={{ maxWidth: 480, margin: '40px auto', fontFamily: 'system-ui' }}>
      <h1>Wordle</h1>
      <p>Status: <b>{state}</b></p>
      <div style={{ display:'grid', gap:8 }}>
        {rows.map((row, i) => (
          <div key={i} style={{ display:'grid', gridTemplateColumns:'repeat(5,1fr)', gap:6 }}>
            {row.guess.split('').map((c, j) => (
              <div key={j} style={{
                border:'1px solid #ccc', textAlign:'center', padding:'10px 0',
                background: row.marks[j]==='hit' ? '#6aaa64' :
                            row.marks[j]==='present' ? '#c9b458' : '#787c7e',
                color: 'white', fontWeight: 700
              }}>{c}</div>
            ))}
          </div>
        ))}
      </div>

      {state === 'playing' && (
        <form onSubmit={e => { e.preventDefault(); submitGuess(); }}>
          <input value={guess} onChange={e => setGuess(e.target.value)} maxLength={5} placeholder="type 5 letters"
            style={{ marginTop: 16, padding: 8, width: '100%', textTransform:'uppercase' }} />
          <button style={{ marginTop: 8, padding: 8, width: '100%' }}>Guess</button>
        </form>
      )}
    </div>
  );
}
