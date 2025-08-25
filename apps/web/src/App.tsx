// App.tsx
import { useEffect, useMemo, useRef, useState } from 'react';
import type { Mark } from '@wordle/game-core';
import './styles.css';
import { useAuth } from './auth/AuthProvider';
import { useHashRoute } from "./lib/useHashRoute";
import SaveProgressBanner from './components/SaveProgressBanner';
import Header from "./components/Header";
import AuthPage from './pages/AuthPage';
import ProfilePage from './pages/ProfilePage';
import DailyPage from './pages/DailyPage';

type Mode = 'normal' | 'cheat' | 'daily';
const API = import.meta.env.VITE_API_URL as string;
const ROWS = 6, COLS = 5;
type State = 'idle' | 'playing' | 'won' | 'lost' | 'error';

const MODE_TITLES: Record<Mode, string> = {
  normal: 'Classic Wordle',
  cheat: 'Cheating Host',
  daily: 'Daily Challenge',
};

export default function App() {
  const hash = useHashRoute();
  const page = useMemo(() => (hash || '#/').split('?')[0], [hash]);
  if (page === '#/auth') return <AuthPage key={hash} />;
  if (page === '#/profile') return <ProfilePage key={hash} />;
  if (page === '#/daily') return <DailyPage key={hash} />;

  return <GameScreen key="game" />;
}

function GameScreen() {
  const { me } = useAuth();

  const [mode, setMode] = useState<Mode>(() => (localStorage.getItem('mode') as Mode) || 'normal');
  const [cb, setCb] = useState(() => localStorage.getItem('cb') === '1');
  const [gameId, setGameId] = useState<string | null>(null);
  const [state, setState] = useState<State>('idle');
  const [err, setErr] = useState<string | null>(null);

  const [rows, setRows] = useState<string[]>([]);
  const [marks, setMarks] = useState<Mark[][]>([]);
  const [guess, setGuess] = useState('');
  const submittingRef = useRef(false);

  const keyState = useMemo(() => {
    const ord: Record<Mark, number> = { miss: 0, present: 1, hit: 2 };
    const best: Record<string, Mark> = {};
    marks.forEach((ms, rowIdx) => {
      ms.forEach((m, i) => {
        const letter = rows[rowIdx]?.[i];
        if (!letter) return;
        const cur = best[letter];
        if (!cur || ord[m] > ord[cur]) best[letter] = m;
      });
    });
    return best;
  }, [rows, marks]);

  async function newGame(m: Mode) {
    if (m === 'daily') {
      // Daily mode is handled on its own page
      return;
    }
    try {
      setErr(null);
      const r = await fetch(`${API}/game/new`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ mode: m, maxRounds: ROWS }),
      });
      if (!r.ok) throw new Error(`/game/new ${r.status}`);
      const data = (await r.json()) as { gameId: string };
      setGameId(data.gameId);
      setRows([]); setMarks([]); setGuess('');
      setState('playing');
    } catch (e: any) {
      setErr(e?.message ?? String(e));
      setState('error');
    }
  }

  useEffect(() => {
    if (!API) return;
    if (mode !== 'daily') {
      newGame(mode);
    } 
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => { document.title = MODE_TITLES[mode]; }, [mode]);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (state !== 'playing') return;
      if (e.key === 'Enter' || e.key === 'Backspace') e.preventDefault();
      if (e.repeat) return;
      if (e.key === 'Enter') return void submit();
      if (e.key === 'Backspace') return setGuess(g => g.slice(0, -1));
      const k = e.key.toUpperCase();
      if (/^[A-Z]$/.test(k) && guess.length < COLS) setGuess(g => g + k);
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [state, guess.length]);

  async function submit() {
    if (!gameId || guess.length !== COLS || submittingRef.current) return;
    submittingRef.current = true;
    try {
      const r = await fetch(`${API}/game/guess`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ gameId, guess }),
      });
      if (!r.ok) {
        const text = (await r.text()).trim();
        const msg =
          /not in word list/i.test(text) ? 'Not in word list' :
          /invalid/i.test(text) ? 'Enter a valid 5‑letter word' :
          text || `Error ${r.status}`;
        setErr(msg);
        const rowEl = document.querySelectorAll<HTMLElement>('.row')[rows.length] ?? null;
        rowEl?.classList.add('shake'); setTimeout(() => rowEl?.classList.remove('shake'), 400);
        setTimeout(() => setErr(null), 1500);
        return;
      }
      const data = (await r.json()) as { marks: Mark[]; state: Exclude<State, 'idle' | 'error'>; };
      setRows(rs => [...rs, guess.toUpperCase()]);
      setMarks(ms => [...ms, data.marks]);
      setGuess('');
      setState(data.state);
    } catch (e: any) {
      setErr(e?.message ?? String(e));
    } finally {
      submittingRef.current = false;
    }
  }

  function onKeyClick(k: string) {
    if (state !== 'playing') return;
    if (k === 'ENTER') return void submit();
    if (k === 'DEL') return setGuess(g => g.slice(0, -1));
    if (guess.length < COLS && /^[A-Z]$/.test(k)) setGuess(g => g + k);
  }

  const showBanner = state === 'won' || state === 'lost';
  const bannerText = state === 'won' ? 'You won!' : state === 'lost' ? 'You lost' : '';

  return (
    <>
    <Header />
    <div className={`app ${cb ? 'cb' : ''}`}>
      <div className="shell">
        {/* Header: title + controls only */}
        <header className="header">
          <h1 className="title">{MODE_TITLES[mode]}</h1>

          <div className="controls" role="group" aria-label="Game controls">
            <label className="control">
              <span className="label">Mode</span>
              <select
                value={mode}
                onChange={(e) => {
                  const m = e.target.value as Mode;
                  localStorage.setItem('mode', m);
                  if (m === 'daily') {
                    setMode(m);
                    // Navigate to the Daily page; don't start a normal game
                    if (location.hash !== '#/daily') location.hash = '#/daily';
                    return;
                  }
                  // switching away from daily → ensure we're on main game route
                  if (location.hash === '#/daily') location.hash = '#/';
                  setMode(m);
                  newGame(m);
                }}
              >
                <option value="normal">Normal</option>
                <option value="cheat">Cheating Host</option>
                <option value="daily">Daily Challenge</option>
              </select>
            </label>

            <button className="btn" onClick={() => newGame(mode)} disabled={mode === 'daily'}>
              New Game
            </button>

            <label className="control switch" title="Colour-blind palette">
              <span className="nowrap">Colour Blind Palette</span>
              <input
                type="checkbox"
                checked={cb}
                onChange={(e) => {
                  setCb(e.target.checked);
                  localStorage.setItem('cb', e.target.checked ? '1' : '0');
                }}
              />
            </label>
          </div>
        </header>

        {/* Guest CTA (top) */}
        {!me && state === 'playing' && <SaveProgressBanner />}

        {/* Toast */}
        <div className={`toast ${err ? 'show' : ''}`} role="status" aria-live="polite">
          {err ?? ''}
        </div>

        {/* Board */}
        <main className="main">
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
        </main>

        {/* Status banner */}
        {showBanner && (
          <div className={`banner ${state}`}>
            <div className="banner-content">
              <strong>{bannerText}</strong>
              <button className="btn ghost" onClick={() => newGame(mode)}>
                Play again
              </button>
            </div>
            {!me && <SaveProgressBanner />}
          </div>
        )}

        {/* On-screen keyboard */}
        <Keyboard keyState={keyState} onKey={onKeyClick} />
      </div>
    </div>
    </>
  );
}

function Keyboard({
  keyState,
  onKey,
}: {
  keyState: Record<string, Mark>;
  onKey: (k: string) => void;
}) {
  const rows = [
    'QWERTYUIOP'.split(''),
    'ASDFGHJKL'.split(''),
    ['ENTER', ...'ZXCVBNM'.split(''), 'DEL'],
  ];
  return (
    <div
      className="kb"
      role="group"
      aria-label="Keyboard"
      onKeyDownCapture={(e) => { if (e.key === 'Enter') e.stopPropagation(); }}
    >
      {rows.map((r, i) => (
        <div className="krow" key={i}>
          {r.map((k) => {
            const st = keyState[k] ?? '';
            const wide = k === 'ENTER' || k === 'DEL' ? 'wide' : '';
            return (
              <button
                key={k}
                className={`key ${st} ${wide}`}
                onClick={() => onKey(k)}
                aria-label={k}
              >
                {k}
              </button>
            );
          })}
        </div>
      ))}
    </div>
  );
}
