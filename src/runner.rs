use std::{ops::Sub, sync::Arc};

use dissimilar::{diff, Chunk};
use tokio::{
    process::Command,
    sync::{mpsc, watch, Mutex},
};

use crate::{
    action::Action,
    config::{Config, RuntimeConfig},
    exec::exec,
    store::{Record, Store},
    types::ExecutionId,
};

pub async fn run_executor(
    actions: mpsc::UnboundedSender<Action>,
    mut store: Store,
    runtime_config: RuntimeConfig,
    shell: Option<(String, Vec<String>)>,
    is_suspend: Arc<Mutex<bool>>,
) {
    let mut counter = 0;
    loop {
        if *is_suspend.lock().await {
            tokio::time::sleep(std::time::Duration::from_secs(1)).await;
            continue;
        }

        let id = ExecutionId(counter);
        let start_time = chrono::Local::now();
        if let Err(e) = actions.send(Action::StartExecution(id, start_time)) {
            eprintln!("Failed to send start: {:?}", e);
        }

        let result = exec(runtime_config.command.clone(), shell.clone()).await;
        if result.is_err() {
            eprintln!("Failed to execute command");
            tokio::time::sleep(runtime_config.interval.to_std().unwrap()).await;
        }

        let (stdout, stderr, status) = result.unwrap();
        let exit_code = status;
        let utf8_stdout = String::from_utf8_lossy(&stdout).to_string();
        let utf8_stderr = String::from_utf8_lossy(&stderr).to_string();
        let end_time = chrono::Local::now();

        let latest_id = store.get_latest_id();
        let diff = if let Some(latest_id) = latest_id {
            if let Some(record) = store.get_record(latest_id) {
                let old_stdout = String::from_utf8_lossy(&record.stdout).to_string();
                Some(count_diff(&old_stdout, &utf8_stdout))
            } else {
                None
            }
        } else {
            None
        };

        if let Some((diff_add, diff_delete)) = diff {
            if diff_add != 0 || diff_delete != 0 {
                if let Err(e) = actions.send(Action::DiffDetected) {
                    eprintln!("Failed to send diff detected: {:?}", e);
                }
            }
        }

        let record = Record {
            id,
            start_time,
            stdout,
            stderr,
            end_time,
            exit_code,
            diff,
            previous_id: latest_id,
        };
        store.add_record(record);

        if let Err(e) = actions.send(Action::FinishExecution(id, start_time, diff, exit_code)) {
            eprintln!("Failed to send result: {:?}", e);
        }

        counter += 1;

        tokio::time::sleep(runtime_config.interval.to_std().unwrap()).await;
    }
}

pub async fn run_executor_precise(
    actions: mpsc::UnboundedSender<Action>,
    mut store: Store,
    runtime_config: RuntimeConfig,
    shell: Option<(String, Vec<String>)>,
    is_suspend: Arc<Mutex<bool>>,
) {
    let mut counter = 0;
    loop {
        let start_time = chrono::Local::now();
        if *is_suspend.lock().await {
            tokio::time::sleep(std::time::Duration::from_secs(1)).await;
            continue;
        }

        let id = ExecutionId(counter);
        if let Err(e) = actions.send(Action::StartExecution(id, start_time)) {
            eprintln!("Failed to send start: {:?}", e);
        }

        let result = exec(runtime_config.command.clone(), shell.clone()).await;
        if result.is_err() {
            eprintln!("Failed to execute command");
            tokio::time::sleep(runtime_config.interval.to_std().unwrap()).await;
        }

        let (stdout, stderr, status) = result.unwrap();
        let exit_code = status;
        let utf8_stdout = String::from_utf8_lossy(&stdout).to_string();
        let utf8_stderr = String::from_utf8_lossy(&stderr).to_string();
        let end_time = chrono::Local::now();

        let latest_id = store.get_latest_id();
        let diff = if let Some(latest_id) = latest_id {
            if let Some(record) = store.get_record(latest_id) {
                let old_stdout = String::from_utf8_lossy(&record.stdout).to_string();
                Some(count_diff(&old_stdout, &utf8_stdout))
            } else {
                None
            }
        } else {
            None
        };

        if let Some((diff_add, diff_delete)) = diff {
            if diff_add != 0 || diff_delete != 0 {
                if let Err(e) = actions.send(Action::DiffDetected) {
                    eprintln!("Failed to send diff detected: {:?}", e);
                }
            }
        }

        let record = Record {
            id,
            start_time,
            stdout,
            stderr,
            end_time,
            exit_code,
            diff,
            previous_id: latest_id,
        };
        store.add_record(record);

        if let Err(e) = actions.send(Action::FinishExecution(id, start_time, diff, exit_code)) {
            eprintln!("Failed to send result: {:?}", e);
        }

        counter += 1;

        let elapased = chrono::Local::now().signed_duration_since(start_time);
        let sleep_time = runtime_config.interval.sub(elapased);
        if let Ok(sleep_time) = sleep_time.to_std() {
            tokio::time::sleep(sleep_time).await;
        }
    }
}

fn count_diff(old: &str, current: &str) -> (u32, u32) {
    diff(old, current)
        .iter()
        .map(|c| match c {
            Chunk::Delete(s) => (0, s.chars().count() as u32),
            Chunk::Insert(s) => (s.chars().count() as u32, 0),
            _ => (0, 0),
        })
        .reduce(|t1, t2| (t1.0 + t2.0, t1.1 + t2.1))
        .unwrap_or_default()
}

#[cfg(test)]
mod test {
    use super::count_diff;

    #[test]
    fn test_count_diff() {
        let current = "hello world!";
        let old = "hello world";

        let result = count_diff(old, current);

        assert_eq!(result, (1, 0))
    }

    #[test]
    fn test_count_delete_diff() {
        let current = "hello world";
        let old = "hello oorld!";

        let result = count_diff(old, current);

        assert_eq!(result, (1, 2))
    }
}
