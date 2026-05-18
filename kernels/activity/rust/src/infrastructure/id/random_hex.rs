use uuid::Uuid;

use crate::application::ports::IdGenerator;

#[derive(Debug, Default, Clone, Copy)]
pub struct RandomHexIdGenerator;

impl IdGenerator for RandomHexIdGenerator {
    fn new_id(&self) -> String {
        Uuid::new_v4().simple().to_string()
    }
}
