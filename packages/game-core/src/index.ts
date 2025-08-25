// packages/game-core/src/index.ts
//
// Entry point for the game-core package.
// Re-exports all core game logic so consumers can import from one place.
//
// Includes:
//   • scoring.ts      → Wordle scoring algorithm (scoreGuess, Mark type)
//   • cheatingHost.ts → "Cheating Host" mode logic (nextCheatingCandidates)
//   • words.ts        → Word list utilities (random word, isAllowed, etc.)
//
// Example usage:
//   import { scoreGuess, nextCheatingCandidates, RandomAnswer } from '@wordle/game-core';

export * from './scoring.js';
export * from './cheatingHost.js';
export * from './words.js';
