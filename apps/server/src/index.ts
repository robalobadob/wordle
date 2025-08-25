// apps/server/src/index.ts
//
// 🚨 Legacy Notice 🚨
// This Express-based TypeScript server was the *original* backend for the project,
// before the transition to the Go-based server (apps/go-server). It is preserved
// here for posterity, reference, and to provide a view of the refactoring process.
//
// Responsibilities (legacy):
//   • Manage Wordle game sessions (normal + cheating host modes).
//   • Maintain in-memory game state (non-persistent).
//   • Load dictionaries from multiple possible sources (files, wordle-words pkg).
//   • Expose simple HTTP APIs for creating games and making guesses.
//
// It is *not* actively used in production anymore. The Go server fully replaced
// it with a more robust, persistent, and feature-rich implementation.
// Keeping this file helps illustrate the evolution from a Node/Express prototype
// to the current Go server.
//
// ---------------------------------------------------------------------------

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
import { newGameReq, newGameRes, guessReq, guessRes } from '@wordle/protocol';

const log = pino({ level: process.env.LOG_LEVEL ?? 'info' });
const app = express();
app.use(cors());
app.use(express.json());

/* -------------------------------------------------------------------------- */
/*                           Dictionary initialization                        */
/* -------------------------------------------------------------------------- */
/**
 * Game word lists can come from several sources:
 *   1. JSON files (WORDS_FILE, GUESSES_FILE)
 *   2. Optional npm package `wordle-words`
 *   3. Fallback defaults in @wordle/game-core
 *
 * This layering made it easier to develop locally without relying on external
 * assets, but was one of the motivations to move word management into the Go
 * server with embedded assets.
 */
const toLower5 = (w: string) => w.trim().toLowerCase();
const normalizeList = (arr: unknown): string[] | null =>
  Array.isArray(arr) &&
  arr.every((w) => typeof w === 'string' && /^[A-Za-z]{5}$/.test(w))
    ? (arr as string[]).map(toLower5)
    : null;

let ANSWERS: string[] =
  normalizeList(DEFAULT_ANSWERS) ?? DEFAULT_ANSWERS.map(toLower5);
let GUESSES: Set<string> | null = null;

function loadListFromFile(path: string): string[] | null {
  try {
    const raw = fs.readFileSync(path, 'utf8');
    const arr = JSON.parse(raw);
    return normalizeList(arr);
  } catch {
    return null;
  }
}

// 1) Try env-provided files
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

// 2) Optional dependency: `wordle-words` npm package
let answersFromPkg: string[] | null = null;
let guessesFromPkg: string[] | null = null;
type WordsPkg = {
  allSolutions?: string[];
  allGuesses?: string[];
  default?: { allSolutions?: string[]; allGuesses?: string[] };
};
try {
  const mod: WordsPkg = await import('wordle-words');
  const ww = mod.default ?? mod;
  if (ww.allSolutions) answersFromPkg = normalizeList(ww.allSolutions);
  if (ww.allGuesses) guessesFromPkg = normalizeList(ww.allGuesses);
} catch {
  /* optional dep */
}

// 3) Choose final lists
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
  GUESSES = null; // accept any 5-letter if no guesses list
}

/* -------------------------------------------------------------------------- */
/*                              In-memory state                               */
/* -------------------------------------------------------------------------- */
type GameStateBase = {
  id: string;
  maxRounds: number;
  round: number;
  state: 'playing' | 'won' | 'lost';
};
type NormalGame = GameStateBase & { mode: 'normal'; answer: string };
type CheatingGame = GameStateBase & {
  mode: 'cheat';
  candidates: string[];
  finalized?: string;
};
type GameState = NormalGame | CheatingGame;

// ⚠️ All game state lives only in memory — restarting loses everything.
// This was another motivator for switching to the Go server + SQLite.
const games = new Map<string, GameState>();

/* -------------------------------------------------------------------------- */
/*                        Answer picker + cheat fallback                      */
/* -------------------------------------------------------------------------- */
function pickAnswer(seed?: string): string {
  const list = ANSWERS;
  if (!seed) return list[Math.floor(Math.random() * list.length)];
  // Deterministic hash for seeded/daily play
  let h = 2166136261;
  for (const ch of seed) {
    h ^= ch.charCodeAt(0);
    h = Math.imul(h, 16777619);
  }
  return list[Math.abs(h) % list.length];
}

// Safe fallback around nextCheatingCandidates
function safeNextCheatingCandidates(
  candidates: string[],
  guess: string,
): { next: string[]; marks: ReturnType<typeof scoreGuess> } {
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
      const key = m.join(',');
      const bucket = buckets.get(key);
      if (bucket) bucket.words.push(ans);
      else buckets.set(key, { marks: m, words: [ans] });
    } catch {
      continue;
    }
  }
  if (buckets.size === 0) {
    const miss = Array.from({ length: guess.length }, () => 'miss') as ReturnType<
      typeof scoreGuess
    >;
    return { next: [], marks: miss };
  }
  let best: { marks: ReturnType<typeof scoreGuess>; words: string[] } | null =
    null;
  for (const b of buckets.values()) {
    if (!best || b.words.length > best.words.length) best = b;
  }
  return { next: best!.words, marks: best!.marks };
}

/* -------------------------------------------------------------------------- */
/*                                  Routes                                    */
/* -------------------------------------------------------------------------- */
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
  if (GUESSES && !GUESSES.has(lo)) return res.status(400).json({ error: 'Not in word list' });

  game.round += 1;
  let marks: ReturnType<typeof scoreGuess>;

  if (game.mode === 'normal') {
    marks = scoreGuess((game as NormalGame).answer, lo);
    if (marks.every((m) => m === 'hit')) game.state = 'won';
  } else {
    // Cheating mode: adversarial narrowing of candidates
    const cg = game as CheatingGame;
    if (!cg.finalized) {
      let next: string[];
      let m: ReturnType<typeof scoreGuess>;
      try {
        const resNc = nextCheatingCandidates(cg.candidates, lo);
        next = resNc.next;
        m = resNc.marks;
      } catch {
        const resSafe = safeNextCheatingCandidates(cg.candidates, lo);
        next = resSafe.next;
        m = resSafe.marks;
      }
      marks = m;
      cg.candidates = next;
      if (cg.candidates.length === 1) cg.finalized = cg.candidates[0];
    } else {
      marks = scoreGuess(cg.finalized, lo);
      if (marks.every((m) => m === 'hit')) game.state = 'won';
    }
  }

  if (game.state !== 'won' && game.round >= game.maxRounds) game.state = 'lost';
  res.json(guessRes.parse({ marks, round: game.round, state: game.state }));
});

/* -------------------------------------------------------------------------- */
/*                                   Boot                                     */
/* -------------------------------------------------------------------------- */
const port = Number(process.env.PORT ?? 3001);
app.listen(port, () => log.info({ port }, 'server up'));
