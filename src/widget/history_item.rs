use chrono::{DateTime, Duration, Local};
use ratatui::prelude::*;
use tui_widget_list::{List, ListState, PreRender, PreRenderContext};

use crate::{config::Config, types::ExecutionId};

#[derive(Debug, Clone)]
pub struct HisotryItem {
    pub id: ExecutionId,
    pub diff: Option<(u32, u32)>,
    pub start_time: DateTime<Local>,
    pub exit_code: Option<i32>,
    pub is_running: bool,
    pub style: Style,
    pub interval: Duration,
    pub count: usize,
    pub selector_style: Style,
    pub secondary_text_style: Style,
}

impl HisotryItem {
    pub fn new(
        id: ExecutionId,
        start_time: DateTime<Local>,
        duration: Duration,
        selector_style: Style,
        secondary_text_style: Style,
    ) -> Self {
        Self {
            id,
            start_time,
            diff: None,
            exit_code: None,
            is_running: true,
            style: Style::default(),
            interval: duration,
            count: 1,
            selector_style,
            secondary_text_style,
        }
    }

    pub fn update_diff(&mut self, diff: Option<(u32, u32)>, exit_code: i32) {
        self.diff = diff;
        self.exit_code = Some(exit_code);
        self.is_running = false;
    }

    pub fn update_same_count(&mut self) {
        self.count += 1;
    }
}

impl PreRender for HisotryItem {
    fn pre_render(&mut self, context: &PreRenderContext) -> u16 {
        if context.is_selected {
            self.style = self.selector_style;
        };

        1
    }
}

impl Widget for HisotryItem {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let time_style = if self.is_running {
            self.secondary_text_style
        } else {
            Style::default().fg(Color::White)
        };

        let mut spans = vec![];
        spans.push(if self.interval >= Duration::seconds(1) {
            Span::raw(self.start_time.format("%H:%M:%S").to_string()).style(time_style)
        } else {
            Span::raw(self.start_time.format("%M:%S.%3f").to_string()).style(time_style)
        });

        if self.is_running {
            spans.push(Span::raw(" Running").style(self.secondary_text_style));
            Line::from("")
                .spans(spans)
                .style(self.style)
                .render(area, buf);
            return;
        }

        let exit_code = self.exit_code.unwrap_or_default();
        if exit_code > 0 {
            let exit_code = Span::styled(
                format!(" E({})", exit_code),
                Style::default().fg(Color::Red),
            );
            spans.push(exit_code);
        } else {
            match self.diff {
                Some((0, 0)) => spans.push(Span::styled(" ±0", self.secondary_text_style)),
                Some((diff_add, diff_delete)) => {
                    let add =
                        Span::styled(format!(" +{}", diff_add), Style::default().fg(Color::Green));
                    let delete = Span::styled(
                        format!(" -{}", diff_delete),
                        Style::default().fg(Color::Red),
                    );
                    spans.push(add);
                    spans.push(delete);
                }
                _ => (),
            };
        }

        if self.count > 1 {
            spans.push(Span::styled(
                format!(" *{}", self.count),
                self.secondary_text_style,
            ));
        }

        Line::raw("")
            .spans(spans)
            .style(self.style)
            .render(area, buf);
    }
}
