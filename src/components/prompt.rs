use std::{collections::HashMap, time::Duration};

use color_eyre::eyre::Result;
use crossterm::event::{KeyCode, KeyEvent};
use ratatui::{prelude::*, widgets::*};
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc::UnboundedSender;
use tui_input::{backend::crossterm::EventHandler, Input};

use super::{Component, Frame};
use crate::{
    action::Action,
    config::{Config, KeyBindings, RuntimeConfig},
};

#[derive(Default)]
pub struct Prompt {
    command_tx: Option<UnboundedSender<Action>>,
    pub input: Input,
    is_searching: bool,
    is_inputtig: bool,

    config: Config,
}

impl Prompt {
    pub fn new() -> Self {
        Self::default()
    }

    fn handle_key_event(&mut self, key_event: KeyEvent) -> Result<()> {
        self.input
            .handle_event(&crossterm::event::Event::Key(key_event));
        if let Some(tx) = &self.command_tx {
            tx.send(Action::SetSearchQuery(self.input.value().to_string()))?;
        }

        Ok(())
    }

    fn enter_search_mode(&mut self) -> Result<()> {
        self.is_inputtig = true;
        self.is_searching = true;
        self.input = Input::default();
        if let Some(tx) = &self.command_tx {
            tx.send(Action::SetSearchQuery(self.input.value().to_string()))?;
        }

        Ok(())
    }

    fn exit_search_mode(&mut self) -> Result<()> {
        self.is_inputtig = false;
        self.is_searching = false;
        self.input = Input::default();
        if let Some(tx) = &self.command_tx {
            tx.send(Action::SetSearchQuery(self.input.value().to_string()))?;
        }

        Ok(())
    }

    fn execute_search(&mut self) -> Result<()> {
        self.is_inputtig = false;
        self.is_searching = true;

        Ok(())
    }
}

impl Component for Prompt {
    fn register_action_handler(&mut self, tx: UnboundedSender<Action>) -> Result<()> {
        self.command_tx = Some(tx);
        Ok(())
    }

    fn register_config_handler(&mut self, config: Config) -> Result<()> {
        self.config = config.clone();
        Ok(())
    }

    fn update(&mut self, action: Action) -> Result<Option<Action>> {
        match action {
            Action::EnterSearchMode => self.enter_search_mode()?,
            Action::KeyEventForPrompt(key_event) => self.handle_key_event(key_event)?,
            Action::ExecuteSearch => self.execute_search()?,
            Action::ExitSearchMode => self.exit_search_mode()?,
            _ => {}
        }
        Ok(None)
    }

    fn draw(&mut self, f: &mut Frame<'_>, area: Rect) -> Result<()> {
        if self.is_inputtig {
            f.set_cursor(area.x + self.input.visual_cursor() as u16 + 1, area.y);
        }

        let input = if !self.is_searching {
            String::new()
        } else {
            format!("/{}", self.input.value())
        };
        let paragraph = Paragraph::new(input);
        f.render_widget(paragraph, area);

        Ok(())
    }
}
