use dissimilar::{diff, Chunk};
use similar::{ChangeTag, TextDiff};

use crate::termtext::Text;

pub fn diff_and_mark(current: &str, pervious: &str, text: &mut Text) {
    let style = anstyle::Style::new()
        .fg_color(Some(anstyle::Color::Ansi(anstyle::AnsiColor::Black)))
        .bg_color(Some(anstyle::Color::Ansi(anstyle::AnsiColor::Green)));

    let chunks = diff(pervious, current);

    let mut cursor = 0;
    for chunk in chunks.into_iter() {
        match chunk {
            Chunk::Equal(s) => {
                cursor += s.chars().count();
            }
            Chunk::Insert(s) => {
                let length = s.chars().count();
                for c in s.chars() {
                    if !c.is_whitespace() {
                        text.mark_text(cursor, cursor + 1, style);
                    }
                    cursor += 1;
                }
            }
            Chunk::Delete(_) => {}
        }
    }
}

pub fn diff_and_mark_delete(current: &str, pervious: &str, text: &mut Text) {
    let style = anstyle::Style::new()
        .fg_color(Some(anstyle::Color::Ansi(anstyle::AnsiColor::Black)))
        .bg_color(Some(anstyle::Color::Ansi(anstyle::AnsiColor::Red)));

    let chunks = diff(pervious, current);

    let mut cursor = 0;
    for chunk in chunks {
        match chunk {
            Chunk::Equal(s) => {
                cursor += s.chars().count();
            }
            Chunk::Delete(s) => {
                let length = s.chars().count();
                for c in s.chars() {
                    if !c.is_whitespace() {
                        text.mark_text(cursor, cursor + 1, style);
                    }
                    cursor += 1;
                }
            }
            Chunk::Insert(_) => {}
        }
    }
}

#[cfg(test)]
mod test {
    use anstyle::Style;

    use crate::termtext::Text;

    #[test]
    fn test_diff_and_mark() {
        let current = "hello world!";
        let pervious = "hello world";
        let mut text = Text::new(current);
        let style = anstyle::Style::new()
            .fg_color(Some(anstyle::Color::Ansi(anstyle::AnsiColor::Black)))
            .bg_color(Some(anstyle::Color::Ansi(anstyle::AnsiColor::Green)));

        super::diff_and_mark(current, pervious, &mut text);

        assert_eq!(text[10].style, Style::new());
        assert_eq!(text[11].style, style);
    }

    #[test]
    fn test_diff_and_mark_new_line() {
        let pervious = "hello world";
        let current = "hello world\nnew world";
        let mut text = Text::new(current);
        let style = anstyle::Style::new()
            .fg_color(Some(anstyle::Color::Ansi(anstyle::AnsiColor::Black)))
            .bg_color(Some(anstyle::Color::Ansi(anstyle::AnsiColor::Green)));

        super::diff_and_mark(current, pervious, &mut text);

        assert_eq!(text[11].style, Style::new());
        for i in 12..=14 {
            assert_eq!(text[i].style, style);
        }
        assert_eq!(text[15].style, Style::new());
        for i in 16..=20 {
            assert_eq!(text[i].style, style);
        }
    }
}
