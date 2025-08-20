export type Mark = 'hit' | 'present' | 'miss';

export function scoreGuess(answer: string, guess: string): Mark[] {
  const A = answer.toUpperCase();
  const G = guess.toUpperCase();
  if (A.length !== 5 || G.length !== 5) throw new Error('Words must be 5 letters');
  if (!/^[A-Z]{5}$/.test(A) || !/^[A-Z]{5}$/.test(G)) throw new Error('Only A–Z letters');

  const marks: Mark[] = Array(5).fill('miss');
  const counts: Record<string, number> = {};

  for (let i = 0; i < 5; i++) {
    if (G[i] === A[i]) marks[i] = 'hit';
    else counts[A[i]] = (counts[A[i]] ?? 0) + 1;
  }
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
