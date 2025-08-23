import { nextCheatingCandidates, scoreGuess } from '../index';

describe('nextCheatingCandidates (cheating mode)', () => {
  const dict = ['CRANE', 'SLATE', 'PLANT', 'CLANG', 'GLARE', 'GRACE'];

  it('filters candidates to remain consistent with a previous round', () => {
    const guess = 'CRANE';

    // If scoreGuess signature is (answer, guess), keep this order;
    // if it's (guess, answer), swap the parameters.
    const feedback = scoreGuess('GLARE', guess); 

    // Two-arg API: dictionary + history
    const next = (nextCheatingCandidates as any)(
      dict,
      [{ guess, feedback }]
    );

    expect(next).toContain('GLARE');   // still possible
    expect(next).not.toContain('CRANE'); // contradicts feedback, should be removed
  });

  it('supports chaining across multiple rounds (intersection of constraints)', () => {
    const g1 = 'SLATE';
    const f1 = scoreGuess('GLARE', g1);

    const after1 = (nextCheatingCandidates as any)(
      dict,
      [{ guess: g1, feedback: f1 }]
    );

    const g2 = 'GRACE';
    const f2 = scoreGuess('GLARE', g2);

    const after2 = (nextCheatingCandidates as any)(
      after1,
      [{ guess: g2, feedback: f2 }]
    );

    expect(after2).toContain('GLARE');
    expect(after2).not.toContain('CRANE');
  });
});
