use unicode_width::UnicodeWidthChar;

const TAB_SIZE: usize = 4;

pub fn normalize_stdout(s: &[u8]) -> Vec<u8> {
    // Naively replace tabs ('\t') with at most `TAB_SIZE` spaces (' ') while
    // maintaining the alignment / elasticity per line (see tests below).
    let str = String::from_utf8_lossy(s).to_string();
    let mut b = Vec::with_capacity(str.len() * TAB_SIZE);
    let mut chars = str.chars();
    let mut width = 0;
    while let Some(c) = chars.next() {
        let count = skip_ansi_escape_sequence(c, &mut chars.clone());
        if count > 0 {
            b.push(c);
            for _ in 0..count {
                b.push(chars.next().unwrap_or(' '));
            }
            continue;
        }

        if c == '\t' {
            let r = TAB_SIZE - (width % TAB_SIZE);
            b.resize(b.len() + r, ' ');
            width += r;
        } else if c == '\n' {
            b.push('\n');
            width = 0;
        } else {
            b.push(c);
            width += c.width().unwrap_or(1);
        }
    }
    b.into_iter().collect::<String>().into_bytes()
}

// Based on https://github.com/mgeisler/textwrap/blob/63970361d1d653ec8715acb931c3c109750d4a57/src/core.rs
/// The CSI or “Control Sequence Introducer” introduces an ANSI escape
/// sequence. This is typically used for colored text and will be
/// ignored when computing the text width.
const CSI: (char, char) = ('\x1b', '[');
/// The final bytes of an ANSI escape sequence must be in this range.
const ANSI_FINAL_BYTE: std::ops::RangeInclusive<char> = '\x40'..='\x7e';
/// Skip ANSI escape sequences.
///
/// The `ch` is the current `char`, the `chars` provide the following
/// characters. The `chars` will be modified if `ch` is the start of
/// an ANSI escape sequence.
///
/// Returns `usize` the count of skipped characters
fn skip_ansi_escape_sequence<I: Iterator<Item = char>>(ch: char, chars: &mut I) -> usize {
    let mut count = 0;
    if ch != CSI.0 {
        return 0; // Nothing to skip here.
    }

    let next = chars.next();
    count += 1;
    if next == Some(CSI.1) {
        // We have found the start of an ANSI escape code, typically
        // used for colored terminal text. We skip until we find a
        // "final byte" in the range 0x40–0x7E.
        for ch in chars {
            count += 1;
            if ANSI_FINAL_BYTE.contains(&ch) {
                break;
            }
        }
    } else if next == Some(']') {
        // We have found the start of an Operating System Command,
        // which extends until the next sequence "\x1b\\" (the String
        // Terminator sequence) or the BEL character. The BEL
        // character is non-standard, but it is still used quite
        // often, for example, by GNU ls.
        let mut last = ']';
        for new in chars {
            count += 1;
            if new == '\x07' || (new == '\\' && last == CSI.0) {
                break;
            }
            last = new;
        }
    }

    count
}

mod test {
    use super::*;

    #[test]
    fn test_normalize_stdout() {
        assert_eq!(normalize_stdout(b"\t"), b"    ");
        // Make sure we don't miss any tabs in edge cases.
        assert_eq!(normalize_stdout(b"\t\t\t\t\t"), b"                    ");
        // Make sure tab is elastic (from 1 space to TAB_SIZE spaces).
        assert_eq!(normalize_stdout(b"\t12345"), b"    12345");
        assert_eq!(normalize_stdout(b"1\t2345"), b"1   2345");
        assert_eq!(normalize_stdout(b"12\t345"), b"12  345");
        assert_eq!(normalize_stdout(b"123\t45"), b"123 45");
        assert_eq!(normalize_stdout(b"1234\t5"), b"1234    5");
        // Make sure we reset alignment on new lines.
        assert_eq!(normalize_stdout(b"123\t\n4\t5"), b"123 \n4   5");
        assert_eq!(normalize_stdout(b"12\t3\n4\t5"), b"12  3\n4   5");
        assert_eq!(normalize_stdout(b"1\t23\n4\t5"), b"1   23\n4   5");
        assert_eq!(normalize_stdout(b"\t123\n4\t5"), b"    123\n4   5");
        assert_eq!(
            normalize_stdout("あ\tい\nう\tえ".as_bytes()),
            "あ  い\nう  え".as_bytes()
        );
        assert_eq!(
            normalize_stdout(b"\x1b[34ma\t\x1b[39mb\x1b[0m"),
            b"\x1b[34ma   \x1b[39mb\x1b[0m"
        );
    }
}
