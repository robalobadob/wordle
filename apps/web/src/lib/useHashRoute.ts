import { useSyncExternalStore } from 'react';

/**
 * useHashRoute
 *
 * A custom React hook providing a **stable hash router**.
 * It ensures React re-renders whenever the location hash changes, including:
 * - Native `hashchange` events (typing URL, anchor navigation).
 * - Programmatic history changes (`history.pushState`, `history.back/forward`).
 * - User clicks on `<a href="#/...">` links, even when nested inside elements.
 *
 * Implementation details:
 * - Uses React 18's `useSyncExternalStore` for a stable subscription model.
 * - Subscribes to `hashchange`, `popstate`, and `click` events.
 * - On click, inspects whether the event target was inside an anchor with a `#` href.
 *   If so, queues a microtask to notify React after the hash is updated.
 *
 * @returns Current hash string (defaults to "#/" if empty).
 *
 * @example
 * const hash = useHashRoute();
 * if (hash === '#/auth') return <AuthPage />;
 */
export function useHashRoute(): string {
  /**
   * Subscribe callback: registers/unregisters listeners for all events
   * that can change the hash.
   */
  const subscribe = (onStoreChange: () => void) => {
    const onHash = () => onStoreChange();

    // Handle clicks on <a href="#..."> links (nested targets included).
    // Use queueMicrotask so state updates after browser updates hash.
    const onClick = (e: Event) => {
      const t = e.target as HTMLElement | null;
      const link = t?.closest?.('a[href^="#"]') as HTMLAnchorElement | null;
      if (link) queueMicrotask(onStoreChange);
    };

    // Capture back/forward navigation (not always covered by hashchange).
    const onPop = () => onStoreChange();

    window.addEventListener('hashchange', onHash);
    window.addEventListener('popstate', onPop);
    document.addEventListener('click', onClick, true); // capture phase

    // Cleanup: remove listeners when the hook unsubscribes
    return () => {
      window.removeEventListener('hashchange', onHash);
      window.removeEventListener('popstate', onPop);
      document.removeEventListener('click', onClick, true);
    };
  };

  /** Snapshot: returns the current hash (or "#/" if none). */
  const getSnapshot = () => window.location.hash || '#/';

  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
