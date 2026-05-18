/**
 * Resalta solo la intersección día × franja horaria en vistas timeGrid (semana/día).
 * Usa geometría (no solo elementsFromPoint) para que funcione sobre turnos (.fc-timegrid-event).
 */

const CROSSHAIR_COL = "fc-crosshair-col";
const CROSSHAIR_ROW = "fc-crosshair-row";

function isTimeGridBody(root: HTMLElement): boolean {
  return Boolean(root.querySelector(".fc-timeGridWeek-view, .fc-timeGridDay-view"));
}

function findTimeGridColAtX(root: HTMLElement, clientX: number): HTMLTableCellElement | null {
  const cols = root.querySelectorAll<HTMLTableCellElement>(
    ".fc-timegrid-col[data-date]:not(.fc-timegrid-axis)",
  );
  for (const col of cols) {
    const r = col.getBoundingClientRect();
    if (clientX >= r.left && clientX < r.right) {
      return col;
    }
  }
  return null;
}

/** La celda de franja horaria va ancha (toda la grilla); alinear solo por Y evita fallos sobre el eje de horas. */
function findSlotLaneAtY(root: HTMLElement, clientY: number): HTMLTableCellElement | null {
  const lanes = root.querySelectorAll<HTMLTableCellElement>("td.fc-timegrid-slot-lane[data-time]");
  for (const lane of lanes) {
    const r = lane.getBoundingClientRect();
    if (clientY >= r.top && clientY < r.bottom) {
      return lane;
    }
  }
  return null;
}

/** Si el puntero está sobre un evento, la columna sigue siendo la del ancho del evento. */
function resolveCol(root: HTMLElement, clientX: number, clientY: number): HTMLTableCellElement | null {
  const fromX = findTimeGridColAtX(root, clientX);
  if (fromX) {
    return fromX;
  }
  if (typeof document === "undefined") {
    return null;
  }
  const stack = document.elementsFromPoint(clientX, clientY);
  for (const el of stack) {
    if (!(el instanceof Element)) {
      continue;
    }
    const hit = el.closest(".fc-timegrid-event");
    if (hit) {
      const col = hit.closest(".fc-timegrid-col[data-date]");
      if (col instanceof HTMLTableCellElement && !col.classList.contains("fc-timegrid-axis")) {
        return col;
      }
    }
  }
  return null;
}

export function clearTimeGridCrosshair(root: HTMLElement | null): void {
  if (!root) {
    return;
  }
  root.querySelectorAll(`.${CROSSHAIR_COL}`).forEach((el) => {
    el.classList.remove(CROSSHAIR_COL);
  });
  root.querySelectorAll(`.${CROSSHAIR_ROW}`).forEach((el) => {
    el.classList.remove(CROSSHAIR_ROW);
  });
  root.querySelectorAll<HTMLElement>(".fc-timegrid-col[data-date]").forEach((col) => {
    col.style.removeProperty("--fc-crosshair-slot-top");
    col.style.removeProperty("--fc-crosshair-slot-height");
  });
}

/**
 * Actualiza clases y variables CSS del crosshair. No-op fuera de timeGrid.
 */
export function updateTimeGridCrosshair(root: HTMLElement | null, clientX: number, clientY: number): void {
  if (!root || typeof document === "undefined") {
    return;
  }
  if (!isTimeGridBody(root)) {
    clearTimeGridCrosshair(root);
    return;
  }

  const col = resolveCol(root, clientX, clientY);
  const lane = findSlotLaneAtY(root, clientY);

  clearTimeGridCrosshair(root);

  if (!col || !lane) {
    return;
  }

  const date = col.dataset.date;
  if (!date) {
    return;
  }

  const frame = col.querySelector<HTMLElement>(".fc-timegrid-col-frame");
  if (!frame) {
    return;
  }

  const fr = frame.getBoundingClientRect();
  const lr = lane.getBoundingClientRect();
  const top = lr.top - fr.top;
  const height = lr.height;

  col.style.setProperty("--fc-crosshair-slot-top", `${Math.max(0, top)}px`);
  col.style.setProperty("--fc-crosshair-slot-height", `${Math.max(0, height)}px`);
  col.classList.add(CROSSHAIR_COL);

  const header = root.querySelector<HTMLElement>(`.fc-col-header-cell[data-date="${date}"]`);
  header?.classList.add(CROSSHAIR_COL);

  const row = lane.closest("tr");
  row?.classList.add(CROSSHAIR_ROW);
}
