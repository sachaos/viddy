pub mod memory;

use std::{
    collections::HashMap,
    sync::{Arc, RwLock},
};

use chrono::{DateTime, Local};

use crate::types::ExecutionId;

pub trait Store {
    fn add_record(&mut self, record: Record);
    fn get_record(&self, id: ExecutionId) -> Option<Record>;
    fn get_latest_id(&self) -> Option<ExecutionId>;
}

#[derive(Debug, Clone)]
pub struct Record {
    pub id: ExecutionId,
    pub start_time: DateTime<Local>,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
    pub end_time: DateTime<Local>,
    pub exit_code: i32,
    pub diff: Option<(u32, u32)>,
    pub previous_id: Option<ExecutionId>,
}
