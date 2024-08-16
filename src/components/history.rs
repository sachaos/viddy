use std::{
    cell::RefCell,
    collections::{HashMap, VecDeque},
    rc::Rc,
    time::Duration,
};

use chrono::{DateTime, Local};
use color_eyre::{
    eyre::{Ok, OptionExt, Result},
    owo_colors::OwoColorize,
};
use crossterm::event::{KeyCode, KeyEvent};
use ratatui::{prelude::*, widgets::*};
use serde::{Deserialize, Serialize};
use symbols::scrollbar;
use tokio::sync::mpsc::UnboundedSender;
use tui_widget_list::{List, ListState};

use super::{Component, Frame};
use crate::{
    action::Action,
    config::{Config, KeyBindings, RuntimeConfig},
    mode::Mode,
    types::ExecutionId,
    widget::history_item::HisotryItem,
};

pub struct History {
    latest_id: Option<ExecutionId>,
    command_tx: Option<UnboundedSender<Action>>,
    config: Config,
    items: VecDeque<Rc<RefCell<HisotryItem>>>,
    index: HashMap<ExecutionId, Rc<RefCell<HisotryItem>>>,
    state: ListState,
    mode: Mode,
    runtime_config: RuntimeConfig,
    timemachine_mode: bool,
    y_state: ScrollbarState,
}

impl History {
    pub fn new(runtime_config: RuntimeConfig) -> Self {
        let state = ListState::default();
        let index = HashMap::new();
        Self {
            latest_id: None,
            command_tx: None,
            config: Config::new().unwrap(),
            items: VecDeque::new(),
            state,
            mode: Default::default(),
            index,
            runtime_config,
            timemachine_mode: false,
            y_state: ScrollbarState::default(),
        }
    }

    fn update_latest_history_count(&self) -> Result<()> {
        if let Some(latest_id) = self.latest_id {
            if let Some(record) = self.index.get(&latest_id) {
                record.borrow_mut().update_same_count();
            }
        }

        Ok(())
    }

    fn insert_history(&mut self, id: ExecutionId, start_time: DateTime<Local>) -> Result<()> {
        let item = Rc::new(RefCell::new(HisotryItem::new(
            id,
            start_time,
            self.runtime_config.interval,
            self.config.get_style("timemachine_selector"),
            self.config.get_style("secondary_text"),
        )));
        self.index.insert(id, Rc::clone(&item));
        self.items.push_front(item);
        self.latest_id = Some(id);
        if self.timemachine_mode {
            self.select(self.state.selected.map(|s| s + 1))?;
        }

        Ok(())
    }

    fn update_history_result(
        &mut self,
        id: ExecutionId,
        diff: Option<(u32, u32)>,
        exit_code: i32,
    ) -> Result<()> {
        if let Some(item) = self.index.get(&id) {
            item.borrow_mut().update_diff(diff, exit_code);
            if self.timemachine_mode && self.state.selected.is_none() {
                self.select_latest()?;
            }
        }

        Ok(())
    }

    fn set_timemachine_mode(&mut self, timemachine_mode: bool) -> Result<()> {
        self.timemachine_mode = timemachine_mode;
        if self.timemachine_mode {
            self.select_latest()?;
        }
        Ok(())
    }

    fn select_latest(&mut self) -> Result<()> {
        for (i, item) in self.items.iter().enumerate() {
            let item = item.borrow();
            if !item.is_running {
                self.state.select(Some(i));
                return Ok(());
            }
        }

        Ok(())
    }

    fn select(&mut self, index: Option<usize>) -> Result<()> {
        if let Some(index) = index {
            if let Some(history_item) = self.items.get(index) {
                let history_item = history_item.borrow();
                if !history_item.is_running {
                    self.state.select(Some(index));

                    if let Some(tx) = &self.command_tx {
                        tx.send(Action::ShowExecution(history_item.id, history_item.id))?;
                    }
                }
            }
        }
        Ok(())
    }

    fn go_to_past(&mut self) -> Result<()> {
        self.select_saturating_add(1)
    }

    fn go_to_more_past(&mut self) -> Result<()> {
        self.select_saturating_add(10)
    }

    fn go_to_future(&mut self) -> Result<()> {
        self.select_saturating_sub(1)
    }

    fn go_to_more_future(&mut self) -> Result<()> {
        self.select_saturating_sub(10)
    }

    fn select_saturating_add(&mut self, n: usize) -> Result<()> {
        if !self.timemachine_mode {
            return Ok(());
        }

        let selected = self
            .state
            .selected
            .map(|s| s.saturating_add(n).min(self.items.len() - 1));
        if selected.is_none() {
            return Ok(());
        }

        self.select(selected)
    }

    fn select_saturating_sub(&mut self, n: usize) -> Result<()> {
        if !self.timemachine_mode {
            return Ok(());
        }

        if self.state.selected.is_none() {
            return Ok(());
        }

        self.select(self.state.selected.map(|s| s.saturating_sub(n)))
    }

    fn go_to_oldest(&mut self) -> Result<()> {
        if !self.timemachine_mode {
            return Ok(());
        }

        self.select(self.items.len().checked_sub(1))
    }

    fn go_to_current(&mut self) -> Result<()> {
        if !self.timemachine_mode {
            return Ok(());
        }

        self.select_latest()
    }
}

impl Component for History {
    fn register_action_handler(&mut self, tx: UnboundedSender<Action>) -> Result<()> {
        self.command_tx = Some(tx);
        Ok(())
    }

    fn register_config_handler(&mut self, config: Config) -> Result<()> {
        self.config = config;
        Ok(())
    }

    fn update(&mut self, action: Action) -> Result<Option<Action>> {
        match action {
            Action::InsertHistory(id, start_time) => self.insert_history(id, start_time)?,
            Action::UpdateHistoryResult(id, diff, exit_code) => {
                self.update_history_result(id, diff, exit_code)?
            }
            Action::UpdateLatestHistoryCount => self.update_latest_history_count()?,
            Action::GoToPast => self.go_to_past()?,
            Action::GoToFuture => self.go_to_future()?,
            Action::SetTimemachineMode(timemachine_mode) => {
                self.set_timemachine_mode(timemachine_mode)?
            }
            Action::GoToMoreFuture => self.go_to_more_future()?,
            Action::GoToMorePast => self.go_to_more_past()?,
            Action::GoToOldest => self.go_to_oldest()?,
            Action::GoToCurrent => self.go_to_current()?,
            _ => {}
        }

        Ok(None)
    }

    fn draw(&mut self, f: &mut Frame<'_>, area: Rect) -> Result<()> {
        let block = Block::default()
            .title("History")
            .borders(Borders::ALL)
            .border_style(self.config.get_style("border"))
            .title_style(self.config.get_style("title"));
        let items = self
            .items
            .iter()
            .map(|i| i.borrow().clone())
            .collect::<Vec<_>>();
        let list = List::new(items).block(block);

        f.render_stateful_widget(list, area, &mut self.state);

        Ok(())
    }
}
