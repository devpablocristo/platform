// @vitest-environment jsdom

import { fireEvent, render, screen } from '@testing-library/react';
import { forwardRef } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { CalendarSurface } from './CalendarSurfaceBoard';

const fullCalendarProps = vi.hoisted(() => ({
  last: null as Record<string, unknown> | null,
}));

vi.mock('@fullcalendar/react', () => ({
  default: forwardRef(function FullCalendarMock(
    props: {
      initialView: string;
      locale: string;
      dateClick?: (info: { date: Date; dateStr: string }) => void;
      select?: (info: { start: Date; end: Date; startStr: string; endStr: string }) => void;
      eventDrop?: (info: { event: { id: string; startStr: string }; revert: () => void }) => void;
      eventResize?: (info: { revert: () => void }) => void;
    },
    _ref,
  ) {
    fullCalendarProps.last = props;
    return (
      <div data-testid="fullcalendar-mock">
        calendar:{props.initialView}:{props.locale}
        <button
          type="button"
          onClick={() =>
            props.dateClick?.({
              date: new Date('2099-04-05T10:00:00Z'),
              dateStr: '2099-04-05T10:00:00Z',
            })
          }
        >
          trigger-date-click
        </button>
        <button
          type="button"
          onClick={() =>
            props.select?.({
              start: new Date('2099-04-05T10:00:00Z'),
              end: new Date('2099-04-05T10:30:00Z'),
              startStr: '2099-04-05T10:00:00Z',
              endStr: '2099-04-05T10:30:00Z',
            })
          }
        >
          trigger-select
        </button>
        <button
          type="button"
          onClick={() =>
            props.eventDrop?.({
              event: { id: 'evt-1', startStr: '2099-04-05T11:00:00Z' },
              revert: vi.fn(),
            })
          }
        >
          trigger-drop
        </button>
        <button type="button" onClick={() => props.eventResize?.({ revert: vi.fn() })}>
          trigger-resize
        </button>
      </div>
    );
  }),
}));

vi.mock('@fullcalendar/daygrid', () => ({ default: {} }));
vi.mock('@fullcalendar/timegrid', () => ({ default: {} }));
vi.mock('@fullcalendar/interaction', () => ({ default: {} }));
vi.mock('@fullcalendar/list', () => ({ default: {} }));
vi.mock('@fullcalendar/luxon', () => ({ default: {} }));

describe('CalendarSurface', () => {
  it('renders the loading fallback when the calendar is not ready', () => {
    render(
      <CalendarSurface
        calendarRef={{ current: null }}
        view="timeGridWeek"
        loaded={false}
        onToday={vi.fn()}
        onPrev={vi.fn()}
        onNext={vi.fn()}
        onViewChange={vi.fn()}
        calendarOptions={{}}
        loadingFallback={<div>loading calendar</div>}
      />,
    );

    expect(screen.getByText('loading calendar')).toBeTruthy();
    expect(screen.queryByTestId('fullcalendar-mock')).toBeNull();
  });

  it('wires toolbar actions, active view and calendar props', () => {
    const onToday = vi.fn();
    const onPrev = vi.fn();
    const onNext = vi.fn();
    const onViewChange = vi.fn();

    render(
      <CalendarSurface
        calendarRef={{ current: null }}
        view="timeGridWeek"
        title="Agenda"
        loaded
        onToday={onToday}
        onPrev={onPrev}
        onNext={onNext}
        onViewChange={onViewChange}
        locale="es"
        calendarOptions={{}}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Hoy' }));
    fireEvent.click(screen.getByRole('button', { name: '‹' }));
    fireEvent.click(screen.getByRole('button', { name: '›' }));
    fireEvent.click(screen.getByRole('button', { name: 'Mes' }));

    expect(onToday).toHaveBeenCalledTimes(1);
    expect(onPrev).toHaveBeenCalledTimes(1);
    expect(onNext).toHaveBeenCalledTimes(1);
    expect(onViewChange).toHaveBeenCalledWith('dayGridMonth');
    expect(screen.getByText('Agenda')).toBeTruthy();
    expect(screen.getByTestId('fullcalendar-mock').textContent).toContain('calendar:timeGridWeek:es');
    expect(screen.getByRole('button', { name: 'Semana' }).className).toContain('modules-calendar__view-btn--active');
  });

  it('supports list views in the toolbar', () => {
    const onViewChange = vi.fn();

    render(
      <CalendarSurface
        calendarRef={{ current: null }}
        view="listWeek"
        loaded
        onToday={vi.fn()}
        onPrev={vi.fn()}
        onNext={vi.fn()}
        onViewChange={onViewChange}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Lista mes' }));

    expect(onViewChange).toHaveBeenCalledWith('listMonth');
    expect(screen.getByRole('button', { name: 'Lista semana' }).className).toContain('modules-calendar__view-btn--active');
  });

  it('forwards explicit board interaction callbacks', () => {
    const onDateClick = vi.fn();
    const onSelect = vi.fn();
    const onEventDrop = vi.fn();
    const onEventResize = vi.fn();
    const onEventContent = vi.fn();
    const onSelectAllow = vi.fn();
    const onEventAllow = vi.fn();
    const businessHours = [{ daysOfWeek: [1, 2], startTime: '09:00', endTime: '18:00' }];

    render(
      <CalendarSurface
        calendarRef={{ current: null }}
        view="timeGridWeek"
        loaded
        onToday={vi.fn()}
        onPrev={vi.fn()}
        onNext={vi.fn()}
        onViewChange={vi.fn()}
        events={[]}
        editable
        selectable
        eventDurationEditable
        onDateClick={onDateClick}
        onSelect={onSelect}
        onEventDrop={onEventDrop}
        onEventResize={onEventResize}
        businessHours={businessHours}
        selectAllow={onSelectAllow}
        eventAllow={onEventAllow}
        eventConstraint="businessHours"
        eventContent={onEventContent}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'trigger-date-click' }));
    fireEvent.click(screen.getByRole('button', { name: 'trigger-select' }));
    fireEvent.click(screen.getByRole('button', { name: 'trigger-drop' }));
    fireEvent.click(screen.getByRole('button', { name: 'trigger-resize' }));

    expect(onDateClick).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onEventDrop).toHaveBeenCalledTimes(1);
    expect(onEventResize).toHaveBeenCalledTimes(1);
    expect(fullCalendarProps.last?.editable).toBe(true);
    expect(fullCalendarProps.last?.selectable).toBe(true);
    expect(fullCalendarProps.last?.eventDurationEditable).toBe(true);
    expect(fullCalendarProps.last?.selectMirror).toBe(true);
    expect(fullCalendarProps.last?.dragScroll).toBe(true);
    expect(fullCalendarProps.last?.stickyHeaderDates).toBe(true);
    expect(fullCalendarProps.last?.navLinks).toBe(false);
    expect(fullCalendarProps.last?.eventResizableFromStart).toBe(true);
    expect(fullCalendarProps.last?.snapDuration).toBe('00:30:00');
    expect(fullCalendarProps.last?.moreLinkClick).toBe('popover');
    expect(fullCalendarProps.last?.businessHours).toEqual(businessHours);
    expect(fullCalendarProps.last?.selectAllow).toBe(onSelectAllow);
    expect(fullCalendarProps.last?.eventAllow).toBe(onEventAllow);
    expect(fullCalendarProps.last?.eventConstraint).toBe('businessHours');
    expect(fullCalendarProps.last?.eventContent).toBe(onEventContent);
  });

  it('activa zona IANA y plugin Luxon cuando timeZone está definido', () => {
    render(
      <CalendarSurface
        calendarRef={{ current: null }}
        view="timeGridWeek"
        loaded
        onToday={vi.fn()}
        onPrev={vi.fn()}
        onNext={vi.fn()}
        onViewChange={vi.fn()}
        timeZone="America/Argentina/Buenos_Aires"
      />,
    );
    const props = fullCalendarProps.last as { timeZone?: string; plugins?: unknown[] } | null;
    expect(props?.timeZone).toBe('America/Argentina/Buenos_Aires');
    expect(Array.isArray(props?.plugins)).toBe(true);
    expect(props?.plugins?.length).toBe(5);
  });
});
