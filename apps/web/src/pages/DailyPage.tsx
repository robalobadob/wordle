/**
 * DailyPage.tsx
 *
 * Implements the daily challenge mode.
 * - Player can play only once per day.
 * - Tracks gameId via server, validates guesses remotely.
 * - Displays win/lose banners, error toasts, and leaderboard of top players.
 * - Keyboard handling is identical to the standard game (both physical + on-screen).
 */

import { useEffect, useMemo, useRef, useState } from 'react';
import Header from '../components/Header';

/** Game lifecycle states for daily challenge. */
type PlayState = 'idle' | 'playing' | 'won' | 'lost' | 'locked';

/** Status for a single letter cell. */
type MarkLabel = 'miss' | 'present' | 'hit';

/** Shape of guess responses returned from server. */
type GuessResponse = {
  marks: number[]; // 0=miss,1=present,2=hit
  state: 'in_progress' | 'won' | 'locked';
  guesses: number;
};

/** Leaderboard row returned by server. */
type LeaderboardRow = { userId: string; guesses: number; elapsedMs: number };

/** API base. Falls back to localhost if no VITE_API_URL defined. */
const API = import.meta.env.VITE_API_URL ?? 'http://localhost:5175';
const ROWS = 6, COLS = 5;

/** Daily API wrapper (no /api prefix). */
const daily = (p: string) => `${API}/daily${p}`;

/**
 * Main page component for daily challenge.
 * Handles:
 * - Starting the daily game
 * - User input (keyboard + on-screen)
 * - Submitting guesses to server
 * - Rendering the board, banners, and leaderboard
 */
export default function DailyPage() {
  // --- Game session state ----------------------------------------------------
  const [gameId, setGameId] = useState<string | null>(null);
  const [state, setState] = useState<PlayState>('idle');
  const [err, setErr] = useState<string | null>(null);

  // --- Board state -----------------------------------------------------------
  const [rows, setRows] = useState<string[]>([]);
  const [marks, setMarks] = useState<MarkLabel[][]>([]);
  const [guess, setGuess] = useState<string>('');

  // Prevent duplicate submissions; store start time for elapsed calc
  const submittingRef = useRef(false);
  const startedRef = useRef<number | null>(null);

  /**
   * On mount: start today's game.
   * - If server returns `played: true`, lock the board (already played today).
   * - Otherwise, initialize with new gameId.
   */
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

  /**
   * Global keyboard listener (physical keyboard).
   * Matches main game behavior: Enter, Backspace, A–Z.
   */
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

  /** Map numeric marks [0,1,2] from server into semantic labels. */
  function mapMarks(nums: number[]): MarkLabel[] {
    return nums.map((n) => (n === 2 ? 'hit' : n === 1 ? 'present' : 'miss'));
  }

  /**
   * Derived keyboard coloring (on-screen).
   * For each letter, retain "best" status so far (hit > present > miss).
   */
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

  /**
   * Submit the current guess to the server.
   * - Validates input (must be 5 letters, not already submitting).
   * - On error, shows toast + shake animation on current row.
   * - On success, appends guess + marks, updates state (won/lost/in_progress).
   */
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

      // Handle errors gracefully
      if (!res.ok) {
        const text = (await res.text()).trim();
        const msg = /word not allowed/i.test(text)
          ? 'Not in word list'
          : /invalid/i.test(text)
            ? 'Enter a valid 5-letter word'
            : text || `Error ${res.status}`;
        setErr(msg);

        // Animate "shake" for current row
        const rowEl =
          document.querySelectorAll<HTMLElement>('.row')[rows.length] ?? null;
        rowEl?.classList.add('shake');
        setTimeout(() => rowEl?.classList.remove('shake'), 400);
        setTimeout(() => setErr(null), 1500);
        return;
      }

      // Success
      const j: GuessResponse = await res.json();
      setRows((rs) => [...rs, guess.toUpperCase()]);
      setMarks((ms) => [...ms, mapMarks(j.marks)]);
      setGuess('');

      if (j.state === 'won') {
        setState('won');
      } else {
        const nextCount = rows.length + 1;
        if (nextCount >= ROWS) setState('lost'); // all rows used
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Guess failed');
    } finally {
      submittingRef.current = false;
    }
  }

  // Derived banner state/text
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

          {/* Info when user already played today */}
          {state === 'locked' && (
            <div className="toast show" role="status" aria-live="polite">
              You already played today. Check the leaderboard below.
            </div>
          )}

          {/* Error toast */}
          <div
            className={`toast ${err ? 'show' : ''}`}
            role="status"
            aria-live="polite"
          >
            {err ?? ''}
          </div>

          {/* Board (6x5 grid) */}
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

          {/* Win/Lose banner */}
          {showBanner && (
            <div className={`banner ${state}`}>
              <div className="banner-content">
                <strong>{bannerText}</strong>
              </div>
            </div>
          )}

          {/* On-screen keyboard (disabled if game locked or finished) */}
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

          {/* Daily leaderboard */}
          <Leaderboard />
        </div>
      </div>
    </>
  );
}

/* ------------------- Keyboard ------------------- */
/**
 * On-screen keyboard for daily mode.
 * - Uses same layout/classes as main game.
 * - Highlights keys according to best known Mark.
 */
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

/* ------------------- Leaderboard ------------------- */
/**
 * Leaderboard shows top 20 players for today’s daily game.
 * - Fetches from `/daily/leaderboard`.
 * - Displays username, number of guesses, and elapsed time.
 */
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
