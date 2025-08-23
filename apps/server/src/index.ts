// apps/server/src/index.ts
import 'dotenv/config';
import express from 'express';
import cors from 'cors';
import pino from 'pino';
import { nanoid } from 'nanoid';
import fs from 'node:fs';

import {
  scoreGuess,
  nextCheatingCandidates,
  ANSWERS as DEFAULT_ANSWERS,
} from '@wordle/game-core';
import {
  newGameReq,
  newGameRes,
  guessReq,
  guessRes,
} from '@wordle/protocol';

const log = pino({ level: process.env.LOG_LEVEL ?? 'info' });
const app = express();
app.use(cors());
app.use(express.json());

/**
 * ---- Configurable dictionaries --------------------------------------------
 * ANSWERS: list of valid answers (required). Defaults to game-core list,
 *          or load from WORDS_FILE (JSON array of 5-letter strings).
 * GUESSES: optional set of allowed guesses. If provided via GUESSES_FILE,
 *          server enforces membership. If null, accepts any 5-letter A–Z.
 */
const toLower5 = (w: string) => w.trim().toLowerCase();
const normalizeList = (arr: unknown): string[] | null =>
  Array.isArray(arr) && arr.every(w => typeof w === 'string' && /^[A-Za-z]{5}$/.test(w))
    ? (arr as string[]).map(toLower5)
    : null;

// Defaults (normalized)
let ANSWERS: string[] = normalizeList(DEFAULT_ANSWERS) ?? DEFAULT_ANSWERS.map(toLower5);
let GUESSES: Set<string> | null = null;

// One-arg file loader (returns lowercase list or null)
function loadListFromFile(path: string): string[] | null {
  try {
    const raw = fs.readFileSync(path, 'utf8');
    const arr = JSON.parse(raw);
    return normalizeList(arr);
  } catch {
    return null;
  }
}

// 1) Try files if provided
let answersFromFile: string[] | null = null;
let guessesFromFile: string[] | null = null;

const wordsPath = process.env.WORDS_FILE;
if (wordsPath && fs.existsSync(wordsPath)) {
  answersFromFile = loadListFromFile(wordsPath);
}

const guessesPath = process.env.GUESSES_FILE;
if (guessesPath && fs.existsSync(guessesPath)) {
  guessesFromFile = loadListFromFile(guessesPath);
}

// 2) Otherwise try the `wordle-words` package
let answersFromPkg: string[] | null = null;
let guessesFromPkg: string[] | null = null;
try {
  const ww = require('wordle-words') as { allSolutions?: string[]; allGuesses?: string[] };
  if (ww.allSolutions) answersFromPkg = normalizeList(ww.allSolutions);
  if (ww.allGuesses)   guessesFromPkg = normalizeList(ww.allGuesses);
} catch { /* optional dep */ }

// 3) Choose final lists (all lowercase by construction)
if (answersFromFile?.length) {
  ANSWERS = answersFromFile;
} else if (answersFromPkg?.length) {
  ANSWERS = answersFromPkg;
}

if (guessesFromFile?.length) {
  GUESSES = new Set(guessesFromFile);
} else if (guessesFromPkg?.length) {
  GUESSES = new Set(guessesFromPkg);
} else {
  GUESSES = null; // accept any 5-letter if no guesses list available
}

/**
 * ---- In-memory game state --------------------------------------------------
 */
type GameStateBase = {
  id: string;
  maxRounds: number;
  round: number;
  state: 'playing' | 'won' | 'lost';
};

type NormalGame = GameStateBase & {
  mode: 'normal';
  answer: string;
};

type CheatingGame = GameStateBase & {
  mode: 'cheat';
  candidates: string[];
  finalized?: string; // once narrowed to one
};

type GameState = NormalGame | CheatingGame;

const games = new Map<string, GameState>();

/**
 * Deterministic answer picker (supports daily/seeded play).
 */
function pickAnswer(seed?: string): string {
  const list = ANSWERS;
  if (!seed) return list[Math.floor(Math.random() * list.length)];
  // simple FNV-1a-ish hash
  let h = 2166136261;
  for (const ch of seed) {
    h ^= ch.charCodeAt(0);
    h = Math.imul(h, 16777619);
  }
  return list[Math.abs(h) % list.length];
}

// Safe, defensive alternative to nextCheatingCandidates.
// Never throws; filters invalid candidates; falls back to a reasonable result.
function safeNextCheatingCandidates(
  candidates: string[],
  guess: string
): { next: string[]; marks: ReturnType<typeof scoreGuess> } {
  // keep only valid 5-letter lowercase strings
  const valid = Array.isArray(candidates)
    ? candidates.filter((w) => typeof w === 'string' && /^[a-z]{5}$/.test(w))
    : [];

  const buckets = new Map<
    string,
    { marks: ReturnType<typeof scoreGuess>; words: string[] }
  >();

  for (const ans of valid) {
    try {
      const m = scoreGuess(ans, guess);
      const key = m.join(','); // key by pattern
      const bucket = buckets.get(key);
      if (bucket) bucket.words.push(ans);
      else buckets.set(key, { marks: m, words: [ans] });
    } catch {
      // skip any bad candidate instead of crashing
      continue;
    }
  }

  if (buckets.size === 0) {
    // No valid candidates -> return "all miss" mask and empty pool
    const miss = Array.from({ length: guess.length }, () => 'miss') as ReturnType<
      typeof scoreGuess
    >;
    return { next: [], marks: miss };
  }

  // pick the largest bucket (max ambiguity)
  let best: { marks: ReturnType<typeof scoreGuess>; words: string[] } | null = null;
  for (const b of buckets.values()) {
    if (!best || b.words.length > best.words.length) best = b;
  }
  return { next: best!.words, marks: best!.marks };
}


/**
 * ---- Routes ----------------------------------------------------------------
 */
app.post('/api/new', (req, res) => {
  const parsed = newGameReq.safeParse(req.body);
  if (!parsed.success) return res.status(400).json(parsed.error.format());

  const { mode, maxRounds, seed } = parsed.data;
  const id = nanoid();

  let game: GameState;
  if (mode === 'normal') {
    game = { id, mode, maxRounds, round: 0, state: 'playing', answer: pickAnswer(seed) };
  } else {
    game = { id, mode, maxRounds, round: 0, state: 'playing', candidates: [...ANSWERS] };
  }

  games.set(id, game);
  res.json(newGameRes.parse({ gameId: id, mode, maxRounds }));
});

app.post('/api/guess', (req, res) => {
  const parsed = guessReq.safeParse(req.body);
  if (!parsed.success) return res.status(400).json(parsed.error.format());

  const { gameId, guess } = parsed.data;
  const game = games.get(gameId);
  if (!game) return res.status(404).json({ error: 'Game not found' });
  if (game.state !== 'playing') return res.status(409).json({ error: 'Game finished' });

    const lo = guess.toLowerCase();
    if (!/^[a-z]{5}$/.test(lo)) return res.status(400).json({ error: 'Invalid format' });

    const enforceDict = !(process.env.CHEAT_ALLOW_ANY === '1' && game.mode === 'cheat');
    if (enforceDict && GUESSES && !GUESSES.has(lo)) {
    return res.status(400).json({ error: 'Not in word list' });
    }


  game.round += 1;

  let marks: ReturnType<typeof scoreGuess>;

  if (game.mode === 'normal') {
    try {
      marks = scoreGuess((game as NormalGame).answer, lo);
    } catch (err) {
      log.error({ err, gameId, lo }, 'Normal mode scoring failed');
      return res.status(500).json({ error: 'Scoring failed' });
    }
    if (marks.every((m) => m === 'hit')) game.state = 'won';
  } else {
    // ---- Cheating Host (defensive) -----------------------------------------
    const cg = game as CheatingGame;

    try {
        if (!cg.finalized) {
        if (!Array.isArray(cg.candidates)) {
            log.error({ gameId, lo, cg }, 'Cheat mode: candidates not an array');
            return res.status(500).json({ error: 'Cheat mode internal state invalid' });
        }

        let next: string[]; let m: ReturnType<typeof scoreGuess>;

        try {
            // Try the primary algorithm from game-core first
            const resNc = nextCheatingCandidates(cg.candidates, lo);
            next = resNc.next; m = resNc.marks;
        } catch (err) {
            // Fallback: never throw
            log.warn({ err, gameId, lo, count: cg.candidates.length }, 'Cheat mode: primary algo failed; using safe fallback');
            const resSafe = safeNextCheatingCandidates(cg.candidates, lo);
            next = resSafe.next; m = resSafe.marks;
        }

        marks = m;
        cg.candidates = next;

        log.debug({ gameId, guess: lo, nextCount: next.length }, 'Cheat mode narrowed candidates');

        if (cg.candidates.length === 0) {
            log.warn({ gameId, lastGuess: lo }, 'Cheat mode: no candidates remain');
            // keep playing; round-limit will determine loss
        } else if (cg.candidates.length === 1) {
            cg.finalized = cg.candidates[0];
            log.info({ gameId, finalized: cg.finalized }, 'Cheat mode finalized answer');
        }
        } else {
        // Once finalized, score like normal
        marks = scoreGuess(cg.finalized, lo);
        if (marks.every((m) => m === 'hit')) game.state = 'won';
        }
    } catch (err) {
        log.error({ err, gameId, lo, game }, 'Cheat mode scoring failed');
        return res.status(500).json({ error: 'Cheating mode failed' });
    }
    }

  if (game.state !== 'won' && game.round >= game.maxRounds) game.state = 'lost';

  // If we somehow got here without marks (shouldn’t happen), guard it.
  if (!marks) {
    log.error({ gameId, game }, 'Marks missing after guess processing');
    return res.status(500).json({ error: 'Internal error (marks missing)' });
  }

  res.json(guessRes.parse({ marks, round: game.round, state: game.state }));
});

/**
 * ---- Boot ------------------------------------------------------------------
 */
const port = Number(process.env.PORT ?? 3001);
app.listen(port, () => log.info({ port }, 'server up'));
