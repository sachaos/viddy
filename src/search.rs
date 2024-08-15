use anstyle::{AnsiColor, Color, Style};

use crate::termtext::Text;

pub fn search_and_mark(string: &str, text: &mut Text, query: &str, style: Style) {
    let mut byte_to_char_idx = vec![0; string.len() + 1];
    let mut char_idx = 0;
    for (i, c) in string.char_indices() {
        byte_to_char_idx[i] = char_idx;
        char_idx += 1;
    }
    byte_to_char_idx[string.len()] = char_idx; // for the last character

    for (byte_index, _) in string.match_indices(query) {
        let char_index = byte_to_char_idx[byte_index];
        text.mark_text(char_index, char_index + query.chars().count(), style);
    }
}

mod test {
    use super::*;

    #[test]
    fn test_search_and_mark() {
        let mut text = Text::new("hello world");
        let style = Style::new()
            .fg_color(Some(Color::Ansi(AnsiColor::Black)))
            .bg_color(Some(Color::Ansi(AnsiColor::Yellow)));
        search_and_mark("hello world", &mut text, "hello", style);
        assert_eq!(text.to_string(), "\u{1b}[30m\u{1b}[43mhello\u{1b}[0m world");
    }
}
