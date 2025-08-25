// packages/game-core/src/words.ts
//
// Static word lists (legacy).
// This file defines two arrays used in early development:
//
//   • ANSWERS → hardcoded list of possible solutions
//   • ALLOWED → valid guess list (currently identical to ANSWERS)
//
// Note: This is a legacy module. In the Go server
// and daily challenge logic, canonical word lists are loaded dynamically from
// embedded assets or external files (see apps/go-server/internal/words/*).
// This file may no longer be used.
//
// It can still serve as a simple fallback for testing the game-core
// package in isolation, without needing server-provided dictionaries.

export const ANSWERS = [
  'HELLO',
  'WORLD',
  'QUITE',
  'FANCY',
  'FRESH',
  'PANIC',
  'CRAZY',
  'BUGGY',
  'SCARE',
  'SWEET',
  'BREAD',
  'APPLE',
  'GRAPE',
  'LEMON',
  'PEACH',
  'PLUMS',
  'CHERRY',
  'MANGO',
  'KIWI',
  'BERRY',
  'PEAR',
  'ORANGE',
  'BANAN',
  'TIGER',
  'LION',
  'BEARS',
  'WOLFS',
  'SHARKS',
  'WHALE',
  'DOLPH',
  'FISHY',
  'OCEAN',
  'RIVER',
  'LAKEY',
  'MOUNT',
  'HILLS',
  'VALLE',
  'CLOUDS',
  'RAINY',
  'SUNNY',
  'STORMY',
  'SNOWY',
];

// For now, ALLOWED = ANSWERS, but in full implementations
// this would include a much larger guess dictionary.
export const ALLOWED = ANSWERS;
