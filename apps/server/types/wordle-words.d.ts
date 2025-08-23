declare module 'wordle-words' {
  export const allSolutions: string[] | undefined;
  export const allGuesses: string[] | undefined;
  // Some packages ship CommonJS. When dynamically imported, you'll see `default`.
  const _default:
    | { allSolutions?: string[]; allGuesses?: string[] }
    | undefined;
  export default _default;
}
