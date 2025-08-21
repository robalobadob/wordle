import 'dotenv/config';
import express from 'express';
import cors from 'cors';
import pino from 'pino';
import { nanoid } from 'nanoid';
import { ANSWERS, ALLOWED, scoreGuess, nextCheatingCandidates } from '@wordle/game-core';
import { newGameReq, newGameRes, guessReq, guessRes, type Mode } from '@wordle/protocol';

const log = pino({ level: process.env.LOG_LEVEL ?? 'info' });
const app = express();
app.use(cors());
app.use(express.json());

type GameState =
  | { id: string; mode: 'normal'; maxRounds: number; round: number; answer: string; state: 'playing'|'won'|'lost' }
  | { id: string; mode: 'cheat';  maxRounds: number; round: number; candidates: string[]; finalized?: string; state: 'playing'|'won'|'lost' };

const games = new Map<string, GameState>();

function pickAnswer(seed?: string): string {
  const list = ANSWERS;
  if (!seed) return list[Math.floor(Math.random() * list.length)];
  // simple deterministic index from seed
  let h = 2166136261;
  for (const c of seed) { h ^= c.charCodeAt(0); h = Math.imul(h, 16777619); }
  const idx = Math.abs(h) % list.length;
  return list[idx];
}

app.post('/api/new', (req, res) => {
  const parsed = newGameReq.safeParse(req.body);
  if (!parsed.success) return res.status(400).json(parsed.error.format());
  const { mode, maxRounds, seed } = parsed.data;

  const id = nanoid();
  let state: GameState;
  if (mode === 'normal') {
    state = { id, mode, maxRounds, round: 0, answer: pickAnswer(seed), state: 'playing' };
  } else {
    state = { id, mode, maxRounds, round: 0, candidates: [...ANSWERS], state: 'playing' };
  }
  games.set(id, state);
  const payload = newGameRes.parse({ gameId: id, mode, maxRounds });
  res.json(payload);
});

app.post('/api/guess', (req, res) => {
  const parsed = guessReq.safeParse(req.body);
  if (!parsed.success) return res.status(400).json(parsed.error.format());
  const { gameId, guess } = parsed.data;

  const g = games.get(gameId);
  if (!g) return res.status(404).json({ error: 'Game not found' });
  if (g.state !== 'playing') return res.status(409).json({ error: 'Game finished' });

  const up = guess.toUpperCase();
  if (!ALLOWED.includes(up)) return res.status(400).json({ error: 'Not in word list' });

  g.round += 1;

  let marks;
  if (g.mode === 'normal') {
    marks = scoreGuess(g.answer, up);
    if (marks.every(m => m === 'hit')) g.state = 'won';
  } else {
    // cheating mode
    if (!g.finalized) {
      const { next, marks: m } = nextCheatingCandidates(g.candidates, up);
      g.candidates = next;
      marks = m;
      if (g.candidates.length === 1) g.finalized = g.candidates[0];
    } else {
      marks = scoreGuess(g.finalized, up);
      if (marks.every(m => m === 'hit')) g.state = 'won';
    }
  }

  if (g.state !== 'won' && g.round >= g.maxRounds) g.state = 'lost';

  const payload = guessRes.parse({ marks, round: g.round, state: g.state });
  res.json(payload);
});

const port = Number(process.env.PORT ?? 3001);
app.listen(port, () => log.info({ port }, 'server up'));
