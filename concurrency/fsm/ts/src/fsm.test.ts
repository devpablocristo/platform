import { describe, expect, it } from 'vitest';
import { Builder, InvalidTransitionError, Machine, TerminalStateError } from './fsm';

describe('Machine (typed)', () => {
  const m = new Machine([
    { from: 'draft', event: 'submit', to: 'pending' },
    { from: 'pending', event: 'approve', to: 'approved' },
    { from: 'pending', event: 'reject', to: 'rejected' },
  ]);

  it('transitions on valid event', () => {
    expect(m.transition('draft', 'submit')).toBe('pending');
    expect(m.transition('pending', 'approve')).toBe('approved');
  });

  it('throws on invalid transition', () => {
    expect(() => m.transition('draft', 'approve')).toThrow(InvalidTransitionError);
  });

  it('canTransition returns boolean', () => {
    expect(m.canTransition('draft', 'submit')).toBe(true);
    expect(m.canTransition('draft', 'approve')).toBe(false);
  });
});

describe('StringMachine (builder)', () => {
  const sm = new Builder()
    .terminal('invoiced', 'cancelled')
    .freeTransitionsAmong('received', 'diagnosing', 'in_progress', 'quality_check', 'on_hold')
    .allowFromStatesTo('ready_for_pickup', 'in_progress', 'quality_check')
    .allow('ready_for_pickup', 'delivered')
    .allowAnyTo('cancelled')
    .build();

  it('allows free transitions within group', () => {
    expect(sm.canTransition('received', 'diagnosing')).toBe(true);
    expect(sm.canTransition('in_progress', 'on_hold')).toBe(true);
    expect(sm.canTransition('quality_check', 'received')).toBe(true);
  });

  it('allows explicit edges', () => {
    expect(sm.canTransition('in_progress', 'ready_for_pickup')).toBe(true);
    expect(sm.canTransition('ready_for_pickup', 'delivered')).toBe(true);
  });

  it('allows self-transitions', () => {
    expect(sm.canTransition('received', 'received')).toBe(true);
    expect(sm.canTransition('invoiced', 'invoiced')).toBe(true);
  });

  it('blocks transitions from terminal states', () => {
    expect(sm.canTransition('invoiced', 'received')).toBe(false);
    expect(sm.canTransition('cancelled', 'received')).toBe(false);
  });

  it('allows anyTo transitions from non-terminal', () => {
    expect(sm.canTransition('received', 'cancelled')).toBe(true);
    expect(sm.canTransition('ready_for_pickup', 'cancelled')).toBe(true);
  });

  it('validate throws TerminalStateError', () => {
    expect(() => sm.validate('invoiced', 'received')).toThrow(TerminalStateError);
  });

  it('validate throws InvalidTransitionError for disallowed', () => {
    expect(() => sm.validate('ready_for_pickup', 'diagnosing')).toThrow(InvalidTransitionError);
  });

  it('validate passes for valid transitions', () => {
    expect(() => sm.validate('received', 'diagnosing')).not.toThrow();
    expect(() => sm.validate('in_progress', 'ready_for_pickup')).not.toThrow();
  });

  it('isTerminal', () => {
    expect(sm.isTerminal('invoiced')).toBe(true);
    expect(sm.isTerminal('cancelled')).toBe(true);
    expect(sm.isTerminal('received')).toBe(false);
  });
});
