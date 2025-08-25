/**
 * Top-level application and main game screen for the Wordle clone.
 * - Hash-based routing (#/, #/auth, #/profile, #/daily)
 * - Modes: normal, cheat, daily (daily is its own page)
 * - Keyboard input (physical + on-screen)
 * - Basic UX: toast errors, win/lose banner, color-blind palette
 */

import { useEffect, useMemo, useRef, useState } from 'react';
import type { Mark } from '@wordle/game-core';
import './styles.css';
import { useAuth } from './auth/AuthProvider';
import { useHashRoute } from './lib/useHashRoute';
import SaveProgressBanner from './components/SaveProgressBanner';
import Header from './components/Header';
import AuthPage from './pages/AuthPage';
import ProfilePage from './pages/ProfilePage';
import DailyPage from './pages/DailyPage';

/** Game modes supported by the client. */
type Mode = 'normal' | 'cheat' | 'daily';

/**
 * Base URL for the API server. Must be set via Vite env.
 * Example: VITE_API_URL="http://localhost:5175"
 */
const API = import.meta.env.VITE_API_URL as string;

/** Standard Wordle dimensions. */
const ROWS = 6, COLS = 5;

/** Coarse-grained game lifecycle states for the non-daily game. */
type State = 'idle' | 'playing' | 'won' | 'lost' | 'error';

/** Browser tab titles per mode. */
const MODE_TITLES: Record<Mode, string> = {
  normal: 'Classic Wordle',
  cheat: 'Cheating Host',
  daily: 'Daily Challenge',
};

/**
 * Router entry point. Delegates to pages based on the URL hash.
 * Falls back to <GameScreen/> on the root route.
 */
export default function App() {
  const hash = useHashRoute();
  const page = useMemo(() => (hash || '#/').split('?')[0], [hash]);

  // Route table
  if (page === '#/auth') return <AuthPage key={hash} />;
  if (page === '#/profile') return <ProfilePage key={hash} />;
  if (page === '#/daily') return <DailyPage key={hash} />;

  return <GameScreen key="game" />;
}

/**
 * The main interactive game screen for "normal" and "cheat" modes.
 * Daily mode lives on a separate page because its state is bound to date.
 */
function GameScreen() {
  const { me } = useAuth();

  // --- Persistent UI preferences (read/write localStorage) -------------------
  const [mode, setMode] = useState<Mode>(
    () => (localStorage.getItem('mode') as Mode) || 'normal',
  );
  const [cb, setCb] = useState(() => localStorage.getItem('cb') === '1'); // color-blind palette

  // --- Game session state ----------------------------------------------------
  const [gameId, setGameId] = useState<string | null>(null);
  const [state, setState] = useState<State>('idle');
  const [err, setErr] = useState<string | null>(null);

  // --- Board state -----------------------------------------------------------
  const [rows, setRows] = useState<string[]>([]);       // submitted guesses
  const [marks, setMarks] = useState<Mark[][]>([]);     // server-returned scores per row
  const [guess, setGuess] = useState('');               // in-progress guess in current row

  // guard to prevent double-submits while awaiting server
  const submittingRef = useRef(false);

  /**
   * Derived key coloring state for the on-screen keyboard.
   * For each letter, retain the "best" status encountered so far:
   * hit > present > miss.
   */
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

  /**
   * Start a new server-side game for the given mode.
   * Notes:
   * - 'daily' is handled by its own page; early return here.
   * - Resets local board state.
   */
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
      setRows([]);
      setMarks([]);
      setGuess('');
      setState('playing');
    } catch (e: any) {
      setErr(e?.message ?? String(e));
      setState('error');
    }
  }

  // On first mount: create a new game for non-daily modes.
  useEffect(() => {
    if (!API) return; // allow build-time validation to catch missing env
    if (mode !== 'daily') {
      newGame(mode);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Keep the document title in sync with the current mode.
  useEffect(() => {
    document.title = MODE_TITLES[mode];
  }, [mode]);

  /**
   * Global physical keyboard handling:
   * - Enter → submit
   * - Backspace → delete last char
   * - A–Z → append if room remains
   *
   * Prevent default for Enter/Backspace to avoid undesired page interactions.
   * Ignore auto-repeats to prevent accidental double submits.
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

  /**
   * Submit the current guess to the server.
   * - Validates local preconditions (gameId exists, length == COLS, not already submitting).
   * - Maps common server error text to short, user-friendly messages.
   * - Applies a temporary "shake" CSS animation to the current row on error.
   */
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

      // Handle common error messages from server with friendlier text.
      if (!r.ok) {
        const text = (await r.text()).trim();
        const msg = /not in word list/i.test(text)
          ? 'Not in word list'
          : /invalid/i.test(text)
            ? 'Enter a valid 5‑letter word'
            : text || `Error ${r.status}`;
        setErr(msg);

        // UX: shake the current row briefly
        const rowEl =
          document.querySelectorAll<HTMLElement>('.row')[rows.length] ?? null;
        rowEl?.classList.add('shake');
        setTimeout(() => rowEl?.classList.remove('shake'), 400);
        setTimeout(() => setErr(null), 1500);
        return;
      }

      // Success: append the guess + marks, clear input, advance state.
      const data = (await r.json()) as {
        marks: Mark[];
        state: Exclude<State, 'idle' | 'error'>;
      };
      setRows((rs) => [...rs, guess.toUpperCase()]);
      setMarks((ms) => [...ms, data.marks]);
      setGuess('');
      setState(data.state);
    } catch (e: any) {
      setErr(e?.message ?? String(e));
    } finally {
      submittingRef.current = false;
    }
  }

  /**
   * On-screen keyboard click handler. Mirrors the physical keyboard behavior.
   */
  function onKeyClick(k: string) {
    if (state !== 'playing') return;
    if (k === 'ENTER') return void submit();
    if (k === 'DEL') return setGuess((g) => g.slice(0, -1));
    if (guess.length < COLS && /^[A-Z]$/.test(k)) setGuess((g) => g + k);
  }

  // Win/Lose banner state + copy
  const showBanner = state === 'won' || state === 'lost';
  const bannerText =
    state === 'won' ? 'You won!' : state === 'lost' ? 'You lost' : '';

  return (
    <>
      {/* Global site header with auth routes */}
      <Header />

      <div className={`app ${cb ? 'cb' : ''}`}>
        <div className="shell">
          {/* Header: title + controls only */}
          <header className="header">
            <h1 className="title">{MODE_TITLES[mode]}</h1>

            <div className="controls" role="group" aria-label="Game controls">
              {/* Mode selector: note that 'daily' navigates to a different page */}
              <label className="control">
                <span className="label">Mode</span>
                <select
                  value={mode}
                  onChange={(e) => {
                    const m = e.target.value as Mode;
                    localStorage.setItem('mode', m);

                    if (m === 'daily') {
                      setMode(m);
                      // Navigate to Daily page; don't start a normal game
                      if (location.hash !== '#/daily') location.hash = '#/daily';
                      return;
                    }

                    // Switching away from daily → ensure we’re back on main route
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

              {/* New Game is disabled for daily since daily is date-bound */}
              <button
                className="btn"
                onClick={() => newGame(mode)}
                disabled={mode === 'daily'}
              >
                New Game
              </button>

              {/* Accessibility: toggle an alternative color palette */}
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

          {/* Guest CTA (top): nudge sign-in while playing */}
          {!me && state === 'playing' && <SaveProgressBanner />}

          {/* Toast for transient errors (word not allowed, invalid, etc.) */}
          <div
            className={`toast ${err ? 'show' : ''}`}
            role="status"
            aria-live="polite"
          >
            {err ?? ''}
          </div>

          {/* Board: renders ROWS x COLS tiles, mixing past rows + live input */}
          <main className="main">
            <div
              className="board"
              style={{ gridTemplateRows: `repeat(${ROWS}, 1fr)` }}
            >
              {Array.from({ length: ROWS }).map((_, r) => {
                // For the current row, show the in-progress guess; for others, show committed rows.
                const g =
                  rows[r] ?? (r === rows.length ? guess.toUpperCase() : '');
                const m = marks[r];
                return (
                  <div className="row" key={r}>
                    {Array.from({ length: COLS }).map((__, c) => {
                      const letter = g[c] ?? '';
                      const status: Mark | '' = m?.[c] ?? '';
                      const cls = status
                        ? `tile ${status}`       // colored after submit
                        : letter
                          ? 'tile filled'        // typed but not submitted
                          : 'tile';              // empty
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

          {/* Status banner (win/lose) with optional guest CTA below it */}
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

          {/* On-screen keyboard mirrors physical keyboard behavior */}
          <Keyboard keyState={keyState} onKey={onKeyClick} />
        </div>
      </div>
    </>
  );
}

/**
 * On-screen keyboard component.
 * - Shows three rows of keys (QWERTY layout).
 * - Highlights keys using the best-known Mark per letter.
 * - ENTER and DEL are rendered as wide keys.
 */
function Keyboard({
  keyState,
  onKey,
}: {
  /** Per-letter best status derived from played rows. */
  keyState: Record<string, Mark>;
  /** Click handler for keys (ENTER/DEL/letters). */
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
        // Prevent bubbling Enter to the window handler twice while focusing buttons.
        if (e.key === 'Enter') e.stopPropagation();
      }}
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
