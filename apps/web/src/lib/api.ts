/**
 * api.ts
 *
 * Thin wrappers around `fetch` for interacting with the backend API.
 * - Automatically prefixes all requests with `VITE_API_URL`.
 * - Always includes credentials (cookies/session).
 * - Throws an Error with server text if the response is not OK.
 * - Parses and returns JSON body on success.
 */

const API = import.meta.env.VITE_API_URL as string;

/**
 * Perform a GET request to the API and parse JSON.
 *
 * @typeParam T - Expected shape of the JSON response.
 * @param path - API path (must begin with `/`).
 * @returns Parsed JSON object of type `T`.
 * @throws Error if the response status is not OK (non-2xx).
 *
 * @example
 * const stats = await apiGET<Stats>('/stats/me');
 */
export async function apiGET<T>(path: string): Promise<T> {
  const res = await fetch(`${API}${path}`, { credentials: 'include' });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

/**
 * Perform a POST request to the API with optional JSON body.
 *
 * @typeParam T - Expected shape of the JSON response.
 * @param path - API path (must begin with `/`).
 * @param body - Optional body to be JSON-stringified and sent in the request.
 * @returns Parsed JSON object of type `T`.
 * @throws Error if the response status is not OK (non-2xx).
 *
 * @example
 * const game = await apiPOST<{ gameId: string }>('/game/new', { mode: 'normal' });
 */
export async function apiPOST<T>(path: string, body?: any): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}
