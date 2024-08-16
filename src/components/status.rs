use std::{collections::HashMap, time::Duration};

use chrono::{DateTime, Local};
use color_eyre::eyre::Result;
use crossterm::event::{KeyCode, KeyEvent};
use ratatui::{prelude::*, widgets::*};
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc::UnboundedSender;

use super::{Component, Frame};
use crate::{
    action::{Action, DiffMode},
    config::{Config, KeyBindings, RuntimeConfig},
};

pub struct Status {
    command_tx: Option<UnboundedSender<Action>>,
    config: Config,

    is_fold: bool,
    diff_mode: Option<DiffMode>,
    is_suspend: bool,
    is_bell: bool,
}

impl Status {
    pub fn new(is_fold: bool, diff_mode: Option<DiffMode>, is_bell: bool) -> Self {
        Self {
            command_tx: None,
            config: Config::new().unwrap(),
            is_fold,
            diff_mode,
            is_suspend: false,
            is_bell,
        }
    }
}

impl Component for Status {
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
            Action::SetFold(is_fold) => self.is_fold = is_fold,
            Action::SetDiff(diff_mode) => self.diff_mode = diff_mode,
            Action::SetBell(is_bell) => self.is_bell = is_bell,
            Action::SetSuspend(is_suspend) => self.is_suspend = is_suspend,
            _ => {}
        }
        Ok(None)
    }

    fn draw(&mut self, f: &mut Frame<'_>, area: Rect) -> Result<()> {
        let enabled_style = Style::default().fg(Color::White).bold();
        let disabled_style = self.config.get_style("secondary_text");

        let mut status = vec![Span::styled(
            "[F]old",
            if self.is_fold {
                enabled_style
            } else {
                disabled_style
            },
        )];
        if let Some(mode) = self.diff_mode {
            status.push(Span::styled(
                " [D]iff",
                if self.diff_mode.is_some() {
                    enabled_style
                } else {
                    disabled_style
                },
            ));
            status.push(match self.diff_mode {
                Some(DiffMode::Add) => Span::styled("+", Style::new().fg(Color::Green).bold()),
                Some(DiffMode::Delete) => Span::styled("-", Style::new().fg(Color::Red).bold()),
                _ => Span::raw(" "),
            });
        } else {
            status.push(Span::styled(" [D]iff±", disabled_style));
        };
        status.push(Span::styled(
            " [S]uspend",
            if self.is_suspend {
                enabled_style
            } else {
                disabled_style
            },
        ));
        status.push(Span::styled(
            " [B]ell",
            if self.is_bell {
                enabled_style
            } else {
                disabled_style
            },
        ));

        let line = Line::raw("").spans(status);
        let paragraph = Paragraph::new(line).alignment(Alignment::Right);
        f.render_widget(paragraph, area);
        Ok(())
    }
}
