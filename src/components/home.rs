use std::{collections::HashMap, time::Duration};

use color_eyre::{eyre::Result, owo_colors::OwoColorize};
use crossterm::event::{KeyCode, KeyEvent};
use ratatui::{prelude::*, style::Style, widgets::*};
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc::UnboundedSender;

use super::{
    clock::Clock, command::Command, execution_result::ExecutionResult, help::Help,
    history::History, interval::Interval, prompt::Prompt, status::Status, Component, Frame,
};
use crate::{
    action::{Action, DiffMode},
    config::{Config, KeyBindings, RuntimeConfig},
    mode::Mode,
};

pub struct Home {
    command_tx: Option<UnboundedSender<Action>>,
    config: Config,
    is_no_title: bool,

    mode: Mode,
    command_component: Command,
    interval_component: Interval,
    clock_component: Clock,
    execution_result_component: ExecutionResult,
    history_component: History,
    prompt_component: Prompt,
    status_component: Status,
    help_component: Help,
    timemachine_mode: bool,
}

impl Home {
    pub fn new(
        config: Config,
        runtime_config: RuntimeConfig,
        is_fold: bool,
        diff_mode: Option<DiffMode>,
        is_bell: bool,
        is_no_title: bool,
    ) -> Self {
        Self {
            command_tx: None,
            config: config.clone(),
            is_no_title,
            mode: Default::default(),
            command_component: Command::new(runtime_config.clone()),
            interval_component: Interval::new(runtime_config.clone()),
            clock_component: Clock::new(),
            execution_result_component: ExecutionResult::new(is_fold),
            history_component: History::new(runtime_config.clone()),
            prompt_component: Prompt::new(),
            status_component: Status::new(is_fold, diff_mode, is_bell),
            help_component: Help::new(config),
            timemachine_mode: false,
        }
    }

    fn set_mode(&mut self, mode: Mode) {
        self.mode = mode;
    }

    fn set_timemachine_mode(&mut self, timemachine_mode: bool) {
        self.timemachine_mode = timemachine_mode;
    }
}

impl Component for Home {
    fn register_action_handler(&mut self, tx: UnboundedSender<Action>) -> Result<()> {
        self.command_tx = Some(tx.clone());

        self.command_component.register_action_handler(tx.clone())?;
        self.interval_component
            .register_action_handler(tx.clone())?;
        self.clock_component.register_action_handler(tx.clone())?;
        self.execution_result_component
            .register_action_handler(tx.clone())?;
        self.history_component.register_action_handler(tx.clone())?;
        self.prompt_component.register_action_handler(tx.clone())?;
        self.status_component.register_action_handler(tx.clone())?;
        self.help_component.register_action_handler(tx.clone())?;

        Ok(())
    }

    fn register_config_handler(&mut self, config: Config) -> Result<()> {
        self.config = config.clone();

        self.command_component
            .register_config_handler(config.clone())?;
        self.interval_component
            .register_config_handler(config.clone())?;
        self.clock_component
            .register_config_handler(config.clone())?;
        self.execution_result_component
            .register_config_handler(config.clone())?;
        self.history_component
            .register_config_handler(config.clone())?;
        self.prompt_component
            .register_config_handler(config.clone())?;
        self.status_component
            .register_config_handler(config.clone())?;
        self.help_component
            .register_config_handler(config.clone())?;

        Ok(())
    }

    fn update(&mut self, action: Action) -> Result<Option<Action>> {
        match action {
            Action::SetMode(mode) => self.set_mode(mode),
            Action::SetTimemachineMode(timemachine_mode) => {
                self.set_timemachine_mode(timemachine_mode)
            }
            Action::SetNoTitle(is_no_title) => self.is_no_title = is_no_title,
            _ => {}
        }

        self.clock_component.update(action.clone())?;
        self.command_component.update(action.clone())?;
        self.interval_component.update(action.clone())?;
        self.execution_result_component.update(action.clone())?;
        self.history_component.update(action.clone())?;
        self.prompt_component.update(action.clone())?;
        self.status_component.update(action.clone())?;
        self.help_component.update(action.clone())?;

        Ok(None)
    }

    fn draw(&mut self, f: &mut Frame<'_>, area: Rect) -> Result<()> {
        f.render_widget(
            Block::new().style(self.config.get_style("background")),
            area,
        );

        if self.mode == Mode::Help {
            self.help_component.draw(f, area)?;

            return Ok(());
        }

        let header_length = if self.is_no_title { 0 } else { 3 };
        let [header, middle, footer] = Layout::vertical([
            Constraint::Length(header_length),
            Constraint::Fill(100),
            Constraint::Length(1),
        ])
        .areas(area);

        let [interval, command, clock] = Layout::horizontal([
            Constraint::Length(10),
            Constraint::Fill(100),
            Constraint::Length(21),
        ])
        .areas(header);

        self.command_component.draw(f, command)?;
        self.interval_component.draw(f, interval)?;
        self.clock_component.draw(f, clock)?;

        if self.timemachine_mode {
            let [execution_result, history] =
                Layout::horizontal([Constraint::Fill(100), Constraint::Length(21)]).areas(middle);
            self.history_component.draw(f, history)?;
            self.execution_result_component.draw(f, execution_result)?;
        } else {
            self.execution_result_component.draw(f, middle)?;
        }

        let [prompt, status] =
            Layout::horizontal([Constraint::Fill(100), Constraint::Length(32)]).areas(footer);
        self.prompt_component.draw(f, prompt)?;
        self.status_component.draw(f, status)?;

        Ok(())
    }
}
