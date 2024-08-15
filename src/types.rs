use std::fmt::{self, Display, Formatter};

use serde::{Deserialize, Serialize};

#[derive(Debug, Copy, PartialEq, Clone, Eq, Hash, Serialize, Deserialize)]
pub struct ExecutionId(pub u32);

impl Display for ExecutionId {
    fn fmt(&self, f: &mut Formatter) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}
