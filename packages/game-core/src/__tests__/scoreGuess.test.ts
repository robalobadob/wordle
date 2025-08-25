// packages/game-core/src/__tests__/scoreGuess.test.ts
//
// Unit tests for scoreGuess(), the Wordle scoring algorithm.
// Verifies correctness of per-letter evaluations under different scenarios,
// including exact matches, misses, presents, and tricky duplicate-letter cases.
//
// Covered cases:
//   • All letters correct → all "hit"
//   • All letters absent → all "miss"
//   • Mixed hits/presents → correct handling of partial matches
//   • Duplicate letters → ensures algorithm respects available counts
//                         (no over-crediting repeats)
//
// These tests help ensure consistency with official Wordle rules.

import { scoreGuess } from '../index';

describe('scoreGuess', () => {
  it('marks exact matches as hits', () => {
    expect(scoreGuess('crane', 'crane')).toEqual([
      'hit',
      'hit',
      'hit',
      'hit',
      'hit',
    ]);
  });

  it('marks absent letters as miss', () => {
    expect(scoreGuess('crane', 'bolts')).toEqual([
      'miss',
      'miss',
      'miss',
      'miss',
      'miss',
    ]);
  });

  it('marks present letters in wrong positions as present', () => {
    expect(scoreGuess('crane', 'cacao')).toEqual([
      'hit',     // c in pos 0
      'present', // a exists but wrong position
      'miss',    // c at pos 2 already consumed
      'miss',    // o not in "crane"
      'miss',    // extra letter not present
    ]);
  });

  it('handles duplicate letters in guess when answer has duplicates', () => {
    // Answer "apple" has two p’s. Guess "paper" also has two p’s.
    // The algorithm should credit both but not overcount.
    expect(scoreGuess('apple', 'paper')).toEqual([
      'present', // p (guess 0) exists in answer
      'present', // a exists (wrong position)
      'hit',     // p (guess 2) exact match with answer
      'present', // e exists in answer
      'miss',    // r not in "apple"
    ]);
  });

  it('handles duplicate letters when answer has a single occurrence', () => {
    // Answer "apple" has one 'l'. Guess "alley" has two 'l’s.
    // Only one should count as present; the second is miss.
    expect(scoreGuess('apple', 'alley')).toEqual([
      'hit',     // a exact match
      'present', // first l matches the single 'l'
      'miss',    // second l has no remaining matches
      'present', // e exists in answer
      'miss',    // y not in "apple"
    ]);
  });
});
