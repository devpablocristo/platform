import type { CalendarOptions } from "@fullcalendar/core";
import FullCalendar from "@fullcalendar/react";
import dayGridPlugin from "@fullcalendar/daygrid";
import interactionPlugin from "@fullcalendar/interaction";
import listPlugin from "@fullcalendar/list";
import luxonPlugin from "@fullcalendar/luxon";
import timeGridPlugin from "@fullcalendar/timegrid";
import type { PointerEvent, ReactNode, RefObject } from "react";
import { useCallback, useEffect, useRef } from "react";
import { clearTimeGridCrosshair, updateTimeGridCrosshair } from "./timeGridCrosshair";

export type CalendarView = "dayGridMonth" | "timeGridWeek" | "timeGridDay" | "listWeek" | "listMonth";

export type CalendarViewOption = {
  value: CalendarView;
  label: string;
};

type EmbeddedCalendarOptions = Omit<
  CalendarOptions,
  | "plugins"
  | "initialView"
  | "headerToolbar"
  | "locale"
  | "height"
  | "slotMinTime"
  | "slotMaxTime"
  | "scrollTime"
  | "scrollTimeReset"
  | "allDayText"
  | "noEventsText"
  | "nowIndicator"
  | "slotDuration"
  | "eventTimeFormat"
  | "events"
  | "editable"
  | "selectable"
  | "eventDurationEditable"
  | "dayMaxEvents"
  | "weekends"
  | "eventClick"
  | "eventDrop"
  | "eventResize"
  | "dateClick"
  | "select"
  | "datesSet"
  | "selectMirror"
  | "dragScroll"
  | "stickyHeaderDates"
  | "navLinks"
  | "eventResizableFromStart"
  | "snapDuration"
  | "moreLinkClick"
  | "businessHours"
  | "selectAllow"
  | "eventAllow"
  | "eventConstraint"
  | "eventContent"
  | "timeZone"
>;

export type CalendarSurfaceProps = {
  calendarRef: RefObject<FullCalendar | null>;
  view: CalendarView;
  title?: string;
  loaded: boolean;
  onToday: () => void;
  onPrev: () => void;
  onNext: () => void;
  onViewChange: (view: CalendarView) => void;
  calendarOptions?: EmbeddedCalendarOptions;
  loadingFallback?: ReactNode;
  viewOptions?: readonly CalendarViewOption[];
  locale?: string;
  slotMinTime?: string;
  slotMaxTime?: string;
  scrollTime?: string;
  scrollTimeReset?: boolean;
  slotDuration?: string;
  className?: string;
  events?: CalendarOptions["events"];
  editable?: boolean;
  selectable?: boolean;
  eventDurationEditable?: boolean;
  dayMaxEvents?: boolean;
  weekends?: boolean;
  onEventClick?: CalendarOptions["eventClick"];
  onEventDrop?: CalendarOptions["eventDrop"];
  onEventResize?: CalendarOptions["eventResize"];
  onDateClick?: CalendarOptions["dateClick"];
  onSelect?: CalendarOptions["select"];
  onDatesSet?: CalendarOptions["datesSet"];
  selectMirror?: boolean;
  dragScroll?: boolean;
  stickyHeaderDates?: boolean;
  navLinks?: boolean;
  eventResizableFromStart?: boolean;
  snapDuration?: string;
  moreLinkClick?: CalendarOptions["moreLinkClick"];
  businessHours?: CalendarOptions["businessHours"];
  selectAllow?: CalendarOptions["selectAllow"];
  eventAllow?: CalendarOptions["eventAllow"];
  eventConstraint?: CalendarOptions["eventConstraint"];
  eventContent?: CalendarOptions["eventContent"];
  /** Contenido opcional a la derecha del grupo de vistas (p. ej. acciones de agenda). */
  toolbarTrailing?: ReactNode;
  /**
   * Zona IANA (p. ej. America/Argentina/Buenos_Aires). Con valor no vacío se registra el plugin Luxon
   * para que selección, arrastre y horario de negocio coincidan con el backend (no con la zona del navegador).
   */
  timeZone?: string | null;
};

const defaultViewOptions: readonly CalendarViewOption[] = [
  { value: "timeGridDay", label: "Día" },
  { value: "timeGridWeek", label: "Semana" },
  { value: "dayGridMonth", label: "Mes" },
  { value: "listWeek", label: "Lista semana" },
  { value: "listMonth", label: "Lista mes" },
];

export function CalendarSurface({
  calendarRef,
  view,
  title,
  loaded,
  onToday,
  onPrev,
  onNext,
  onViewChange,
  calendarOptions = {},
  loadingFallback,
  viewOptions = defaultViewOptions,
  locale = "es",
  slotMinTime = "00:00:00",
  slotMaxTime = "24:00:00",
  scrollTime = "00:00:00",
  scrollTimeReset = false,
  slotDuration = "00:30:00",
  className = "",
  events,
  editable = false,
  selectable = false,
  eventDurationEditable = false,
  dayMaxEvents = true,
  weekends = true,
  onEventClick,
  onEventDrop,
  onEventResize,
  onDateClick,
  onSelect,
  onDatesSet,
  selectMirror = true,
  dragScroll = true,
  stickyHeaderDates = true,
  /** En false evita previews raros y enlaces en cabeceras; el crosshair lo pintamos nosotros. */
  navLinks = false,
  eventResizableFromStart = true,
  snapDuration = slotDuration,
  moreLinkClick = "popover",
  businessHours,
  selectAllow,
  eventAllow,
  eventConstraint,
  eventContent,
  toolbarTrailing,
  timeZone: timeZoneProp,
}: CalendarSurfaceProps) {
  const bodyRef = useRef<HTMLDivElement | null>(null);
  const crosshairRafRef = useRef<number | null>(null);
  const crosshairPointRef = useRef<{ x: number; y: number } | null>(null);

  const flushTimeGridCrosshair = useCallback(() => {
    crosshairRafRef.current = null;
    const p = crosshairPointRef.current;
    if (!p || !bodyRef.current) {
      return;
    }
    updateTimeGridCrosshair(bodyRef.current, p.x, p.y);
  }, []);

  const onCalendarBodyPointerMove = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      crosshairPointRef.current = { x: event.clientX, y: event.clientY };
      if (crosshairRafRef.current != null) {
        return;
      }
      crosshairRafRef.current = window.requestAnimationFrame(flushTimeGridCrosshair);
    },
    [flushTimeGridCrosshair],
  );

  const onCalendarBodyPointerLeave = useCallback(() => {
    crosshairPointRef.current = null;
    if (crosshairRafRef.current != null) {
      window.cancelAnimationFrame(crosshairRafRef.current);
      crosshairRafRef.current = null;
    }
    clearTimeGridCrosshair(bodyRef.current);
  }, []);

  useEffect(() => {
    return () => {
      if (crosshairRafRef.current != null) {
        window.cancelAnimationFrame(crosshairRafRef.current);
      }
      clearTimeGridCrosshair(bodyRef.current);
    };
  }, []);

  const resolvedTimeZone = timeZoneProp?.trim() ?? "";
  const useNamedTimeZone = resolvedTimeZone.length > 0;
  const fullCalendarProps = {
    plugins: useNamedTimeZone
      ? [dayGridPlugin, timeGridPlugin, interactionPlugin, listPlugin, luxonPlugin]
      : [dayGridPlugin, timeGridPlugin, interactionPlugin, listPlugin],
    ...(useNamedTimeZone ? { timeZone: resolvedTimeZone } : {}),
    initialView: view,
    headerToolbar: false,
    locale,
    height: "100%",
    slotMinTime,
    slotMaxTime,
    scrollTime,
    scrollTimeReset,
    allDaySlot: false,
    allDayText: "",
    noEventsText: "",
    nowIndicator: true,
    slotDuration,
    slotLabelInterval: "00:30:00",
    slotLabelFormat: { hour: "2-digit", minute: "2-digit", hour12: false },
    eventTimeFormat: { hour: "2-digit", minute: "2-digit", hour12: false },
    events,
    editable,
    selectable,
    eventDurationEditable,
    dayMaxEvents,
    weekends,
    eventClick: onEventClick,
    eventDrop: onEventDrop,
    eventResize: onEventResize,
    dateClick: onDateClick,
    select: onSelect,
    datesSet: onDatesSet,
    selectMirror,
    dragScroll,
    stickyHeaderDates,
    navLinks,
    eventResizableFromStart,
    snapDuration,
    moreLinkClick,
    businessHours,
    selectAllow,
    eventAllow,
    eventConstraint,
    eventContent,
    ...calendarOptions,
  } as CalendarOptions;

  return (
    <div className={`modules-calendar ${className}`.trim()}>
      <div className="modules-calendar__toolbar">
        <div className="modules-calendar__toolbar-left">
          <button type="button" className="modules-calendar__today-btn" onClick={onToday}>
            Hoy
          </button>
          <button type="button" className="modules-calendar__nav-btn" onClick={onPrev}>
            ‹
          </button>
          <button type="button" className="modules-calendar__nav-btn" onClick={onNext}>
            ›
          </button>
        </div>
        {title ? (
          <div className="modules-calendar__toolbar-center">
            <h2 className="modules-calendar__title">{title}</h2>
          </div>
        ) : null}
        <div className="modules-calendar__toolbar-right">
          <div className="modules-calendar__view-group">
            {viewOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                className={`modules-calendar__view-btn ${view === option.value ? "modules-calendar__view-btn--active" : ""}`}
                onClick={() => onViewChange(option.value)}
              >
                {option.label}
              </button>
            ))}
          </div>
          {toolbarTrailing ? <div className="modules-calendar__toolbar-trailing">{toolbarTrailing}</div> : null}
        </div>
      </div>

      <div
        ref={bodyRef}
        className="modules-calendar__body"
        onPointerMove={onCalendarBodyPointerMove}
        onPointerLeave={onCalendarBodyPointerLeave}
        onPointerCancel={onCalendarBodyPointerLeave}
      >
        {!loaded ? (
          loadingFallback ?? (
            <div style={{ padding: "3rem", textAlign: "center" }}>
              <div className="spinner" />
            </div>
          )
        ) : (
          <FullCalendar ref={calendarRef} {...fullCalendarProps} />
        )}
      </div>
    </div>
  );
}

export default CalendarSurface;
