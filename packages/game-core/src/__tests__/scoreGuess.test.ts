import { scoreGuess } from '../index';

// These expectations assume scoreGuess returns an array of 5 strings:
// 'hit' | 'present' | 'miss' (typical Wordle semantics).
// If your API differs, tweak the asserts accordingly.

describe('scoreGuess', () => {
  it('marks exact matches as hits', () => {
    expect(scoreGuess('CRANE', 'CRANE')).toEqual(['hit', 'hit', 'hit', 'hit', 'hit']);
  });

  it('marks absent letters as miss', () => {
    expect(scoreGuess('CRANE', 'BOLTS')).toEqual(['miss', 'miss', 'miss', 'miss', 'miss']);
  });

  it('marks present letters in wrong positions as present', () => {
    // A in CRANE is present but placed wrongly in GUESS
    expect(scoreGuess('CRANE', 'CACAO')).toEqual(['hit', 'present', 'miss', 'miss', 'miss']);
  });

  it('handles duplicate letters in guess when answer has one occurrence', () => {
    // Answer has one P; guess has two Ps -> only one should be counted present/hit
    expect(scoreGuess('APPLE', 'PAPER')).toEqual([
      'present', // P exists in APPLE but not at pos 0
      'present', // A exists, but not at pos 1
      'hit',     // P at pos 2 is a hit (APPLE[2] = P)
      'miss',    // E not at pos 3 (APPLE[3] = L); we’ll see E later
      'miss',    // R not in APPLE
    ]);
  });

  it('handles duplicate letters when answer also has duplicates', () => {
    // APPLE vs ALLEY: two L’s in guess; APPLE has one L -> only one L should be scored
    expect(scoreGuess('APPLE', 'ALLEY')).toEqual([
      'hit',     // A
      'miss',    // L at pos 1: APPLE[1] = P
      'miss',    // L at pos 2: APPLE[2] = P (only one L total, counted later)
      'present', // E exists in APPLE
      'miss',    // Y not in APPLE
    ]);
  });
});
