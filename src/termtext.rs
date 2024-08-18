use std::{
    fmt::{format, Display},
    io::Read,
    iter,
};

use anstyle::{AnsiColor, Color, RgbColor, Style};
use anstyle_parse::{DefaultCharAccumulator, ParamsIter, Parser, Perform};
use color_eyre::owo_colors::{colors::Default, OwoColorize};
use derive_deref::{Deref, DerefMut};
use serde::{Deserialize, Serialize};
use unicode_width::{UnicodeWidthChar, UnicodeWidthStr};

#[derive(Debug, Deref, DerefMut, Clone, Eq, PartialEq, Default, Serialize, Deserialize)]
pub struct Text {
    #[serde(skip_serializing, skip_deserializing)]
    pub chars: Vec<Char>,
}

impl From<Char> for Text {
    fn from(c: Char) -> Self {
        Self { chars: vec![c] }
    }
}

impl Text {
    pub fn new(text: &str) -> Self {
        Self {
            chars: text
                .chars()
                .map(|c| Char {
                    c,
                    style: Style::new(),
                })
                .collect(),
        }
    }

    pub fn mark_text(&mut self, start: usize, end: usize, style: Style) {
        for i in start..end {
            if let Some(c) = self.chars.get_mut(i) {
                c.style = style;
            }
        }
    }

    fn add_char(&mut self, c: Char) {
        self.chars.push(c);
    }

    pub fn lines(&self) -> Vec<Text> {
        let mut lines = Vec::new();
        let mut line = Text::new("");
        for c in self.chars.iter() {
            if c.c == '\n' {
                lines.push(line.clone());
                line = Text::new("");
            } else {
                line.add_char(*c);
            }
        }
        lines.push(line);
        lines
    }

    pub fn plain_text(&self) -> String {
        self.chars.iter().map(|c| c.c).collect()
    }
}

impl std::fmt::Display for Text {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let mut last_style = None;

        for char in &self.chars {
            let current_style = Some(char.style);
            if last_style != current_style {
                if last_style.is_some() {
                    // Only reset if there was a previous style
                    write!(f, "\x1B[0m")?;
                }
                write!(f, "{}", char.style)?;
                last_style = current_style;
            }
            write!(f, "{}", char.c)?; // Assuming `char` has a field `character`
        }

        Ok(())
    }
}

impl UnicodeWidthStr for Text {
    fn width(&self) -> usize {
        self.chars
            .iter()
            .map(|c| c.width().unwrap_or_default())
            .sum()
    }

    fn width_cjk(&self) -> usize {
        self.chars
            .iter()
            .map(|c| c.width_cjk().unwrap_or_default())
            .sum()
    }
}

pub struct Converter {
    original_style: Style,
    style: Style,
    text: Text,
}

impl Converter {
    pub fn new(original_style: Style) -> Self {
        Self {
            original_style,
            style: original_style,
            text: Text::new(""),
        }
    }

    fn reset_style(&mut self) {
        self.style = self.original_style;
    }

    pub fn convert(&mut self, text: &Vec<u8>) -> Text {
        let mut statemachine = Parser::<DefaultCharAccumulator>::new();
        let mut performer = Converter::new(self.style);

        let bytes = text.bytes();
        for c in bytes {
            statemachine.advance(&mut performer, c.unwrap());
        }

        performer.text
    }
}

impl Perform for Converter {
    fn print(&mut self, c: char) {
        self.text.add_char(Char {
            c,
            style: self.style,
        });
    }

    fn execute(&mut self, byte: u8) {
        if byte == b'\n' || byte == b'\t' {
            self.text.add_char(Char {
                c: byte as char,
                style: self.style,
            });
        }
    }

    fn hook(
        &mut self,
        _params: &anstyle_parse::Params,
        _intermediates: &[u8],
        _ignore: bool,
        _action: u8,
    ) {
    }

    fn put(&mut self, _byte: u8) {}

    fn unhook(&mut self) {}

    fn osc_dispatch(&mut self, _params: &[&[u8]], _bell_terminated: bool) {}

    fn csi_dispatch(
        &mut self,
        params: &anstyle_parse::Params,
        intermediates: &[u8],
        ignore: bool,
        byte: u8,
    ) {
        if ignore || intermediates.len() > 1 {
            return;
        }

        let is_sgr = byte == b'm' && intermediates.first().is_none();
        let style = if is_sgr {
            if params.iter().next() == Some(&[0]) || params.is_empty() {
                self.reset_style();
                return;
            } else {
                Some(ansi_term_style_from_sgr_parameters(&mut params.iter()))
            }
        } else {
            // Some(Element::Csi(0, 0))
            None
        };

        if let Some(style) = style {
            if let Some(c) = style.get_fg_color() {
                self.style = self.style.fg_color(Some(c));
            }

            if let Some(c) = style.get_bg_color() {
                self.style = self.style.bg_color(Some(c));
            }

            let mut ss_effects = self.style.get_effects();
            let effects = style.get_effects();
            ss_effects |= effects;
            self.style = self.style.effects(ss_effects);
        }
    }

    fn esc_dispatch(&mut self, _intermediates: &[u8], _ignore: bool, _byte: u8) {}
}

#[derive(Debug, Copy, Clone, Eq, PartialEq, Default)]
pub struct Char {
    pub c: char,
    pub style: Style,
}

impl std::fmt::Display for Char {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.c)
    }
}

impl UnicodeWidthChar for Char {
    fn width(self) -> Option<usize> {
        self.c.width()
    }

    fn width_cjk(self) -> Option<usize> {
        self.c.width_cjk()
    }
}

// Based on https://github.com/dandavison/delta/blob/f5b37173fe88a62e37208a9587a0ab4fec0ef107/src/ansi/iterator.rs#L173
fn ansi_term_style_from_sgr_parameters(params: &mut ParamsIter<'_>) -> Style {
    let mut style = Style::new();
    while let Some(param) = params.next() {
        style = match param {
            // [0] => Some(Attr::Reset),
            [1] => style.bold(),
            [2] => style.dimmed(),
            [3] => style.italic(),
            [4, ..] => style.underline(),
            [5] => style.blink(), // blink slow
            // [6] => style.blink_fast(), // blink fast
            // [7] => style.reversed(),
            [8] => style.hidden(),
            [9] => style.strikethrough(),
            // [21] => Some(Attr::CancelBold),
            // [22] => Some(Attr::CancelBoldDim),
            // [23] => Some(Attr::CancelItalic),
            // [24] => Some(Attr::CancelUnderline),
            // [25] => Some(Attr::CancelBlink),
            // [27] => Some(Attr::CancelReverse),
            // [28] => Some(Attr::CancelHidden),
            // [29] => Some(Attr::CancelStrike),
            [30] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::Black))),
            [31] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::Red))),
            [32] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::Green))),
            [33] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::Yellow))),
            [34] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::Blue))),
            [35] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::Magenta))),
            [36] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::Cyan))),
            [37] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::White))),
            [38] => {
                let mut iter = params.map(|param| param[0]);
                style.fg_color(parse_sgr_color(&mut iter))
            }
            [38, params @ ..] => {
                let rgb_start = if params.len() > 4 { 2 } else { 1 };
                let rgb_iter = params[rgb_start..].iter().copied();
                let mut iter = iter::once(params[0]).chain(rgb_iter);

                style.fg_color(parse_sgr_color(&mut iter))
            }
            // [39] => Some(Attr::Foreground(Color::Named(NamedColor::Foreground))),
            [40] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::Black))),
            [41] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::Red))),
            [42] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::Green))),
            [43] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::Yellow))),
            [44] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::Blue))),
            [45] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::Magenta))),
            [46] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::Cyan))),
            [47] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::White))),
            [48] => {
                let mut iter = params.map(|param| param[0]);
                style.bg_color(parse_sgr_color(&mut iter))
            }
            [48, params @ ..] => {
                let rgb_start = if params.len() > 4 { 2 } else { 1 };
                let rgb_iter = params[rgb_start..].iter().copied();
                let mut iter = iter::once(params[0]).chain(rgb_iter);
                style.bg_color(parse_sgr_color(&mut iter))
            }
            // [49] => Some(Attr::Background(Color::Named(NamedColor::Background))),
            [90] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightBlack))),
            [91] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightRed))),
            [92] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightGreen))),
            [93] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightYellow))),
            [94] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightBlue))),
            [95] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightMagenta))),
            [96] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightCyan))),
            [97] => style.fg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightWhite))),
            [100] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightBlack))),
            [101] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightRed))),
            [102] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightGreen))),
            [103] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightYellow))),
            [104] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightBlue))),
            [105] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightMagenta))),
            [106] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightCyan))),
            [107] => style.bg_color(Some(anstyle::Color::Ansi(AnsiColor::BrightWhite))),
            _ => style,
        };
    }
    style
}

// Based on https://github.com/dandavison/delta/blob/f5b37173fe88a62e37208a9587a0ab4fec0ef107/src/ansi/iterator.rs#L173
fn parse_sgr_color(params: &mut dyn Iterator<Item = u16>) -> Option<Color> {
    match params.next() {
        Some(2) => {
            let r = u8::try_from(params.next()?).ok()?;
            let g = u8::try_from(params.next()?).ok()?;
            let b = u8::try_from(params.next()?).ok()?;
            Some(Color::Rgb(RgbColor(r, g, b)))
        }
        Some(5) => Some(Color::Ansi256(anstyle::Ansi256Color(
            u8::try_from(params.next()?).ok()?,
        ))),
        _ => None,
    }
}

pub fn convert_to_anstyle(s: ratatui::style::Style) -> Style {
    let mut style = Style::default();
    if let Some(bg) = s.bg {
        style = style.bg_color(Some(convert_to_anstyle_color(bg)));
    }
    if let Some(fg) = s.fg {
        style = style.fg_color(Some(convert_to_anstyle_color(fg)));
    }
    style
}

pub fn convert_to_anstyle_color(c: ratatui::style::Color) -> Color {
    match c {
        ratatui::style::Color::Rgb(r, g, b) => Color::Rgb(RgbColor(r, g, b)),
        ratatui::style::Color::Indexed(i) => Color::Ansi256(anstyle::Ansi256Color(i)),
        _ => unimplemented!("Unsupported color: {c:?}"),
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn test_text() {
        let text = Text::new("Hello, world!");
        assert_eq!(text.to_string(), "Hello, world!");
    }

    #[test]
    fn test_text_accessed_by_index() {
        let text = Text::new("Hello, world!");
        assert_eq!(text[3].to_string(), "l");
    }

    #[test]
    fn test_convert() {
        // "\e[31mr\e[32mg\e[0m"
        // 00000000: 1b5b 3331 6d72 1b5b 3332 6d67 1b5b 306d  .[31mr.[32mg.[0m
        // 00000010: 0a
        let text: Vec<u8> = vec![
            0x1b, 0x5b, 0x33, 0x31, 0x6d, 0x72, 0x1b, 0x5b, 0x33, 0x32, 0x6d, 0x67, 0x1b, 0x5b,
            0x30, 0x6d, 0x0a,
        ];
        let mut converter = Converter::new(Style::new());
        let result = converter.convert(&text);

        assert_eq!(
            result,
            Text {
                chars: vec![
                    Char {
                        c: 'r',
                        style: Style::new().fg_color(Some(Color::Ansi(AnsiColor::Red)))
                    },
                    Char {
                        c: 'g',
                        style: Style::new().fg_color(Some(Color::Ansi(AnsiColor::Green)))
                    },
                    Char {
                        c: '\n',
                        style: Style::new()
                    },
                ]
            }
        );
    }

    #[test]
    fn test_convert_tab() {
        // echo "1\t2" | xxd
        // 00000000: 3109 320a                                1.2.
        let text: Vec<u8> = vec![0x31, 0x09, 0x32, 0x0a];
        let mut converter = Converter::new(Style::new());
        let result = converter.convert(&text);

        assert_eq!(
            result,
            Text {
                chars: vec![
                    Char {
                        c: '1',
                        style: Style::new(),
                    },
                    Char {
                        c: '\t',
                        style: Style::new(),
                    },
                    Char {
                        c: '2',
                        style: Style::new(),
                    },
                    Char {
                        c: '\n',
                        style: Style::new()
                    },
                ]
            }
        );
    }
}
