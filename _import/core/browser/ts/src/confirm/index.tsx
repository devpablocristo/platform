import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";

export type ConfirmDialogTone = "primary" | "danger";

export type ConfirmDialogOptions = {
  title?: ReactNode;
  description?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: ConfirmDialogTone;
  allowEscapeClose?: boolean;
  allowBackdropClose?: boolean;
};

type ConfirmRequest = Required<
  Pick<ConfirmDialogOptions, "confirmLabel" | "cancelLabel" | "tone" | "allowEscapeClose" | "allowBackdropClose">
> &
  Omit<ConfirmDialogOptions, "confirmLabel" | "cancelLabel" | "tone" | "allowEscapeClose" | "allowBackdropClose"> & {
    id: number;
    resolve: (confirmed: boolean) => void;
  };

type ConfirmOpenHandler = (options: ConfirmDialogOptions) => Promise<boolean>;

let confirmOpenHandler: ConfirmOpenHandler | null = null;
let confirmRequestId = 0;

function normalizeConfirmOptions(options: ConfirmDialogOptions, resolve: (confirmed: boolean) => void): ConfirmRequest {
  return {
    id: ++confirmRequestId,
    title: options.title,
    description: options.description,
    confirmLabel: options.confirmLabel ?? "Confirmar",
    cancelLabel: options.cancelLabel ?? "Cancelar",
    tone: options.tone ?? "primary",
    allowEscapeClose: options.allowEscapeClose ?? true,
    allowBackdropClose: options.allowBackdropClose ?? true,
    resolve,
  };
}

function fallbackConfirmMessage(options: ConfirmDialogOptions): string {
  const title = typeof options.title === "string" ? options.title : "";
  const description = typeof options.description === "string" ? options.description : "";
  return [title, description].filter(Boolean).join("\n\n") || "¿Confirmar acción?";
}

export function confirmAction(options: ConfirmDialogOptions): Promise<boolean> {
  if (confirmOpenHandler) {
    return confirmOpenHandler(options);
  }
  if (typeof window !== "undefined" && typeof window.confirm === "function") {
    return Promise.resolve(window.confirm(fallbackConfirmMessage(options)));
  }
  return Promise.resolve(false);
}

export function ConfirmDialogProvider({ children }: { children: ReactNode }) {
  const [queue, setQueue] = useState<ConfirmRequest[]>([]);
  const activeRequestRef = useRef<ConfirmRequest | null>(null);

  const openConfirm = useCallback((options: ConfirmDialogOptions) => {
    return new Promise<boolean>((resolve) => {
      setQueue((current) => [...current, normalizeConfirmOptions(options, resolve)]);
    });
  }, []);

  useEffect(() => {
    confirmOpenHandler = openConfirm;
    return () => {
      if (confirmOpenHandler === openConfirm) {
        confirmOpenHandler = null;
      }
    };
  }, [openConfirm]);

  const activeRequest = queue[0] ?? null;

  useEffect(() => {
    activeRequestRef.current = activeRequest;
  }, [activeRequest]);

  const settleActiveRequest = useCallback((confirmed: boolean) => {
    const request = activeRequestRef.current;
    if (!request) {
      return;
    }
    activeRequestRef.current = null;
    request.resolve(confirmed);
    setQueue((current) => current.slice(1));
  }, []);

  useEffect(() => {
    if (!activeRequest?.allowEscapeClose) {
      return;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") {
        return;
      }
      event.preventDefault();
      settleActiveRequest(false);
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [activeRequest, settleActiveRequest]);

  const dialog = useMemo(() => {
    if (!activeRequest) {
      return null;
    }
    const titleId = `confirm-dialog-title-${activeRequest.id}`;
    const descriptionId = `confirm-dialog-description-${activeRequest.id}`;

    return (
      <div
        className="confirm-dialog__backdrop"
        role="presentation"
        onMouseDown={(event) => {
          if (event.target !== event.currentTarget || !activeRequest.allowBackdropClose) {
            return;
          }
          settleActiveRequest(false);
        }}
      >
        <section
          className="confirm-dialog"
          role="alertdialog"
          aria-modal="true"
          aria-labelledby={titleId}
          aria-describedby={activeRequest.description != null ? descriptionId : undefined}
          onMouseDown={(event) => event.stopPropagation()}
        >
          <div className="confirm-dialog__header">
            <h2 id={titleId} className="confirm-dialog__title">
              {activeRequest.title ?? "Confirmar acción"}
            </h2>
          </div>
          {activeRequest.description != null ? (
            <div className="confirm-dialog__body">
              <p id={descriptionId} className="confirm-dialog__description">
                {activeRequest.description}
              </p>
            </div>
          ) : null}
          <div className="confirm-dialog__footer">
            <button type="button" className="btn-secondary btn-sm" onClick={() => settleActiveRequest(false)}>
              {activeRequest.cancelLabel}
            </button>
            <button
              type="button"
              className={`${activeRequest.tone === "danger" ? "btn-danger" : "btn-primary"} btn-sm`}
              onClick={() => settleActiveRequest(true)}
            >
              {activeRequest.confirmLabel}
            </button>
          </div>
        </section>
      </div>
    );
  }, [activeRequest, settleActiveRequest]);

  return (
    <>
      {children}
      {dialog && typeof document !== "undefined" ? createPortal(dialog, document.body) : dialog}
    </>
  );
}
