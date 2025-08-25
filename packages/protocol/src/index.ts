// packages/protocol/src/index.ts
//
// Shared protocol definitions for Wordle client and server.
// Uses Zod schemas for runtime validation + TypeScript types for compile-time safety.
//
// Defines:
//   - Mark:  per-letter evaluation ("hit", "present", "miss").
//   - Mode:  game mode ("normal", "cheat").
//   - Request/response shapes for starting a game and submitting guesses.
//
// These schemas are consumed on both ends (server validates inputs, client
// infers types and ensures consistent expectations).

import { z } from 'zod';

/**
 * Mark schema:
 *  - "hit"     → correct letter, correct position
 *  - "present" → correct letter, wrong position
 *  - "miss"    → letter not in the answer
 */
export const markSchema = z.enum(['hit', 'present', 'miss']);
export type Mark = z.infer<typeof markSchema>;

/**
 * Mode schema:
 *  - "normal" → standard game rules
 *  - "cheat"  → alternate testing/debug mode
 */
export const modeSchema = z.enum(['normal', 'cheat']);
export type Mode = z.infer<typeof modeSchema>;

/* -------------------------------------------------------------------------- */
/*                             /game/new endpoint                             */
/* -------------------------------------------------------------------------- */

/**
 * Request to start a new game.
 *  - mode:       optional, defaults to "normal"
 *  - maxRounds:  number of allowed guesses (1–10), defaults to 6
 *  - seed:       optional string for deterministic word selection (if supported)
 */
export const newGameReq = z.object({
  mode: modeSchema.default('normal'),
  maxRounds: z.number().int().min(1).max(10).default(6),
  seed: z.string().optional(),
});

/**
 * Response to /game/new:
 *  - gameId:    unique game identifier
 *  - mode:      mode actually used
 *  - maxRounds: configured max rounds
 */
export const newGameRes = z.object({
  gameId: z.string(),
  mode: modeSchema,
  maxRounds: z.number().int(),
});

/* -------------------------------------------------------------------------- */
/*                            /game/guess endpoint                            */
/* -------------------------------------------------------------------------- */

/**
 * Request to submit a guess.
 *  - gameId: game identifier from newGameRes
 *  - guess:  exactly 5 alphabetic characters (A–Z, case-insensitive)
 */
export const guessReq = z.object({
  gameId: z.string(),
  guess: z.string().regex(/^[A-Za-z]{5}$/),
});

/**
 * Response to /game/guess:
 *  - marks: array of per-letter results (length 5)
 *  - round: round number (1-based, increments per guess)
 *  - state: "playing" | "won" | "lost"
 */
export const guessRes = z.object({
  marks: z.array(markSchema).length(5),
  round: z.number().int().min(1),
  state: z.enum(['playing', 'won', 'lost']),
});
