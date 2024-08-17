use std::{collections::HashMap, time::Duration};

use ansi_parser::{AnsiParser, AnsiSequence, Output};
use ansi_to_tui::IntoText;
use chrono::{DateTime, Local};
use color_eyre::eyre::Result;
use crossterm::event::{KeyCode, KeyEvent};
use ratatui::{prelude::*, widgets::*};
use serde::{Deserialize, Serialize};
use symbols::scrollbar;
use tokio::sync::mpsc::UnboundedSender;
use tracing_subscriber::field::debug;
use unicode_width::UnicodeWidthStr;

use super::{Component, Frame};
use crate::{
    action::Action,
    config::{Config, KeyBindings, RuntimeConfig},
    termtext::{Char, Text},
};

pub struct ExecutionResult {
    command_tx: Option<UnboundedSender<Action>>,
    config: Config,

    result: Option<Text>,

    x_state: ScrollbarState,
    y_state: ScrollbarState,
    x_position: u16,
    y_position: u16,
    x_area_size: u16,
    y_area_size: u16,
    y_max_scroll_size: u16,
    fold: bool,
}

impl ExecutionResult {
    pub fn new(fold: bool) -> Self {
        Self {
            command_tx: None,
            config: Config::new().unwrap(),
            result: None,
            x_state: ScrollbarState::default(),
            y_state: ScrollbarState::default(),
            fold,
            x_area_size: 0,
            y_area_size: 0,
            y_max_scroll_size: 0,
            x_position: 0,
            y_position: 0,
        }
    }

    fn set_result(&mut self, new: Option<Text>) {
        self.result = new;
    }

    fn scroll_down(&mut self) {
        self.y_position = self.y_position.saturating_add(1);
        self.y_state = self.y_state.position(self.y_position as usize);
    }

    fn scroll_up(&mut self) {
        self.y_position = self.y_position.saturating_sub(1);
        self.y_state = self.y_state.position(self.y_position as usize);
    }

    fn page_up(&mut self) {
        self.y_position = self.y_position.saturating_sub(self.y_area_size);
        self.y_state = self.y_state.position(self.y_position as usize);
    }

    fn page_down(&mut self) {
        self.y_position = self.y_position.saturating_add(self.y_area_size);
        self.y_state = self.y_state.position(self.y_position as usize);
    }

    fn half_page_up(&mut self) {
        self.y_position = self.y_position.saturating_sub(self.y_area_size / 2);
        self.y_state = self.y_state.position(self.y_position as usize);
    }

    fn half_page_down(&mut self) {
        self.y_position = self.y_position.saturating_add(self.y_area_size / 2);
        self.y_state = self.y_state.position(self.y_position as usize);
    }

    fn bottom_of_page(&mut self) {
        self.y_position = self.y_max_scroll_size;
        self.y_state = self.y_state.position(self.y_position as usize);
    }

    fn top_of_page(&mut self) {
        self.y_position = 0;
        self.y_state = self.y_state.position(self.y_position as usize);
    }

    fn scroll_right(&mut self) {
        self.x_position = self.x_position.saturating_add(10);
        self.x_state = self.x_state.position(self.x_position as usize);
    }

    fn scroll_left(&mut self) {
        self.x_position = self.x_position.saturating_sub(10);
        self.x_state = self.x_state.position(self.x_position as usize);
    }

    fn set_fold(&mut self, is_fold: bool) {
        self.fold = is_fold;
    }
}

fn text_width(text: &Text) -> usize {
    text.lines()
        .into_iter()
        .map(|l| l.width())
        .max()
        .unwrap_or(0)
}

fn text_height(text: &Text) -> usize {
    text.lines().len()
}

impl Component for ExecutionResult {
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
            Action::SetResult(result) => self.set_result(result),
            Action::ResultScrollDown => self.scroll_down(),
            Action::ResultScrollUp => self.scroll_up(),
            Action::ScrollRight => self.scroll_right(),
            Action::ScrollLeft => self.scroll_left(),
            Action::ResultPageUp => self.page_up(),
            Action::ResultPageDown => self.page_down(),
            Action::ResultHalfPageDown => self.half_page_down(),
            Action::ResultHalfPageUp => self.half_page_up(),
            Action::SetFold(is_fold) => self.set_fold(is_fold),
            Action::BottomOfPage => self.bottom_of_page(),
            Action::TopOfPage => self.top_of_page(),
            _ => {}
        }
        Ok(None)
    }

    fn draw(&mut self, f: &mut Frame<'_>, area: Rect) -> Result<()> {
        let text = self.result.clone().unwrap_or(Text::new(""));
        let mut current = text.to_string();
        let mut y_max;
        let mut x_max;
        if self.fold {
            x_max = area.width as usize;
            let folded_text = fold_text(&text, x_max);
            current = folded_text.to_string();
            y_max = text_height(&folded_text);
            if y_max > area.height as usize {
                x_max = (area.width - 1) as usize;
                let folded_text = fold_text(&text, x_max);
                current = folded_text.to_string();
                y_max = text_height(&folded_text);
            }

            self.x_position = 0;
            self.x_state = self.x_state.position(0);
        } else {
            x_max = text_width(&text);
            y_max = text_height(&text);
        }

        let mut body = area;

        let mut y_scrollable = y_max.saturating_sub(body.height as usize);
        let mut x_scrollable = x_max.saturating_sub(body.width as usize);
        let scroll_style = self.config.get_style("scrollbar");

        if y_scrollable > 0 {
            body.width = area.width.saturating_sub(1);
            let scrollbar = Scrollbar::new(ScrollbarOrientation::VerticalRight)
                .symbols(scrollbar::VERTICAL)
                .style(scroll_style)
                .thumb_symbol("║");
            f.render_stateful_widget(scrollbar, area, &mut self.y_state);
            if x_max > body.width as usize {
                x_scrollable = x_scrollable.saturating_add(1);
            }
        }

        if x_scrollable > 0 {
            body.height = area.height.saturating_sub(1);
            let scrollbar = Scrollbar::new(ScrollbarOrientation::HorizontalBottom)
                .symbols(scrollbar::HORIZONTAL)
                .style(scroll_style)
                .thumb_symbol("=");
            f.render_stateful_widget(scrollbar, area, &mut self.x_state);
            if y_max > body.height as usize {
                y_scrollable = y_scrollable.saturating_add(1);
            }
        }

        if y_scrollable > 0 {
            body.width = area.width.saturating_sub(1);
            let scrollbar = Scrollbar::new(ScrollbarOrientation::VerticalRight)
                .symbols(scrollbar::VERTICAL)
                .style(scroll_style)
                .thumb_symbol("║");
            f.render_stateful_widget(scrollbar, area, &mut self.y_state);
            if x_max > body.width as usize {
                x_scrollable = x_scrollable.saturating_add(1);
            }
        }

        self.y_state = self.y_state.content_length(y_scrollable);
        self.x_state = self.x_state.content_length(x_scrollable);

        if self.x_position > x_scrollable as u16 {
            self.x_position = x_scrollable as u16;
            self.x_state = self.x_state.position(x_scrollable);
        }

        if self.y_position > y_scrollable as u16 {
            self.y_position = y_scrollable as u16;
            self.y_state = self.y_state.position(y_scrollable);
        }

        let current = current.into_text()?;
        let paragraph = Paragraph::new(current).scroll((self.y_position, self.x_position));
        f.render_widget(paragraph, body);

        self.y_max_scroll_size = y_scrollable as u16;
        self.y_area_size = body.height;

        Ok(())
    }
}

fn fold_text(str: &Text, width: usize) -> Text {
    let mut result: Vec<Char> = Vec::new();
    let mut current = 0;
    let mut previous_style = anstyle::Style::default();
    for char in str.chars.iter() {
        let c = char.to_string();
        if c == "\n" {
            current = 0;
            result.push(*char);
            previous_style = char.style;
            continue;
        }

        if current == width {
            let char = Char {
                c: '\n',
                style: previous_style,
            };
            result.push(char);

            current = 0;
        }

        if current + c.width() > width {
            let char = Char {
                c: '\n',
                style: previous_style,
            };
            result.push(char);
            current = 0;
        }

        result.push(*char);
        previous_style = char.style;
        current += c.width();
    }

    Text { chars: result }
}

fn remove_ansi(text: &str) -> String {
    text.ansi_parse()
        .filter(|o| matches!(o, Output::TextBlock(_)))
        .map(|o| o.to_string())
        .collect::<String>()
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn test_fold_text() {
        let text = Text::new("hello world");
        let result = fold_text(&text, 5);
        assert_eq!(result.to_string(), "hello\n worl\nd");
    }

    #[test]
    fn test_fold_text_long() {
        let str = r#"use std::{collections::HashMap, time::Duration};
use chrono::{DateTime, Local};
use color_eyre::eyre::Result;
    "#;
        let text = Text::new(str);

        let result = fold_text(&text, 97);

        assert_eq!(result.to_string(), str)
    }

    #[test]
    fn test_fold_text_wide_chars() {
        let text = Text::new("あいうえおかきくけこさしすせそたちつてとなにぬねの");

        let result = fold_text(&text, 10);

        assert_eq!(
            result.to_string(),
            "あいうえお\nかきくけこ\nさしすせそ\nたちつてと\nなにぬねの"
        )
    }

    #[test]
    fn test_fold_text_wide_chars_2() {
        let text = Text::new("iあいうえおかきくけこさしすせそたちつてとなにぬねの");

        let result = fold_text(&text, 10);

        assert_eq!(
            result.to_string(),
            "iあいうえ\nおかきくけ\nこさしすせ\nそたちつて\nとなにぬね\nの"
        )
    }

    #[test]
    fn test_remove_ansi() {
        let text = "\u{1b}[31mredtextredtext\u{1b}[0m";
        let result = remove_ansi(text);
        assert_eq!(result, "redtextredtext");
    }
}
