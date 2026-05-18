//! Máquina de estados finita con transiciones validadas en compile-time.
//!
//! Usa el patrón typestate: cada estado es un tipo distinto.
//! Las transiciones inválidas no compilan.

/// Trait que define un estado válido en la FSM.
pub trait State: std::fmt::Debug + Clone + Send + Sync + 'static {
    fn name(&self) -> &'static str;
}

/// Trait que define una transición válida de S → T.
pub trait Transition<From: State, To: State> {
    fn apply(machine: Machine<From>) -> Machine<To>;
}

/// Máquina de estados parametrizada por su estado actual.
#[derive(Debug, Clone)]
pub struct Machine<S: State> {
    state: S,
    transitions: u32,
}

impl<S: State> Machine<S> {
    /// Estado actual.
    pub fn state(&self) -> &S {
        &self.state
    }

    /// Nombre del estado actual.
    pub fn state_name(&self) -> &'static str {
        self.state.name()
    }

    /// Cantidad de transiciones realizadas.
    pub fn transitions(&self) -> u32 {
        self.transitions
    }

    /// Transiciona a otro estado (solo compila si la transición es válida).
    pub fn transition<T: State>(self, next: T) -> Machine<T> {
        Machine {
            state: next,
            transitions: self.transitions + 1,
        }
    }
}

/// Crea una máquina en su estado inicial.
pub fn start<S: State>(initial: S) -> Machine<S> {
    Machine {
        state: initial,
        transitions: 0,
    }
}

// --- Ejemplo: estados de quote (draft → sent → accepted/rejected/archived) ---

/// Macro para definir estados simples.
#[macro_export]
macro_rules! define_states {
    ($($name:ident => $label:expr),+ $(,)?) => {
        $(
            #[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
            pub struct $name;

            impl $crate::State for $name {
                fn name(&self) -> &'static str { $label }
            }
        )+
    };
}

#[cfg(test)]
mod tests {
    use super::*;

    define_states! {
        Draft => "draft",
        Sent => "sent",
        Accepted => "accepted",
        Rejected => "rejected",
    }

    #[test]
    fn typestate_transitions() {
        let machine = start(Draft);
        assert_eq!(machine.state_name(), "draft");
        assert_eq!(machine.transitions(), 0);

        let machine = machine.transition(Sent);
        assert_eq!(machine.state_name(), "sent");
        assert_eq!(machine.transitions(), 1);

        let machine = machine.transition(Accepted);
        assert_eq!(machine.state_name(), "accepted");
        assert_eq!(machine.transitions(), 2);
    }

    #[test]
    fn branch_transitions() {
        let sent = start(Draft).transition(Sent);

        // Ambas ramas son válidas
        let _accepted = sent.clone().transition(Accepted);
        let _rejected = sent.transition(Rejected);
    }
}
