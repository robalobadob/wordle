import { nextCheatingCandidates, scoreGuess } from '../index';

type Mark = 'hit' | 'present' | 'miss';

function maskKey(marks: ReadonlyArray<Mark>): string {
  return marks.join('|');
}

/** Compute the maximal bucket size achievable for a given candidates+guess. */
function maxBucketSize(candidates: string[], guess: string): number {
  const counts = new Map<string, number>();
  for (const w of candidates) {
    const key = maskKey(scoreGuess(w, guess) as Mark[]);
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  let max = 0;
  for (const v of counts.values()) max = Math.max(max, v);
  return max;
}

/** Ensure all words in arr are 5-letter lowercase strings. */
function isWordArray(arr: unknown): arr is string[] {
  return (
    Array.isArray(arr) &&
    arr.every(
      (s) => typeof s === 'string' && s.length === 5 && /^[a-z]+$/.test(s),
    )
  );
}

/** Normalize possible return shapes from nextCheatingCandidates() to string[]. */
function normalizeCandidates(ret: unknown): string[] {
  if (Array.isArray(ret)) return ret;
  if (ret instanceof Set) return Array.from(ret);
  if (ret instanceof Map) {
    // choose the largest array value
    let best: string[] = [];
    for (const arr of ret.values()) {
      if (Array.isArray(arr) && arr.length >= best.length)
        best = arr as string[];
    }
    return best;
  }
  if (ret && typeof ret === 'object') {
    const r = ret as Record<string, unknown>;

    // Try obvious keys
    for (const k of [
      'candidates',
      'words',
      'bucket',
      'best',
      'next',
      'result',
      'partition',
      'value',
    ]) {
      const v = r[k];
      if (isWordArray(v)) return v;
      if (v && typeof v === 'object' && isWordArray((v as any).words))
        return (v as any).words;
    }

    // Try bucketed shapes
    for (const k of ['buckets', 'partitions', 'byMask', 'groups']) {
      const v = r[k] as Record<string, unknown> | undefined;
      if (v && typeof v === 'object') {
        let best: string[] = [];
        for (const arr of Object.values(v)) {
          if (isWordArray(arr) && arr.length >= best.length) best = arr;
        }
        if (best.length) return best;
      }
    }

    // Fallback: the longest 5-letter string array among values
    let best: string[] = [];
    for (const v of Object.values(r)) {
      if (isWordArray(v) && v.length >= best.length) best = v;
    }
    if (best.length) return best;
  }
  throw new Error('Unexpected nextCheatingCandidates() return shape');
}

describe('nextCheatingCandidates (cheating mode)', () => {
  const dict = ['crane', 'slate', 'plant', 'clang', 'glare', 'grace'];

  it('returns a maximal bucket where all words share the same mask vs the guess', () => {
    const guess = 'crane';

    const raw = nextCheatingCandidates(dict, guess);
    const next = normalizeCandidates(raw);

    // Non-empty and shaped like words
    expect(isWordArray(next)).toBe(true);
    expect(next.length).toBeGreaterThan(0);

    // All masks equal
    const masks = new Set(
      next.map((w) => maskKey(scoreGuess(w, guess) as Mark[])),
    );
    expect(masks.size).toBe(1);

    // Maximal size among all partitions
    const maxSize = maxBucketSize(dict, guess);
    expect(next.length).toBe(maxSize);
  });

  it('chains: each step yields a maximal single-mask bucket for the given guess', () => {
    const g1 = 'slate';
    const raw1 = nextCheatingCandidates(dict, g1);
    const after1 = normalizeCandidates(raw1);

    expect(isWordArray(after1)).toBe(true);
    expect(after1.length).toBeGreaterThan(0);
    expect(
      new Set(after1.map((w) => maskKey(scoreGuess(w, g1) as Mark[]))).size,
    ).toBe(1);
    expect(after1.length).toBe(maxBucketSize(dict, g1));

    const g2 = 'grace';
    const raw2 = nextCheatingCandidates(after1, g2);
    const after2 = normalizeCandidates(raw2);

    expect(isWordArray(after2)).toBe(true);
    expect(after2.length).toBeGreaterThan(0);
    expect(
      new Set(after2.map((w) => maskKey(scoreGuess(w, g2) as Mark[]))).size,
    ).toBe(1);
    expect(after2.length).toBe(maxBucketSize(after1, g2));
  });
});
