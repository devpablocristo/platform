/**
 * Máquinas de estados finitas — paridad con core/concurrency/go/fsm.
 *
 * Dos variantes:
 * - `Machine<S, E>`: genérica tipada con reglas From+Event→To
 * - `StringMachine`: builder fluido con terminales, grupos libres y aristas explícitas
 */

// --- Machine genérica tipada ---

export type Rule<S, E> = {
  from: S;
  event: E;
  to: S;
};

export class Machine<S, E> {
  private transitions: Map<string, S>;

  constructor(rules: Rule<S, E>[]) {
    this.transitions = new Map();
    for (const r of rules) {
      this.transitions.set(`${r.from}|${r.event}`, r.to);
    }
  }

  transition(from: S, event: E): S {
    const to = this.transitions.get(`${from}|${event}`);
    if (to === undefined) {
      throw new InvalidTransitionError(`${from}`, `${event}`);
    }
    return to;
  }

  canTransition(from: S, event: E): boolean {
    return this.transitions.has(`${from}|${event}`);
  }
}

// --- StringMachine con builder ---

export class StringMachine {
  private terminals: Set<string>;
  private freeGroup: Set<string>;
  private explicit: Map<string, Set<string>>;
  private allowAnyNonTerminalTo: Set<string>;

  constructor(
    terminals: Set<string>,
    freeGroup: Set<string>,
    explicit: Map<string, Set<string>>,
    allowAnyNonTerminalTo: Set<string>,
  ) {
    this.terminals = terminals;
    this.freeGroup = freeGroup;
    this.explicit = explicit;
    this.allowAnyNonTerminalTo = allowAnyNonTerminalTo;
  }

  canTransition(from: string, to: string): boolean {
    if (from === to) return true;
    if (this.terminals.has(from)) return false;
    if (this.freeGroup.has(from) && this.freeGroup.has(to)) return true;
    if (this.explicit.get(from)?.has(to)) return true;
    if (this.allowAnyNonTerminalTo.has(to)) return true;
    return false;
  }

  validate(from: string, to: string): void {
    if (from === to) return;
    if (this.terminals.has(from)) {
      throw new TerminalStateError(from);
    }
    if (!this.canTransition(from, to)) {
      throw new InvalidTransitionError(from, to);
    }
  }

  isTerminal(state: string): boolean {
    return this.terminals.has(state);
  }
}

export class Builder {
  private _terminals = new Set<string>();
  private _freeGroup = new Set<string>();
  private _explicit = new Map<string, Set<string>>();
  private _allowAnyNonTerminalTo = new Set<string>();

  terminal(...states: string[]): this {
    for (const s of states) {
      if (s) this._terminals.add(s);
    }
    return this;
  }

  freeTransitionsAmong(...states: string[]): this {
    for (const s of states) {
      if (s) this._freeGroup.add(s);
    }
    return this;
  }

  allow(from: string, to: string): this {
    return this.allowFromStatesTo(to, from);
  }

  allowFromStatesTo(to: string, ...froms: string[]): this {
    if (!to) return this;
    for (const from of froms) {
      if (!from) continue;
      let targets = this._explicit.get(from);
      if (!targets) {
        targets = new Set();
        this._explicit.set(from, targets);
      }
      targets.add(to);
    }
    return this;
  }

  allowAnyTo(to: string): this {
    if (to) this._allowAnyNonTerminalTo.add(to);
    return this;
  }

  build(): StringMachine {
    return new StringMachine(
      new Set(this._terminals),
      new Set(this._freeGroup),
      new Map(Array.from(this._explicit.entries()).map(([k, v]) => [k, new Set(v)])),
      new Set(this._allowAnyNonTerminalTo),
    );
  }
}

// --- Errores ---

export class InvalidTransitionError extends Error {
  constructor(from: string, to: string) {
    super(`fsm: invalid transition (${from} -> ${to})`);
    this.name = 'InvalidTransitionError';
  }
}

export class TerminalStateError extends Error {
  constructor(state: string) {
    super(`fsm: cannot leave terminal state (${state})`);
    this.name = 'TerminalStateError';
  }
}
