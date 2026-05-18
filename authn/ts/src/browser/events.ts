export const AUTH_FORCE_LOGOUT_EVENT = "auth:force-logout";

type EventDispatcher = Pick<Window, "dispatchEvent"> | EventTarget;

function resolveTarget(target?: EventDispatcher): EventDispatcher | null {
  if (target) {
    return target;
  }
  if (typeof window === "undefined") {
    return null;
  }
  return window;
}

export function dispatchAuthForceLogout(target?: EventDispatcher): void {
  const dispatcher = resolveTarget(target);
  if (!dispatcher) {
    return;
  }
  dispatcher.dispatchEvent(new CustomEvent(AUTH_FORCE_LOGOUT_EVENT));
}
