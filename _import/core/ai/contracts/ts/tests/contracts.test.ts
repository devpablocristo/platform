import { describe, it, expect, expectTypeOf } from 'vitest';
import type {
  ChatBlock,
  ChatRequest,
  ChatResponse,
  ConversationSummary,
  ConversationDetail,
  NotificationsRequest,
  NotificationsResponse,
} from '../src/index';

describe('contracts shapes', () => {
  it('ChatBlock currently only allows text variant', () => {
    const block: ChatBlock = { type: 'text', text: 'hola' };
    expect(block.type).toBe('text');
    expect(block.text).toBe('hola');
  });

  it('ChatRequest accepts a minimal message-only payload', () => {
    const req: ChatRequest = { message: 'hola' };
    expect(req.message).toBe('hola');
  });

  it('ChatResponse round-trips through JSON', () => {
    const resp: ChatResponse = {
      chat_id: 'c1',
      reply: 'r',
      blocks: [{ type: 'text', text: 'r' }],
    };
    const raw = JSON.stringify(resp);
    const parsed = JSON.parse(raw) as ChatResponse;
    expect(parsed.chat_id).toBe('c1');
    expect(parsed.blocks?.[0]).toEqual({ type: 'text', text: 'r' });
  });

  it('ConversationSummary requires id, message_count and timestamps', () => {
    const cs: ConversationSummary = {
      id: 'c1',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      message_count: 0,
    };
    expectTypeOf(cs).toMatchTypeOf<ConversationSummary>();
  });

  it('ConversationDetail nests messages with role', () => {
    const cd: ConversationDetail = {
      id: 'c1',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      messages: [{ role: 'user', content: 'hi' }],
    };
    expect(cd.messages[0].role).toBe('user');
  });

  it('NotificationsRequest and Response are open enough', () => {
    const req: NotificationsRequest = { kind: 'insight', period: 'today', compare: true, top_limit: 5 };
    const resp: NotificationsResponse = { items: [{ id: '1', title: 't' }] };
    expect(req.period).toBe('today');
    expect(resp.items).toHaveLength(1);
  });
});
