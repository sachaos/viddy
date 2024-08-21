use std::{collections::HashMap, time::Duration};

use ansi_to_tui::IntoText;
use color_eyre::{eyre::Result, owo_colors::OwoColorize};
use crossterm::event::{KeyCode, KeyEvent};
use ratatui::{prelude::*, widgets::*};
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc::UnboundedSender;
use tracing::Instrument;

use super::{Component, Frame};
use crate::{
    action::Action,
    config::{Config, KeyBindings, RuntimeConfig},
    mode::Mode,
};

pub struct Help {
    command_tx: Option<UnboundedSender<Action>>,
    config: Config,
    keybindings: HashMap<(Mode, String), Vec<Vec<KeyEvent>>>,
    y_position: u16,
    y_area_size: u16,
}

fn keys_str(
    keybindings: &HashMap<(Mode, String), Vec<Vec<KeyEvent>>>,
    mode: Mode,
    action: String,
) -> Vec<Span> {
    keybindings.get(&(mode, action.clone())).map_or_else(
        || vec![Span::from("None")],
        |keys| {
            keys.iter()
                .map(|keys| {
                    let mut s = String::new();
                    for key in keys {
                        s.push('<');
                        s.push_str(&display_key(key));
                        s.push('>');
                    }
                    Span::styled(s, Style::default().fg(Color::Yellow))
                })
                .intersperse(Span::from(", "))
                .collect()
        },
    )
}

impl Help {
    pub fn new(config: Config) -> Self {
        Self {
            command_tx: None,
            config: config.clone(),
            keybindings: get_action_keys(config.keybindings),
            y_position: 0,
            y_area_size: 0,
        }
    }

    fn scroll_down(&mut self) {
        self.y_position = self.y_position.saturating_add(1);
    }

    fn scroll_up(&mut self) {
        self.y_position = self.y_position.saturating_sub(1);
    }

    fn page_up(&mut self) {
        self.y_position = self.y_position.saturating_sub(self.y_area_size);
    }

    fn page_down(&mut self) {
        self.y_position = self.y_position.saturating_add(self.y_area_size);
    }

    fn half_page_up(&mut self) {
        self.y_position = self.y_position.saturating_sub(self.y_area_size / 2);
    }

    fn half_page_down(&mut self) {
        self.y_position = self.y_position.saturating_add(self.y_area_size / 2);
    }

    fn reset_position(&mut self) {
        self.y_position = 0;
    }
}

fn display_key(key: &KeyEvent) -> String {
    let mut s: String = String::new();

    for m in key.modifiers.iter() {
        match m {
            crossterm::event::KeyModifiers::CONTROL => s.push_str("Ctrl-"),
            crossterm::event::KeyModifiers::ALT => s.push_str("Alt-"),
            crossterm::event::KeyModifiers::SHIFT => s.push_str("Shift-"),
            _ => {}
        }
    }

    match key.code {
        KeyCode::Char(' ') => s.push_str("SPACE"),
        KeyCode::Char(c) => s.push(c),
        KeyCode::Enter => s.push_str("Enter"),
        KeyCode::Backspace => s.push_str("Backspace"),
        KeyCode::Left => s.push_str("Left"),
        KeyCode::Right => s.push_str("Right"),
        KeyCode::BackTab => s.push_str("BackTab"),
        KeyCode::Tab => s.push_str("Tab"),
        KeyCode::Home => s.push_str("Home"),
        KeyCode::End => s.push_str("End"),
        KeyCode::Up => s.push_str("Up"),
        KeyCode::Down => s.push_str("Down"),
        KeyCode::PageUp => s.push_str("PageUp"),
        KeyCode::PageDown => s.push_str("PageDown"),
        KeyCode::Delete => s.push_str("Delete"),
        KeyCode::Insert => s.push_str("Insert"),
        KeyCode::F(i) => s.push_str(format!("F{:?}", i).as_str()),
        KeyCode::Null => s.push_str("Null"),
        KeyCode::Esc => s.push_str("Esc"),
        KeyCode::CapsLock => s.push_str("CapsLock"),
        KeyCode::ScrollLock => s.push_str("ScrollLock"),
        KeyCode::NumLock => s.push_str("NumLock"),
        KeyCode::PrintScreen => s.push_str("PrintScreen"),
        KeyCode::Pause => s.push_str("Pause"),
        KeyCode::Menu => s.push_str("Menu"),
        KeyCode::KeypadBegin => s.push_str("KeypadBegin"),
        KeyCode::Media(c) => s.push_str(format!("Media({:?})", c).as_str()),
        KeyCode::Modifier(c) => s.push_str(format!("Modifier({:?})", c).as_str()),
    };

    s
}

impl Component for Help {
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
            Action::ShowHelp => self.reset_position(),
            Action::HelpScrollDown => self.scroll_down(),
            Action::HelpScrollUp => self.scroll_up(),
            Action::HelpPageDown => self.page_down(),
            Action::HelpPageUp => self.page_up(),
            Action::HelpHalfPageDown => self.half_page_down(),
            Action::HelpHalfPageUp => self.half_page_up(),
            _ => {}
        }
        Ok(None)
    }

    fn draw(&mut self, f: &mut Frame<'_>, area: Rect) -> Result<()> {
        let basic_keys = [
            (
                "Toggle time machine mode  ",
                Mode::All,
                Action::SwitchTimemachineMode.to_string(),
            ),
            (
                "Toggle suspend execution  ",
                Mode::All,
                Action::SwitchSuspend.to_string(),
            ),
            (
                "Toggle ring terminal bell ",
                Mode::All,
                Action::SwitchBell.to_string(),
            ),
            (
                "Toggle diff               ",
                Mode::All,
                Action::SwitchDiff.to_string(),
            ),
            (
                "Toggle deletion diff      ",
                Mode::All,
                Action::SwitchDeletionDiff.to_string(),
            ),
            (
                "Toggle header display     ",
                Mode::All,
                Action::SwitchNoTitle.to_string(),
            ),
            (
                "Toggle help view          ",
                Mode::All,
                Action::ShowHelp.to_string(),
            ),
            (
                "Toggle unfold             ",
                Mode::All,
                Action::SwitchFold.to_string(),
            ),
            (
                "Quit Viddy                ",
                Mode::All,
                Action::Quit.to_string(),
            ),
        ];

        let pager_keys = [
            (
                "Search text           ",
                Mode::All,
                Action::EnterSearchMode.to_string(),
            ),
            (
                "Move to next line     ",
                Mode::All,
                Action::ResultScrollDown.to_string(),
            ),
            (
                "Move to previous line ",
                Mode::All,
                Action::ResultScrollUp.to_string(),
            ),
            (
                "Move to right         ",
                Mode::All,
                Action::ScrollRight.to_string(),
            ),
            (
                "Move to left          ",
                Mode::All,
                Action::ScrollLeft.to_string(),
            ),
            (
                "Page down             ",
                Mode::All,
                Action::ResultPageDown.to_string(),
            ),
            (
                "Page up               ",
                Mode::All,
                Action::ResultPageUp.to_string(),
            ),
            (
                "Half page down        ",
                Mode::All,
                Action::ResultHalfPageDown.to_string(),
            ),
            (
                "Half page up          ",
                Mode::All,
                Action::ResultHalfPageUp.to_string(),
            ),
            (
                "Go to top of page     ",
                Mode::All,
                Action::BottomOfPage.to_string(),
            ),
            (
                "Go to bottom of page  ",
                Mode::All,
                Action::TopOfPage.to_string(),
            ),
        ];

        let timemachine_keys = [
            (
                "Go to the past           ",
                Mode::All,
                Action::GoToPast.to_string(),
            ),
            (
                "Back to the future       ",
                Mode::All,
                Action::GoToFuture.to_string(),
            ),
            (
                "Go to more past          ",
                Mode::All,
                Action::GoToMorePast.to_string(),
            ),
            (
                "Back to more future      ",
                Mode::All,
                Action::GoToMoreFuture.to_string(),
            ),
            (
                "Go to oldest position    ",
                Mode::All,
                Action::GoToOldest.to_string(),
            ),
            (
                "Back to current position ",
                Mode::All,
                Action::GoToCurrent.to_string(),
            ),
        ];

        let mut lines = vec![
            Line::from("Press ESC or q to go back"),
            Line::from(""),
            Line::styled(
                " Key Bindings",
                Style::default().add_modifier(Modifier::BOLD),
            ),
            Line::from(""),
            Line::from(vec![
                Span::from("   "),
                Span::styled(
                    "General",
                    Style::default().add_modifier(Modifier::UNDERLINED),
                ),
            ]),
            Line::from(""),
        ];

        for (action, mode, key) in basic_keys.into_iter() {
            let keys_str = keys_str(&self.keybindings, mode, key);
            lines.push(Line::from(
                [
                    vec![
                        Span::from("   "),
                        Span::styled(action, Style::default().add_modifier(Modifier::BOLD)),
                        Span::from(": "),
                    ],
                    keys_str,
                ]
                .concat(),
            ));
        }

        lines.push(Line::from(""));
        lines.push(Line::from(vec![
            Span::from("   "),
            Span::styled("Pager", Style::default().add_modifier(Modifier::UNDERLINED)),
        ]));
        lines.push(Line::from(""));

        for (description, mode, action) in pager_keys.into_iter() {
            let keys_str = keys_str(&self.keybindings, mode, action);
            lines.push(Line::from(
                [
                    vec![
                        Span::from("   "),
                        Span::styled(description, Style::default().add_modifier(Modifier::BOLD)),
                        Span::from(": "),
                    ],
                    keys_str,
                ]
                .concat(),
            ));
        }

        lines.push(Line::from(""));
        lines.push(Line::from(vec![
            Span::from("   "),
            Span::styled(
                "Time machine",
                Style::default().add_modifier(Modifier::UNDERLINED),
            ),
        ]));
        lines.push(Line::from(""));

        for (action, mode, key) in timemachine_keys.into_iter() {
            let keys_str = keys_str(&self.keybindings, mode, key);
            lines.push(Line::from(
                [
                    vec![
                        Span::from("   "),
                        Span::styled(action, Style::default().add_modifier(Modifier::BOLD)),
                        Span::from(": "),
                    ],
                    keys_str,
                ]
                .concat(),
            ));
        }

        lines.push(Line::from(""));

        self.y_position = self
            .y_position
            .min((lines.len().saturating_sub(area.height as usize)) as u16);

        let paragraph = Paragraph::new(Text::from(lines)).scroll((self.y_position, 0));
        f.render_widget(paragraph, area);

        self.y_area_size = area.height;
        Ok(())
    }
}

fn get_action_keys(keybindings: KeyBindings) -> HashMap<(Mode, String), Vec<Vec<KeyEvent>>> {
    let mut action_keys: HashMap<(Mode, String), Vec<Vec<KeyEvent>>> = HashMap::new();
    keybindings.iter().for_each(|(mode, bindings)| {
        bindings.iter().for_each(|(event, action)| {
            action_keys
                .entry((*mode, action.to_string()))
                .and_modify(|keys| {
                    keys.push(event.clone());
                })
                .or_insert_with(|| vec![event.clone()]);
        });
    });
    action_keys
}
