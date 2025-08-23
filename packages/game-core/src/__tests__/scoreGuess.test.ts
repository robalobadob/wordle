import { scoreGuess } from '../index';

describe('scoreGuess', () => {
  it('marks exact matches as hits', () => {
    expect(scoreGuess('crane', 'crane')).toEqual(['hit', 'hit', 'hit', 'hit', 'hit']);
  });

  it('marks absent letters as miss', () => {
    expect(scoreGuess('crane', 'bolts')).toEqual(['miss', 'miss', 'miss', 'miss', 'miss']);
  });

  it('marks present letters in wrong positions as present', () => {
    expect(scoreGuess('crane', 'cacao')).toEqual(['hit', 'present', 'miss', 'miss', 'miss']);
  });

  it('handles duplicate letters in guess when answer has duplicates', () => {
    // "apple" has two p’s. "paper" has two p’s too.
    expect(scoreGuess('apple', 'paper')).toEqual([
      'present', // p (pos 0) exists
      'present', // a exists (wrong spot)
      'hit',     // p (pos 2) is exact
      'present', // e exists (wrong spot)
      'miss',    // r not in "apple"
    ]);
  });

  it('handles duplicate letters when answer has a single occurrence', () => {
    // "apple" has one l; "alley" has two. Only one should count.
    expect(scoreGuess('apple', 'alley')).toEqual([
      'hit',      // a
      'present',  // first l gets credit
      'miss',     // second l is exhausted
      'present',  // e present
      'miss',     // y not in "apple"
    ]);
  });
});
