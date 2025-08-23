import { scoreGuess, type Mark } from './scoring.js';

function score(marks: Mark[]): [hits: number, presents: number] {
  let h = 0,
    p = 0;
  for (const m of marks) {
    if (m === 'hit') h++;
    else if (m === 'present') p++;
  }
  return [h, p];
}

export function nextCheatingCandidates(candidates: string[], guess: string) {
  type Bucket = { marks: Mark[]; words: string[]; s: [number, number] };
  const buckets = new Map<string, Bucket>();
  for (const ans of candidates) {
    const marks = scoreGuess(ans, guess);
    const key = marks
      .map((m) => (m === 'hit' ? 'O' : m === 'present' ? '?' : '_'))
      .join('');
    const b = buckets.get(key) ?? { marks, words: [], s: score(marks) };
    b.words.push(ans);
    buckets.set(key, b);
  }
  const chosen = [...buckets.values()].sort(
    (a, b) => a.s[0] - b.s[0] || a.s[1] - b.s[1],
  )[0];
  return { next: chosen.words, marks: chosen.marks };
}
