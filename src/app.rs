use core::time;
use std::sync::Arc;

use anstyle::{Color, RgbColor, Style};
use chrono::Duration;
use color_eyre::{eyre::Result, owo_colors::OwoColorize};
use crossterm::event::{Event, KeyEvent};
use ratatui::{prelude::Rect, widgets::Block};
use serde::{Deserialize, Serialize};
use tokio::{
    runtime,
    sync::{mpsc, Mutex},
};
use tracing_subscriber::field::debug;

use crate::{
    action::{self, Action, DiffMode},
    cli::Cli,
    components::{fps::FpsCounter, home::Home, Component},
    config::{Config, RuntimeConfig},
    diff::{diff_and_mark, diff_and_mark_delete},
    mode::Mode,
    old_config::OldConfig,
    runner::{run_executor, run_executor_precise},
    search::search_and_mark,
    store::{self, RuntimeConfig as StoreRuntimeConfig, Store},
    termtext, tui,
    types::ExecutionId,
};

pub struct App<S: Store> {
    pub config: Config,
    pub runtime_config: RuntimeConfig,
    pub tick_rate: f64,
    pub frame_rate: f64,
    pub components: Vec<Box<dyn Component>>,
    pub should_quit: bool,
    pub should_suspend: bool,
    pub mode: Mode,
    pub last_tick_key_events: Vec<KeyEvent>,
    pub timemachine_mode: bool,
    pub search_query: Option<String>,
    is_precise: bool,
    diff_mode: Option<DiffMode>,
    is_suspend: Arc<Mutex<bool>>,
    is_bell: bool,
    is_fold: bool,
    is_no_title: bool,
    is_skip_empty_diffs: bool,
    showing_execution_id: Option<ExecutionId>,
    shell: Option<(String, Vec<String>)>,
    store: S,
    read_only: bool,
}

impl<S: Store> App<S> {
    pub fn new(cli: Cli, mut store: S, read_only: bool) -> Result<Self> {
        let runtime_config = if read_only {
            let store_runtime_config = store.get_runtime_config()?.unwrap_or_default();

            RuntimeConfig {
                interval: Duration::from_std(humantime::parse_duration(
                    &store_runtime_config.interval,
                )?)?,
                command: store_runtime_config
                    .command
                    .split(' ')
                    .map(|s| s.to_string())
                    .collect(),
            }
        } else {
            let runtime_config = RuntimeConfig {
                interval: cli.interval,
                command: cli.command.clone(),
            };

            let interval =
                humantime::format_duration(cli.interval.to_std().unwrap_or_default()).to_string();
            let command = cli.command.join(" ");
            store.set_runtime_config(StoreRuntimeConfig { interval, command })?;

            runtime_config
        };

        let diff_mode = match (cli.is_diff, cli.is_deletion_diff) {
            (true, false) => Some(DiffMode::Add),
            (false, true) => Some(DiffMode::Delete),
            _ => None,
        };
        let config = if let Ok(config) = OldConfig::new() {
            let mut c = Config::from(config);
            c.defaulting();
            c
        } else {
            Config::new()?
        };

        let default_exec = config.general.no_shell.unwrap_or_default();
        let default_shell = config
            .general
            .shell
            .clone()
            .unwrap_or_else(|| "sh".to_string());
        let default_shell_options = match config.general.shell_options {
            Some(ref shell_options) if !shell_options.is_empty() => {
                shell_options.split(' ').map(|s| s.to_string()).collect()
            }
            _ => vec!["-c".to_string()],
        };
        let is_exec = cli.is_exec || default_exec;
        let shell = if is_exec {
            None
        } else {
            Some((
                cli.shell.unwrap_or(default_shell),
                cli.shell_options.unwrap_or(default_shell_options),
            ))
        };

        let timemachine_mode = false;
        let home = Home::new(
            config.clone(),
            runtime_config.clone(),
            !cli.is_unfold,
            diff_mode,
            cli.is_bell,
            cli.is_no_title,
            read_only,
            timemachine_mode,
        );
        let mut components: Vec<Box<dyn Component>> = vec![Box::new(home)];
        if cli.is_debug {
            components.push(Box::new(FpsCounter::new()));
        }

        let default_skip_empty_diffs = config.general.skip_empty_diffs.unwrap_or_default();
        let is_skip_empty_diffs = cli.is_skip_empty_diffs || default_skip_empty_diffs;

        Ok(Self {
            store,
            tick_rate: 1.0,
            frame_rate: 20.0,
            components,
            should_quit: false,
            read_only,
            should_suspend: false,
            config,
            runtime_config,
            mode: Mode::All,
            last_tick_key_events: Vec::new(),
            timemachine_mode,
            search_query: None,
            is_precise: cli.is_precise,
            is_bell: cli.is_bell,
            is_fold: !cli.is_unfold,
            is_no_title: cli.is_no_title,
            is_suspend: Arc::new(Mutex::new(false)),
            is_skip_empty_diffs,
            showing_execution_id: None,
            diff_mode,
            shell,
        })
    }

    fn set_mode(&mut self, mode: Mode) {
        self.mode = mode;
    }

    pub async fn run(&mut self) -> Result<()> {
        let (action_tx, mut action_rx) = mpsc::unbounded_channel();

        let records = self.store.get_records()?;
        for r in records {
            action_tx.send(Action::StartExecution(r.id, r.start_time))?;
            action_tx.send(Action::FinishExecution(
                r.id,
                r.end_time,
                r.diff,
                r.exit_code,
            ))?;
        }
        if self.read_only {
            action_tx.send(Action::SetTimemachineMode(true))?;
        }

        let executor_handle = if self.read_only {
            tokio::spawn(async move {
                loop {
                    tokio::time::sleep(Duration::seconds(1).to_std().unwrap()).await;
                }
            })
        } else if self.is_precise {
            tokio::spawn(run_executor_precise(
                action_tx.clone(),
                self.store.clone(),
                self.runtime_config.clone(),
                self.shell.clone(),
                self.is_suspend.clone(),
            ))
        } else {
            tokio::spawn(run_executor(
                action_tx.clone(),
                self.store.clone(),
                self.runtime_config.clone(),
                self.shell.clone(),
                self.is_suspend.clone(),
            ))
        };

        let mut tui = tui::Tui::new()?
            .tick_rate(self.tick_rate)
            .frame_rate(self.frame_rate);
        // tui.mouse(true);
        tui.enter()?;

        for component in self.components.iter_mut() {
            component.register_action_handler(action_tx.clone())?;
        }

        for component in self.components.iter_mut() {
            component.register_config_handler(self.config.clone())?;
        }

        for component in self.components.iter_mut() {
            component.init(tui.size()?)?;
        }

        loop {
            if let Some(e) = tui.next().await {
                match e {
                    tui::Event::Quit => action_tx.send(Action::Quit)?,
                    tui::Event::Tick => action_tx.send(Action::Tick)?,
                    tui::Event::Render => action_tx.send(Action::Render)?,
                    tui::Event::Resize(x, y) => action_tx.send(Action::Resize(x, y))?,
                    tui::Event::Key(key) => {
                        if let Some(keymap) = self.config.keybindings.get(&self.mode) {
                            if let Some(action) = keymap.get(&vec![key]) {
                                log::info!("Got action: {action:?}");
                                action_tx.send(action.clone())?;
                            } else {
                                if self.mode == Mode::Search {
                                    action_tx.send(Action::KeyEventForPrompt(key))?;
                                    continue;
                                }

                                // If the key was not handled as a single key action,
                                // then consider it for multi-key combinations.
                                self.last_tick_key_events.push(key);

                                // Check for multi-key combinations
                                if let Some(action) = keymap.get(&self.last_tick_key_events) {
                                    log::info!("Got action: {action:?}");
                                    action_tx.send(action.clone())?;
                                }
                            }
                        };
                    }
                    _ => {}
                }
                for component in self.components.iter_mut() {
                    if let Some(action) = component.handle_events(Some(e.clone()))? {
                        action_tx.send(action)?;
                    }
                }
            }

            while let Ok(action) = action_rx.try_recv() {
                if action != Action::Tick
                    && action != Action::Render
                    && !matches!(action, Action::SetResult(_))
                {
                    log::debug!("{action:?}");
                }
                match action {
                    Action::Tick => {
                        self.last_tick_key_events.drain(..);
                    }
                    Action::Quit => self.should_quit = true,
                    Action::Suspend => self.should_suspend = true,
                    Action::Resume => self.should_suspend = false,
                    Action::Resize(w, h) => {
                        tui.resize(Rect::new(0, 0, w, h))?;
                        tui.draw(|f| {
                            for component in self.components.iter_mut() {
                                let r = component.draw(f, f.size());
                                if let Err(e) = r {
                                    action_tx
                                        .send(Action::Error(format!("Failed to draw: {:?}", e)))
                                        .unwrap();
                                }
                            }
                        })?;
                    }
                    Action::Render => {
                        tui.draw(|f| {
                            for component in self.components.iter_mut() {
                                let r = component.draw(f, f.size());
                                if let Err(e) = r {
                                    action_tx
                                        .send(Action::Error(format!("Failed to draw: {:?}", e)))
                                        .unwrap();
                                }
                            }
                        })?;
                    }
                    Action::FinishExecution(id, start_time, diff, exit_code) => {
                        if self.is_skip_empty_diffs {
                            if let Some((diff_add, diff_delete)) = diff {
                                if diff_add == 0 && diff_delete == 0 {
                                    action_tx.send(Action::UpdateLatestHistoryCount)?;
                                } else {
                                    action_tx.send(Action::InsertHistory(id, start_time))?;
                                    action_tx
                                        .send(Action::UpdateHistoryResult(id, diff, exit_code))?;
                                }
                            } else {
                                action_tx.send(Action::InsertHistory(id, start_time))?;
                                action_tx.send(Action::UpdateHistoryResult(id, diff, exit_code))?;
                            }
                        } else {
                            action_tx.send(Action::UpdateHistoryResult(id, diff, exit_code))?;
                        }

                        if !self.timemachine_mode && !self.read_only {
                            action_tx.send(Action::ShowExecution(id, id))?;
                        }
                    }
                    Action::StartExecution(id, start_time) => {
                        if !self.is_skip_empty_diffs {
                            action_tx.send(Action::InsertHistory(id, start_time))?;
                        }
                    }
                    Action::ShowExecution(id, end_id) => {
                        let style =
                            termtext::convert_to_anstyle(self.config.get_style("background"));
                        let record = self.store.get_record(id)?;
                        let mut string = "".to_string();
                        if let Some(record) = record {
                            action_tx.send(Action::SetClock(record.start_time))?;
                            let mut result =
                                termtext::Converter::new(style).convert(&record.stdout);
                            if record.stdout.is_empty() {
                                result = termtext::Converter::new(style).convert(&record.stderr);
                                result.mark_text(
                                    0,
                                    result.len(),
                                    Style::new()
                                        .fg_color(Some(Color::Ansi(anstyle::AnsiColor::Red))),
                                );
                            } else {
                                string = result.plain_text();
                                if let Some(diff_mode) = self.diff_mode {
                                    if let Some(previous_id) = record.previous_id {
                                        let previous_record = self.store.get_record(previous_id)?;
                                        if let Some(previous_record) = previous_record {
                                            let previous_result = termtext::Converter::new(style)
                                                .convert(&previous_record.stdout);
                                            let previous_string = previous_result.plain_text();
                                            if diff_mode == DiffMode::Add {
                                                diff_and_mark(
                                                    &string,
                                                    &previous_string,
                                                    &mut result,
                                                );
                                            } else if diff_mode == DiffMode::Delete {
                                                result = previous_result;
                                                diff_and_mark_delete(
                                                    &string,
                                                    &previous_string,
                                                    &mut result,
                                                );
                                                string = previous_string; // Use previous string for search
                                            }
                                        }
                                    }
                                }
                            }

                            if let Some(ref search_query) = self.search_query {
                                search_and_mark(
                                    &string,
                                    &mut result,
                                    search_query,
                                    termtext::convert_to_anstyle(
                                        self.config.get_style("search_highlight"),
                                    ),
                                );
                            }
                            action_tx.send(Action::SetResult(Some(result)))?;
                            self.showing_execution_id = Some(id);
                        }
                    }
                    Action::SwitchTimemachineMode => {
                        action_tx.send(Action::SetTimemachineMode(!self.timemachine_mode))?;
                    }
                    Action::SetTimemachineMode(timemachine_mode) => {
                        self.timemachine_mode = timemachine_mode;
                        if let Some(latest_id) = self.store.get_latest_id()? {
                            log::debug!("Latest ID: {latest_id}");
                            action_tx.send(Action::ShowExecution(latest_id, latest_id))?;
                        }
                    }
                    Action::ExecuteSearch => {
                        action_tx.send(Action::SetMode(Mode::All))?;
                    }
                    Action::EnterSearchMode => {
                        action_tx.send(Action::SetMode(Mode::Search))?;
                    }
                    Action::ExitSearchMode => {
                        action_tx.send(Action::SetMode(Mode::All))?;
                    }
                    Action::SwitchFold => {
                        action_tx.send(Action::SetFold(!self.is_fold))?;
                    }
                    Action::SetFold(is_fold) => {
                        self.is_fold = is_fold;
                    }
                    Action::SetSearchQuery(ref query) => {
                        self.search_query = Some(query.clone());
                        if let Some(id) = self.showing_execution_id {
                            action_tx.send(Action::ShowExecution(id, id))?;
                        }
                    }
                    Action::SwitchDiff => match self.diff_mode {
                        None => action_tx.send(Action::SetDiff(Some(DiffMode::Add)))?,
                        Some(DiffMode::Add) => action_tx.send(Action::SetDiff(None))?,
                        Some(DiffMode::Delete) => {
                            action_tx.send(Action::SetDiff(Some(DiffMode::Add)))?
                        }
                    },
                    Action::SwitchDeletionDiff => match self.diff_mode {
                        None => action_tx.send(Action::SetDiff(Some(DiffMode::Delete)))?,
                        Some(DiffMode::Delete) => action_tx.send(Action::SetDiff(None))?,
                        Some(DiffMode::Add) => {
                            action_tx.send(Action::SetDiff(Some(DiffMode::Delete)))?
                        }
                    },
                    Action::SetDiff(diff_mode) => {
                        self.diff_mode = diff_mode;
                        if let Some(id) = self.showing_execution_id {
                            action_tx.send(Action::ShowExecution(id, id))?;
                        }
                    }
                    Action::SwitchBell => {
                        action_tx.send(Action::SetBell(!self.is_bell))?;
                    }
                    Action::SetBell(is_bell) => {
                        self.is_bell = is_bell;
                    }
                    Action::SwitchSuspend => {
                        let is_suspend = self.is_suspend.lock().await;
                        action_tx.send(Action::SetSuspend(!*is_suspend))?;
                    }
                    Action::SetSuspend(new_is_suspend) => {
                        let mut is_suspend = self.is_suspend.lock().await;
                        *is_suspend = new_is_suspend;
                    }
                    Action::DiffDetected => {
                        if self.is_bell {
                            print!("\x07");
                        }
                    }
                    Action::SwitchNoTitle => {
                        action_tx.send(Action::SetNoTitle(!self.is_no_title))?;
                    }
                    Action::SetNoTitle(is_no_title) => {
                        self.is_no_title = is_no_title;
                    }
                    Action::ShowHelp => {
                        action_tx.send(Action::SetMode(Mode::Help))?;
                    }
                    Action::ExitHelp => {
                        action_tx.send(Action::SetMode(Mode::All))?;
                    }
                    Action::SetMode(mode) => {
                        self.set_mode(mode);
                    }
                    _ => {}
                }
                for component in self.components.iter_mut() {
                    if let Some(action) = component.update(action.clone())? {
                        action_tx.send(action)?
                    };
                }
            }

            if executor_handle.is_finished() {
                tui.stop()?;
                break;
            }

            if self.should_suspend {
                tui.suspend()?;
                action_tx.send(Action::Resume)?;
                tui = tui::Tui::new()?
                    .tick_rate(self.tick_rate)
                    .frame_rate(self.frame_rate);
                // tui.mouse(true);
                tui.enter()?;
            } else if self.should_quit {
                tui.stop()?;
                break;
            }
        }
        tui.exit()?;

        if !executor_handle.is_finished() {
            log::debug!("Waiting for executor to finish");
            executor_handle.abort();
            return Ok(());
        }

        executor_handle.await?
    }
}
