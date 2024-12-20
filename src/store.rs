pub mod memory;
pub mod sqlite;

use color_eyre::eyre::Result;
use std::{
    collections::HashMap,
    sync::{Arc, RwLock},
};

use chrono::{DateTime, Local};

use crate::types::ExecutionId;

pub trait Store: Clone + Send + Sync + 'static {
    fn add_record(&mut self, record: Record) -> Result<()>;
    fn get_record(&self, id: ExecutionId) -> Result<Option<Record>>;
    fn get_latest_id(&self) -> Result<Option<ExecutionId>>;
    fn get_records(&self) -> Result<Vec<Record>>;
    fn get_runtime_config(&self) -> Result<Option<RuntimeConfig>>;
    fn set_runtime_config(&mut self, config: RuntimeConfig) -> Result<()>;
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

#[derive(Debug, Clone, Default)]
pub struct RuntimeConfig {
    pub interval: u64,
    pub command: String,
}
