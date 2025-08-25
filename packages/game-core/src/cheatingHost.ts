// packages/game-core/src/cheatingHost.ts
//
// Implements "Cheating Host" logic — a variant of Wordle where the host
// attempts to avoid giving away the true answer as long as possible.
//
// Instead of committing to one answer at the start, the host maintains a set
// of candidate words. When the player submits a guess, the host buckets
// candidates by what score they *would* return. It then chooses the bucket
// that is most advantageous for the host (fewest hits, then fewest presents).
//
// This strategy maximizes ambiguity and prolongs the game,
// effectively "cheating" the player.
//
// Exports:
//   • nextCheatingCandidates — picks the next candidate bucket given a guess.

import { scoreGuess, type Mark } from './scoring.js';

/**
 * score returns a summary of a marks array:
 *   - hits:    number of "hit" letters (correct position)
 *   - presents:number of "present" letters (wrong position, correct letter)
 */
function score(marks: Mark[]): [hits: number, presents: number] {
  let h = 0, p = 0;
  for (const m of marks) {
    if (m === 'hit') h++;
    else if (m === 'present') p++;
  }
  return [h, p];
}

/**
 * nextCheatingCandidates selects the most "host-friendly" partition of candidates.
 *
 * @param candidates - current possible answer words
 * @param guess      - player's guess
 * @returns          - object with:
 *                      • next:  narrowed candidate list (chosen bucket)
 *                      • marks: marks pattern returned for the guess
 *
 * Algorithm:
 *   1. For each candidate answer:
 *        - Score the guess as if that word were the answer.
 *        - Group words into buckets by the resulting marks pattern.
 *   2. Each bucket stores:
 *        - the marks pattern
 *        - all candidate words consistent with that pattern
 *        - a summary score [hits, presents]
 *   3. Choose the "worst" bucket for the player:
 *        - fewest hits (minimizes certainty),
 *        - then fewest presents (minimizes information).
 *   4. Return the words in that bucket as the new candidate list,
 *      along with the chosen marks pattern.
 *
 * Example:
 *   candidates = ["crane","slate","plant"], guess = "crane"
 *   → might return { next:["slate"], marks:["miss","present","miss","miss","miss"] }
 */
export function nextCheatingCandidates(candidates: string[], guess: string) {
  type Bucket = { marks: Mark[]; words: string[]; s: [number, number] };
  const buckets = new Map<string, Bucket>();

  // Partition candidates by marks pattern
  for (const ans of candidates) {
    const marks = scoreGuess(ans, guess);
    const key = marks
      .map((m) => (m === 'hit' ? 'O' : m === 'present' ? '?' : '_'))
      .join('');
    const b = buckets.get(key) ?? { marks, words: [], s: score(marks) };
    b.words.push(ans);
    buckets.set(key, b);
  }

  // Pick bucket with fewest hits, then fewest presents
  const chosen = [...buckets.values()].sort(
    (a, b) => a.s[0] - b.s[0] || a.s[1] - b.s[1],
  )[0];

  return { next: chosen.words, marks: chosen.marks };
}
