use std::collections::HashMap;
use std::sync::{Arc, RwLock};

use crate::types::ExecutionId;
use crate::store::{Record, Store};

#[derive(Debug)]
struct MemoryStoreData {
    records: HashMap<ExecutionId, Record>,
    latest_id: Option<ExecutionId>,
}

#[derive(Clone, Debug)]
pub struct MemoryStore {
    data: Arc<RwLock<MemoryStoreData>>,
}

impl MemoryStore {
    pub fn new() -> Self {
        Self {
            data: Arc::new(RwLock::new(MemoryStoreData {
                records: HashMap::new(),
                latest_id: None,
            })),
        }
    }
}

impl Store for MemoryStore {
    fn add_record(&mut self, record: Record) {
        if let Ok(mut data) = self.data.write() {
            data.latest_id = Some(record.id);
            data.records.insert(record.id, record);
        }
    }

    fn get_record(&self, id: ExecutionId) -> Option<Record> {
        if let Ok(data) = self.data.read() {
            data.records.get(&id).cloned()
        } else {
            None
        }
    }

    fn get_latest_id(&self) -> Option<ExecutionId> {
        if let Ok(data) = self.data.read() {
            data.latest_id
        } else {
            None
        }
    }
}
