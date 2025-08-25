import { useEffect, useMemo, useRef, useState } from 'react';
import Header from '../components/Header';

type PlayState = 'idle' | 'playing' | 'won' | 'lost' | 'locked';
type MarkLabel = 'miss' | 'present' | 'hit';

type GuessResponse = {
  marks: number[]; // 0=miss,1=present,2=hit
  state: 'in_progress' | 'won' | 'locked';
  guesses: number;
};

type LeaderboardRow = { userId: string; guesses: number; elapsedMs: number };

const API = import.meta.env.VITE_API_URL ?? 'http://localhost:5175';
const ROWS = 6,
  COLS = 5;
const daily = (p: string) => `${API}/daily${p}`; // no /api prefix

export default function DailyPage() {
  const [gameId, setGameId] = useState<string | null>(null);
  const [state, setState] = useState<PlayState>('idle');
  const [err, setErr] = useState<string | null>(null);

  const [rows, setRows] = useState<string[]>([]);
  const [marks, setMarks] = useState<MarkLabel[][]>([]);
  const [guess, setGuess] = useState<string>('');

  const submittingRef = useRef(false);
  const startedRef = useRef<number | null>(null);

  // Start today's game
  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(daily('/new'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
        });
        if (!res.ok) throw new Error(await res.text());
        const j: { gameId: string; date: string; played: boolean } =
          await res.json();
        if (j.played) {
          setState('locked');
          return;
        }
        setGameId(j.gameId);
        setState('playing');
        startedRef.current = performance.now();
      } catch (e) {
        setErr(e instanceof Error ? e.message : 'Failed to start daily');
      }
    })();
  }, []);

  // Keyboard: mirror main game behavior
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (state !== 'playing') return;
      if (e.key === 'Enter' || e.key === 'Backspace') e.preventDefault();
      if (e.repeat) return;
      if (e.key === 'Enter') return void submit();
      if (e.key === 'Backspace') return setGuess((g) => g.slice(0, -1));
      const k = e.key.toUpperCase();
      if (/^[A-Z]$/.test(k) && guess.length < COLS) setGuess((g) => g + k);
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [state, guess.length]);

  function mapMarks(nums: number[]): MarkLabel[] {
    return nums.map((n) => (n === 2 ? 'hit' : n === 1 ? 'present' : 'miss'));
  }

  // Build per-letter best status for keyboard (miss < present < hit)
  const keyState = useMemo(() => {
    const rank: Record<MarkLabel, number> = { miss: 0, present: 1, hit: 2 };
    const best: Record<string, MarkLabel> = {};
    marks.forEach((ms, rowIdx) => {
      ms.forEach((m, i) => {
        const letter = rows[rowIdx]?.[i]?.toUpperCase();
        if (!letter) return;
        const cur = best[letter];
        if (!cur || rank[m] > rank[cur]) best[letter] = m;
      });
    });
    return best;
  }, [rows, marks]);

  async function submit() {
    if (!gameId || guess.length !== COLS || submittingRef.current) return;
    submittingRef.current = true;
    try {
      const res = await fetch(daily('/guess'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ gameId: gameId, word: guess.toLowerCase() }),
      });
      if (!res.ok) {
        const text = (await res.text()).trim();
        const msg = /word not allowed/i.test(text)
          ? 'Not in word list'
          : /invalid/i.test(text)
            ? 'Enter a valid 5‑letter word'
            : text || `Error ${res.status}`;
        setErr(msg);
        const rowEl =
          document.querySelectorAll<HTMLElement>('.row')[rows.length] ?? null;
        rowEl?.classList.add('shake');
        setTimeout(() => rowEl?.classList.remove('shake'), 400);
        setTimeout(() => setErr(null), 1500);
        return;
      }
      const j: GuessResponse = await res.json();
      setRows((rs) => [...rs, guess.toUpperCase()]);
      setMarks((ms) => [...ms, mapMarks(j.marks)]);
      setGuess('');

      if (j.state === 'won') {
        setState('won');
      } else {
        // Detect loss when we've used all rows and server is still in_progress
        const nextCount = rows.length + 1;
        if (nextCount >= ROWS) setState('lost');
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Guess failed');
    } finally {
      submittingRef.current = false;
    }
  }

  const showBanner = state === 'won' || state === 'lost';
  const bannerText =
    state === 'won' ? 'You won!' : state === 'lost' ? 'You lost' : '';

  return (
    <>
      <Header />
      <div className="app">
        <div className="shell">
          {/* Title */}
          <header className="header">
            <h1 className="title">Daily Challenge</h1>
          </header>

          {/* Notice when already played */}
          {state === 'locked' && (
            <div className="toast show" role="status" aria-live="polite">
              You already played today. Check the leaderboard below.
            </div>
          )}

          {/* Toast (errors) */}
          <div
            className={`toast ${err ? 'show' : ''}`}
            role="status"
            aria-live="polite"
          >
            {err ?? ''}
          </div>

          {/* Board (identical structure/classes to main game) */}
          <main className="main">
            <div
              className="board"
              style={{ gridTemplateRows: `repeat(${ROWS}, 1fr)` }}
            >
              {Array.from({ length: ROWS }).map((_, r) => {
                const g =
                  rows[r] ?? (r === rows.length ? guess.toUpperCase() : '');
                const m = marks[r];
                return (
                  <div className="row" key={r}>
                    {Array.from({ length: COLS }).map((__, c) => {
                      const letter = g[c] ?? '';
                      const status: MarkLabel | '' = m?.[c] ?? '';
                      const cls = status
                        ? `tile ${status}`
                        : letter
                          ? 'tile filled'
                          : 'tile';
                      return (
                        <div key={c} className={cls}>
                          {letter}
                        </div>
                      );
                    })}
                  </div>
                );
              })}
            </div>
          </main>

          {/* Win/Lose banner like standard game */}
          {showBanner && (
            <div className={`banner ${state}`}>
              <div className="banner-content">
                <strong>{bannerText}</strong>
              </div>
            </div>
          )}

          {/* On-screen keyboard (same markup/classes) */}
          {state !== 'locked' && !showBanner && (
            <Keyboard
              keyState={keyState}
              onKey={(k) => {
                if (state !== 'playing') return;
                if (k === 'ENTER') return void submit();
                if (k === 'DEL') return setGuess((g) => g.slice(0, -1));
                if (guess.length < COLS && /^[A-Z]$/.test(k))
                  setGuess((g) => g + k);
              }}
            />
          )}

          {/* Leaderboard */}
          <Leaderboard />
        </div>
      </div>
    </>
  );
}

/* ------- Keyboard (same markup/classes as main game) ------- */
function Keyboard({
  keyState,
  onKey,
}: {
  keyState: Record<string, MarkLabel>;
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
      onKeyDownCapture={(e) => {
        if (e.key === 'Enter') e.stopPropagation();
      }}
    >
      {rows.map((r, i) => (
        <div className="krow" key={i}>
          {r.map((k) => {
            const wide = k === 'ENTER' || k === 'DEL' ? 'wide' : '';
            const st = keyState[k] ?? ''; // "", "miss", "present", "hit"
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

/* ------- Leaderboard ------- */
function Leaderboard() {
  const [rows, setRows] = useState<LeaderboardRow[]>([]);
  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(daily('/leaderboard'), {
          credentials: 'include',
        });
        if (!res.ok) throw new Error(await res.text());
        const j: { date: string; top: LeaderboardRow[] } = await res.json();
        setRows(Array.isArray(j.top) ? j.top : []);
      } catch {
        setRows([]);
      }
    })();
  }, []);

  return (
    <div className="card" style={{ marginTop: '1rem' }}>
      <h2 className="section-title">Today’s Top 20</h2>
      <div
        className="row-between muted"
        style={{ paddingBottom: 6, borderBottom: '1px solid #e6e9ed' }}
      >
        <div style={{ flex: 1 }}>Player</div>
        <div style={{ width: 90, textAlign: 'right' }}>Guesses</div>
        <div style={{ width: 110, textAlign: 'right' }}>Time (ms)</div>
      </div>
      {rows.map((r, i) => (
        <div
          key={`${r.userId}-${i}`}
          className="row-between"
          style={{ padding: '6px 0', borderBottom: '1px solid #f0f2f5' }}
        >
          <div
            style={{
              flex: 1,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {r.userId}
          </div>
          <div style={{ width: 90, textAlign: 'right' }}>{r.guesses}</div>
          <div style={{ width: 110, textAlign: 'right' }}>{r.elapsedMs}</div>
        </div>
      ))}
    </div>
  );
}
