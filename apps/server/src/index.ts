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
let ANSWERS: string[] = DEFAULT_ANSWERS;
let GUESSES: Set<string> | null = null;

function loadListFromFile(path: string, label: string): string[] | null {
  try {
    const raw = fs.readFileSync(path, 'utf8');
    const arr = JSON.parse(raw);
    if (
      Array.isArray(arr) &&
      arr.every((w: unknown) => typeof w === 'string' && /^[A-Za-z]{5}$/.test(w as string))
    ) {
      const list = (arr as string[]).map((w) => w.toUpperCase());
      log.info({ count: list.length, file: path }, `Loaded ${label}`);
      return list;
    }
    log.warn({ file: path }, `${label} file is not a valid array of 5-letter words`);
  } catch (err) {
    log.warn({ err, file: path }, `Failed to load ${label} file`);
  }
  return null;
}

// Load ANSWERS (required list)
if (process.env.WORDS_FILE && fs.existsSync(process.env.WORDS_FILE)) {
  const list = loadListFromFile(process.env.WORDS_FILE, 'answers');
  if (list) ANSWERS = list;
}

// Load optional GUESSES (if present, we enforce; else accept any 5-letter)
if (process.env.GUESSES_FILE && fs.existsSync(process.env.GUESSES_FILE)) {
  const list = loadListFromFile(process.env.GUESSES_FILE, 'guesses');
  if (list) GUESSES = new Set(list);
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

  const up = guess.toUpperCase();

  // Basic format validation
  if (!/^[A-Z]{5}$/.test(up)) {
    return res.status(400).json({ error: 'Invalid format' }); // not 5 letters A–Z
  }
  // Optional dictionary check
  if (GUESSES && !GUESSES.has(up)) {
    return res.status(400).json({ error: 'Not in word list' });
  }

  game.round += 1;

  let marks: ReturnType<typeof scoreGuess>;
  if (game.mode === 'normal') {
    marks = scoreGuess(game.answer, up);
    if (marks.every((m) => m === 'hit')) game.state = 'won';
  } else {
    // cheating mode
    if (!game.finalized) {
      const { next, marks: m } = nextCheatingCandidates(game.candidates, up);
      game.candidates = next;
      marks = m;
      if (game.candidates.length === 1) game.finalized = game.candidates[0];
    } else {
      marks = scoreGuess(game.finalized, up);
      if (marks.every((m) => m === 'hit')) game.state = 'won';
    }
  }

  if (game.state !== 'won' && game.round >= game.maxRounds) game.state = 'lost';

  res.json(guessRes.parse({ marks, round: game.round, state: game.state }));
});

/**
 * ---- Boot ------------------------------------------------------------------
 */
const port = Number(process.env.PORT ?? 3001);
app.listen(port, () => log.info({ port }, 'server up'));
