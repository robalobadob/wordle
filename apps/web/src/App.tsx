import { useEffect, useMemo, useState } from 'react';
import type { Mark } from '@wordle/game-core';
import './styles.css';

type Mode = 'normal' | 'cheat';
const API = import.meta.env.VITE_API_URL as string;
const ROWS = 6, COLS = 5;

type State = 'idle' | 'playing' | 'won' | 'lost' | 'error';

export default function App() {
  const [mode, setMode] = useState<Mode>(() => (localStorage.getItem('mode') as Mode) || 'normal');
  const [cb, setCb] = useState(() => localStorage.getItem('cb') === '1'); // color-blind
  const [gameId, setGameId] = useState<string | null>(null);
  const [state, setState] = useState<State>('idle');
  const [err, setErr] = useState<string | null>(null);

  // guesses + marks from server
  const [rows, setRows] = useState<string[]>([]);
  const [marks, setMarks] = useState<Mark[][]>([]);
  // current input
  const [guess, setGuess] = useState('');

  // derived keyboard coloring
  const keyState = useMemo(() => {
    const ord: Record<Mark, number> = { miss: 0, present: 1, hit: 2 };
    const best: Record<string, Mark> = {};
    for (const ms of marks) {
      ms.forEach((m, i) => {
        const letter = rows[marks.indexOf(ms)]?.[i];
        if (!letter) return;
        const cur = best[letter];
        if (!cur || ord[m] > ord[cur]) best[letter] = m;
      });
    }
    return best; // e.g., { A:'present', E:'hit' }
  }, [rows, marks]);

  // create or recreate a game
  async function newGame(m: Mode) {
    try {
      setErr(null);
      const r = await fetch(`${API}/api/new`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode: m, maxRounds: ROWS })
      });
      if (!r.ok) throw new Error(`/api/new ${r.status}`);
      const data = await r.json();
      setGameId(data.gameId);
      setRows([]);
      setMarks([]);
      setGuess('');
      setState('playing');
    } catch (e: any) {
      setErr(e.message ?? String(e));
      setState('error');
    }
  }

  useEffect(() => { if (API) newGame(mode); }, []); // initial

  // physical keyboard
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (state !== 'playing') return;
      if (e.key === 'Enter') { submit(); return; }
      if (e.key === 'Backspace') { setGuess(g => g.slice(0, -1)); return; }
      const k = e.key.toUpperCase();
      if (/^[A-Z]$/.test(k) && guess.length < COLS) setGuess(g => g + k);
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [state, guess.length]);

  async function submit() {
    if (!gameId || guess.length !== COLS) return;
    try {
      const r = await fetch(`${API}/api/guess`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ gameId, guess })
      });
      if (!r.ok) {
        const text = await r.text();
        if (text.includes('Not in word list')) {
          setErr('Not in word list');
          setTimeout(() => setErr(null), 1500);
        }
        return;
      }
      const data = await r.json() as { marks: Mark[]; round: number; state: State };
      setRows(rs => [...rs, guess.toUpperCase()]);
      setMarks(ms => [...ms, data.marks]);
      setGuess('');
      setState(data.state);
    } catch (e: any) {
      setErr(e.message ?? String(e));
    }
  }

  function onKeyClick(k: string) {
    if (state !== 'playing') return;
    if (k === 'ENTER') return void submit();
    if (k === 'DEL') return setGuess(g => g.slice(0, -1));
    if (guess.length < COLS && /^[A-Z]$/.test(k)) setGuess(g => g + k);
  }

  return (
    <div className={`app ${cb ? 'cb' : ''}`}>
      <div className="hbar">
        <h1 style={{ marginRight: 'auto' }}>Wordle</h1>
        <label>Mode:&nbsp;
          <select
            value={mode}
            onChange={e => {
              const m = e.target.value as Mode;
              setMode(m); localStorage.setItem('mode', m);
              newGame(m);
            }}
          >
            <option value="normal">Normal</option>
            <option value="cheat">Cheating Host</option>
          </select>
        </label>
        <button onClick={() => newGame(mode)}>New Game</button>
        <label title="Color-blind palette" style={{ display:'flex', alignItems:'center', gap:6 }}>
          <input type="checkbox" checked={cb}
            onChange={e => { setCb(e.target.checked); localStorage.setItem('cb', e.target.checked ? '1':'0'); }} />
          CB
        </label>
      </div>

      <p>Status: <b>{state}</b></p>
      <div className="toast">{err ?? ''}</div>

      {/* Board */}
      <div className="board" style={{ gridTemplateRows: `repeat(${ROWS}, 1fr)` }}>
        {Array.from({ length: ROWS }).map((_, r) => {
          const g = rows[r] ?? (r === rows.length ? guess.toUpperCase() : '');
          const m = marks[r];
          return (
            <div className="row" key={r}>
              {Array.from({ length: COLS }).map((__, c) => {
                const letter = g[c] ?? '';
                const status: Mark | '' = m?.[c] ?? '';
                const cls = status ? `tile ${status}` : letter ? 'tile filled' : 'tile';
                return <div key={c} className={cls}>{letter}</div>;
              })}
            </div>
          );
        })}
      </div>

      {/* On-screen keyboard */}
      <Keyboard keyState={keyState} onKey={onKeyClick} />
    </div>
  );
}

function Keyboard({ keyState, onKey }: {
  keyState: Record<string, Mark>, onKey: (k: string) => void
}) {
  const rows = [
    'QWERTYUIOP'.split(''),
    'ASDFGHJKL'.split(''),
    ['ENTER', ...'ZXCVBNM'.split(''), 'DEL']
  ];
  return (
    <div className="kb" role="group" aria-label="Keyboard">
      {rows.map((r, i) => (
        <div className="krow" key={i}>
          {r.map(k => {
            const st = keyState[k] ?? '';
            const wide = (k === 'ENTER' || k === 'DEL') ? 'wide' : '';
            return (
              <button
                key={k}
                className={`key ${st} ${wide}`}
                onClick={() => onKey(k)}
                aria-label={k}
              >{k}</button>
            );
          })}
        </div>
      ))}
    </div>
  );
}
