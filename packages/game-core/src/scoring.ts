// packages/game-core/src/scoring.ts
//
// Wordle scoring logic, shared across client and server.
// Implements the standard two-pass algorithm to evaluate a guess
// against the correct answer.
//
// Mark legend:
//   - "hit":     correct letter, correct position
//   - "present": correct letter, wrong position
//   - "miss":    letter not in the answer
//
// Rules:
//   • Both words must be exactly 5 characters, lowercase a–z.
//   • Input is normalized to lowercase before comparison.
//   • The algorithm correctly handles repeated letters by counting
//     non-hit answer letters and decrementing counts as matches are used.

export type Mark = 'hit' | 'present' | 'miss';

/**
 * scoreGuess compares a guess against the answer and produces a per-letter evaluation.
 *
 * @param answer - the correct solution word (must be 5 letters, a–z only)
 * @param guess  - the player's guess word (must be 5 letters, a–z only)
 * @returns      - an array of 5 marks: "hit", "present", or "miss"
 *
 * Example:
 *   answer = "crane", guess = "cared"
 *   → ["hit", "present", "present", "present", "miss"]
 */
export function scoreGuess(answer: string, guess: string): Mark[] {
  const A = answer.toLowerCase();
  const G = guess.toLowerCase();

  // Validate inputs
  if (A.length !== 5 || G.length !== 5)
    throw new Error('Words must be 5 letters');
  if (!/^[a-z]{5}$/.test(A) || !/^[a-z]{5}$/.test(G)) {
    throw new Error('Only a–z letters allowed');
  }

  const marks: Mark[] = Array(5).fill('miss');
  const counts: Record<string, number> = {};

  // Pass 1: mark exact hits and count remaining answer letters
  for (let i = 0; i < 5; i++) {
    if (G[i] === A[i]) {
      marks[i] = 'hit';
    } else {
      counts[A[i]] = (counts[A[i]] ?? 0) + 1;
    }
  }

  // Pass 2: mark "present" if guess letters exist in remaining counts
  for (let i = 0; i < 5; i++) {
    if (marks[i] === 'hit') continue;
    const c = G[i];
    if ((counts[c] ?? 0) > 0) {
      marks[i] = 'present';
      counts[c]--;
    }
  }

  return marks;
}
