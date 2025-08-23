import { z } from 'zod';

export const markSchema = z.enum(['hit', 'present', 'miss']);
export type Mark = z.infer<typeof markSchema>;

export const modeSchema = z.enum(['normal', 'cheat']);
export type Mode = z.infer<typeof modeSchema>;

export const newGameReq = z.object({
  mode: modeSchema.default('normal'),
  maxRounds: z.number().int().min(1).max(10).default(6),
  seed: z.string().optional(),
});
export const newGameRes = z.object({
  gameId: z.string(),
  mode: modeSchema,
  maxRounds: z.number().int(),
});

export const guessReq = z.object({
  gameId: z.string(),
  guess: z.string().regex(/^[A-Za-z]{5}$/),
});
export const guessRes = z.object({
  marks: z.array(markSchema).length(5),
  round: z.number().int().min(1),
  state: z.enum(['playing', 'won', 'lost']),
});
