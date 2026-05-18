// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from "vitest";
import { clearTimeGridCrosshair, updateTimeGridCrosshair } from "./timeGridCrosshair";

function mockRect(el: Element, r: Partial<DOMRect>) {
  const full: DOMRect = {
    x: r.left ?? 0,
    y: r.top ?? 0,
    width: (r.right ?? 0) - (r.left ?? 0),
    height: (r.bottom ?? 0) - (r.top ?? 0),
    top: r.top ?? 0,
    right: r.right ?? 0,
    bottom: r.bottom ?? 0,
    left: r.left ?? 0,
    toJSON() {
      return {};
    },
  };
  vi.spyOn(el, "getBoundingClientRect").mockReturnValue(full);
}

describe("timeGridCrosshair", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("clearTimeGridCrosshair quita clases y variables", () => {
    const root = document.createElement("div");
    root.innerHTML = `
      <div class="fc-timeGridWeek-view">
        <div class="fc-col-header-cell fc-crosshair-col" data-date="2026-04-01">h</div>
        <table><tbody><tr class="fc-crosshair-row"><td class="fc-timegrid-slot-lane" data-time="09:00:00"></td></tr></tbody></table>
        <table><tbody><tr>
          <td class="fc-timegrid-col fc-crosshair-col" data-date="2026-04-01">
            <div class="fc-timegrid-col-frame"></div>
          </td>
        </tr></tbody></table>
      </div>`;
    const col = root.querySelector<HTMLElement>(".fc-timegrid-col")!;
    col.style.setProperty("--fc-crosshair-slot-top", "5px");

    clearTimeGridCrosshair(root);

    expect(root.querySelectorAll(".fc-crosshair-col")).toHaveLength(0);
    expect(root.querySelectorAll(".fc-crosshair-row")).toHaveLength(0);
    expect(col.style.getPropertyValue("--fc-crosshair-slot-top")).toBe("");
  });

  it("updateTimeGridCrosshair marca columna, cabecera, fila y variables de slot", () => {
    const root = document.createElement("div");
    root.innerHTML = `
      <div class="fc-timeGridWeek-view">
        <div class="fc-col-header-cell" data-date="2026-04-08">mié</div>
        <table class="slats"><tbody><tr>
          <td class="fc-timegrid-slot-label">09:00</td>
          <td class="fc-timegrid-slot-lane" data-time="09:00:00"></td>
        </tr></tbody></table>
        <table class="cols"><tbody><tr>
          <td class="fc-timegrid-col" data-date="2026-04-08">
            <div class="fc-timegrid-col-frame"></div>
          </td>
        </tr></tbody></table>
      </div>`;

    const col = root.querySelector<HTMLElement>(".fc-timegrid-col")!;
    const frame = root.querySelector<HTMLElement>(".fc-timegrid-col-frame")!;
    const lane = root.querySelector<HTMLElement>(".fc-timegrid-slot-lane")!;
    const tr = lane.closest("tr")!;

    mockRect(col, { left: 200, right: 280, top: 0, bottom: 600 });
    mockRect(frame, { left: 200, right: 280, top: 100, bottom: 700 });
    mockRect(lane, { left: 0, right: 800, top: 250, bottom: 274 });

    updateTimeGridCrosshair(root, 240, 260);

    expect(col.classList.contains("fc-crosshair-col")).toBe(true);
    expect(
      root.querySelector(".fc-col-header-cell[data-date='2026-04-08']")?.classList.contains("fc-crosshair-col"),
    ).toBe(true);
    expect(tr.classList.contains("fc-crosshair-row")).toBe(true);
    expect(col.style.getPropertyValue("--fc-crosshair-slot-top")).toBe("150px");
    expect(col.style.getPropertyValue("--fc-crosshair-slot-height")).toBe("24px");
  });

  it("sobre un evento resuelve la columna con closest", () => {
    const root = document.createElement("div");
    root.innerHTML = `
      <div class="fc-timeGridWeek-view">
        <div class="fc-col-header-cell" data-date="2026-04-08"></div>
        <table><tbody><tr>
          <td class="fc-timegrid-slot-lane" data-time="10:00:00"></td>
        </tr></tbody></table>
        <table><tbody><tr>
          <td class="fc-timegrid-col" data-date="2026-04-08">
            <div class="fc-timegrid-col-frame">
              <div class="fc-timegrid-event-harness">
                <div class="fc-timegrid-event"><span>Juan</span></div>
              </div>
            </div>
          </td>
        </tr></tbody></table>
      </div>`;

    const col = root.querySelector<HTMLElement>(".fc-timegrid-col")!;
    const frame = root.querySelector<HTMLElement>(".fc-timegrid-col-frame")!;
    const lane = root.querySelector<HTMLElement>(".fc-timegrid-slot-lane")!;
    const eventEl = root.querySelector<HTMLElement>(".fc-timegrid-event")!;

    mockRect(col, { left: 100, right: 180, top: 0, bottom: 500 });
    mockRect(frame, { left: 100, right: 180, top: 80, bottom: 580 });
    mockRect(lane, { left: 0, right: 600, top: 200, bottom: 224 });
    mockRect(eventEl, { left: 110, right: 170, top: 205, bottom: 230 });

    const doc = document as Document & { elementsFromPoint?: (x: number, y: number) => Element[] };
    const prev = doc.elementsFromPoint;
    doc.elementsFromPoint = () => [eventEl];
    try {
      /* X fuera del ancho de la columna pero el hit-test devuelve el evento → closest(.fc-timegrid-col). */
      updateTimeGridCrosshair(root, 50, 210);
      expect(col.classList.contains("fc-crosshair-col")).toBe(true);
    } finally {
      doc.elementsFromPoint = prev;
    }
  });
});
