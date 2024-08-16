use std::fmt::{self, Display, Formatter};

use rusqlite::{
    types::{FromSql, ToSql, ToSqlOutput},
    Result,
};
use serde::{Deserialize, Serialize};

#[derive(Debug, Copy, PartialEq, Clone, Eq, Hash, Serialize, Deserialize)]
pub struct ExecutionId(pub u32);

impl Display for ExecutionId {
    fn fmt(&self, f: &mut Formatter) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl ToSql for ExecutionId {
    fn to_sql(&self) -> Result<ToSqlOutput<'_>> {
        Ok(ToSqlOutput::Owned(rusqlite::types::Value::Integer(
            i64::from(self.0),
        )))
    }
}

impl FromSql for ExecutionId {
    fn column_result(value: rusqlite::types::ValueRef<'_>) -> rusqlite::types::FromSqlResult<Self> {
        value.as_i64().map(|v| ExecutionId(v as u32))
    }
}
