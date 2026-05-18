/**
 * Interfaz genérica para error tracking (Sentry, Datadog, LogRocket, etc.).
 * El producto inyecta su implementación concreta.
 */
export type ErrorReporter = {
  init: () => void;
  captureException: (error: unknown, context?: Record<string, string>) => void;
};

let reporter: ErrorReporter | null = null;

/**
 * Registra un error reporter. Llamar una sola vez al inicio de la app.
 */
export function registerErrorReporter(r: ErrorReporter): void {
  reporter = r;
  reporter.init();
}

/**
 * Captura un error. No-op si no hay reporter registrado.
 */
export function captureError(error: unknown, context?: Record<string, string>): void {
  reporter?.captureException(error, context);
}

/**
 * Crea un error reporter para Sentry a partir de un DSN.
 * Retorna null si el DSN está vacío (desarrollo local).
 * El caller debe pasar el módulo @sentry/* para evitar la dependencia en core.
 */
export function createSentryReporter(
  dsn: string | undefined,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- interfaz genérica compatible con cualquier SDK de error tracking
  sentry: { init: (...args: any[]) => any; captureException: (...args: any[]) => any },
  environment: string,
): ErrorReporter | null {
  if (!dsn) return null;
  return {
    init: () =>
      sentry.init({
        dsn,
        environment,
        tracesSampleRate: 0,
        beforeSend: (event: unknown) => {
          const e = event as { exception?: { values?: Array<{ value?: string }> } };
          const message = e?.exception?.values?.[0]?.value ?? '';
          if (message.includes('401') || message.includes('NetworkError')) return null;
          return event;
        },
      }),
    captureException: (error, context) =>
      sentry.captureException(error, context ? { tags: context } : undefined),
  };
}
