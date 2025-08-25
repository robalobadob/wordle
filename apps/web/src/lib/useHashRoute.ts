import { useSyncExternalStore } from 'react';

/**
 * Stable hash router: re-renders on hashchange, programmatic changes,
 * and even when clicks hit <a href="#/..."> inside nested elements.
 */
export function useHashRoute(): string {
  const subscribe = (onStoreChange: () => void) => {
    const onHash = () => onStoreChange();

    // Some setups change hash via click handlers or nested targets;
    // listen to document clicks and schedule a microtask check.
    const onClick = (e: Event) => {
      const t = e.target as HTMLElement | null;
      const link = t?.closest?.('a[href^="#"]') as HTMLAnchorElement | null;
      if (link) queueMicrotask(onStoreChange);
    };

    // Also catch back/forward in some browsers
    const onPop = () => onStoreChange();

    window.addEventListener('hashchange', onHash);
    window.addEventListener('popstate', onPop);
    document.addEventListener('click', onClick, true); // capture phase

    return () => {
      window.removeEventListener('hashchange', onHash);
      window.removeEventListener('popstate', onPop);
      document.removeEventListener('click', onClick, true);
    };
  };

  const getSnapshot = () => window.location.hash || '#/';
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
