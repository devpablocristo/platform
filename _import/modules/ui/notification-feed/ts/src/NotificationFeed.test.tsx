// @vitest-environment jsdom

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { NotificationFeed } from './NotificationFeed';
import {
  buildHandoffUserMessage,
  toAINotificationFeedItem,
  toApprovalNotificationFeedItem,
} from './notificationModels';

describe('NotificationFeed', () => {
  it('renders the loading state before the empty state', () => {
    render(
      <NotificationFeed
        items={[]}
        loading
        loadingMessage="Loading feed"
        emptyMessage="No notifications"
      />,
    );

    expect(screen.getByText('Loading feed')).toBeTruthy();
    expect(screen.queryByText('No notifications')).toBeNull();
  });

  it('renders summary, error and rich notification cards with tone and unread state', () => {
    render(
      <NotificationFeed
        summary="2 unread notifications"
        error="Temporary sync issue"
        emptyMessage="No notifications"
        items={[
          {
            id: 'notif-1',
            title: 'Critical approval',
            body: 'A request is blocked waiting for action.',
            meta: 'Operations',
            timestamp: '10:35',
            badge: 'High',
            actions: <button type="button">Resolve</button>,
            extra: 'Request #42',
            tone: 'critical',
            unread: true,
          },
        ]}
      />,
    );

    expect(screen.getByText('Temporary sync issue')).toBeTruthy();
    expect(screen.getByText('2 unread notifications')).toBeTruthy();
    expect(screen.getByText('Critical approval')).toBeTruthy();
    expect(screen.getByText('A request is blocked waiting for action.')).toBeTruthy();
    expect(screen.getByText('Operations')).toBeTruthy();
    expect(screen.getByText('10:35')).toBeTruthy();
    expect(screen.getByText('High')).toBeTruthy();
    expect(screen.getByText('Request #42')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Resolve' })).toBeTruthy();

    const item = screen.getByText('Critical approval').closest('li');
    expect(item).toBeTruthy();
    expect(item?.className).toContain('m-notification-feed__card--critical');
    expect(item?.className).toContain('m-notification-feed__card--unread');
  });

  it('renders the empty state when there are no items and the feed is idle', () => {
    render(<NotificationFeed items={[]} emptyMessage="No notifications" />);

    expect(screen.getByText('No notifications')).toBeTruthy();
  });

  it('builds a reusable handoff message and maps AI notifications', () => {
    const item = toAINotificationFeedItem({
      id: 'insight-1',
      title: 'Costo alto',
      body: 'Se detecto un desvio de costos.',
      severity: 90,
      unread: true,
      scope: 'cost_overrun',
    });

    expect(item.badge).toBe('Alta');
    expect(item.tone).toBe('critical');
    expect(item.eyebrow).toBe('cost_overrun');
    expect(
      buildHandoffUserMessage({
        notificationId: 'insight-1',
        title: 'Costo alto',
        body: 'Se detecto un desvio de costos.',
      }),
    ).toContain('Costo alto');
  });

  it('maps approval notifications with risk tone', () => {
    const item = toApprovalNotificationFeedItem({
      id: 'approval-1',
      title: 'Aprobacion pendiente',
      body: 'Una accion sensible espera revision.',
      riskLevel: 'high',
      status: 'pending',
      unread: true,
    });

    expect(item.tone).toBe('critical');
    expect(item.badge).toBe('pending');
    expect(item.unread).toBe(true);
  });
});
