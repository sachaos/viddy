use std::{collections::HashMap, time::Duration};

use color_eyre::eyre::Result;
use crossterm::event::{KeyCode, KeyEvent};
use ratatui::{prelude::*, widgets::*};
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc::UnboundedSender;

use super::{Component, Frame};
use crate::{
    action::Action,
    config::{Config, KeyBindings, RuntimeConfig},
};

pub struct Interval {
    command_tx: Option<UnboundedSender<Action>>,
    config: Config,
    runtime_config: RuntimeConfig,
}

impl Interval {
    pub fn new(runtime_config: RuntimeConfig) -> Self {
        Self {
            runtime_config,
            command_tx: None,
            config: Config::new().unwrap(),
        }
    }

    pub fn increase_interval(&mut self) {
        self.runtime_config.interval += chrono::Duration::milliseconds(500);
    }

    pub fn decrease_interval(&mut self) {
        let new_interval = self.runtime_config.interval - chrono::Duration::milliseconds(500);

        if new_interval >= chrono::Duration::milliseconds(500) {
            self.runtime_config.interval = new_interval;
        }
    }
}

impl Component for Interval {
    fn register_action_handler(&mut self, tx: UnboundedSender<Action>) -> Result<()> {
        self.command_tx = Some(tx);
        Ok(())
    }

    fn register_config_handler(&mut self, config: Config) -> Result<()> {
        self.config = config;
        Ok(())
    }

    fn update(&mut self, action: Action) -> Result<Option<Action>> {
        Ok(None)
    }

    fn draw(&mut self, f: &mut Frame<'_>, area: Rect) -> Result<()> {
        let block = Block::default()
            .title("Every")
            .borders(Borders::ALL)
            .border_style(self.config.get_style("border"))
            .title_style(self.config.get_style("title"));
        let text =
            humantime::format_duration(self.runtime_config.interval.to_std().unwrap_or_default())
                .to_string();
        let paragraph = Paragraph::new(text).block(block);

        f.render_widget(paragraph, area);
        Ok(())
    }
}
