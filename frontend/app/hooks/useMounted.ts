import { useSyncExternalStore } from "react";

const noopSubscribe = () => () => {};

// True on the client after hydration, false during server rendering.
// Drop-in replacement for the setState-in-effect "mounted" pattern.
export function useMounted(): boolean {
  return useSyncExternalStore(
    noopSubscribe,
    () => true,
    () => false
  );
}
