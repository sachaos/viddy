const TAB_SIZE: usize = 4;

pub fn normalize_stdout(b: Vec<u8>) -> Vec<u8> {
    // Naively replace tabs ('\t') with at most `TAB_SIZE` spaces (' ') while
    // maintaining the alignment / elasticity per line (see tests below).
    let mut b = b;
    let (mut i, mut j) = (0, 0); // j tracks alignment
    while i < b.len() {
        if b[i] == b'\n' {
            (i, j) = (i + 1, 0);
        } else if b[i] == b'\t' {
            b[i] = b' ';
            let r = TAB_SIZE - (j % TAB_SIZE);
            for _ in 1..r {
                b.insert(i, b' ');
            }
            (i, j) = (i + r, 0);
        } else {
            (i, j) = (i + 1, j + 1);
        }
    }
    b
}

mod test {
    use super::*;

    #[test]
    fn test_normalize_stdout() {
        // Make sure we don't miss any tabs in edge cases.
        assert_eq!(
            normalize_stdout(b"\t\t\t\t\t".to_vec()),
            b"                    ".to_vec()
        );
        // Make sure tab is elastic (from 1 space to TAB_SIZE spaces).
        assert_eq!(normalize_stdout(b"\t12345".to_vec()), b"    12345".to_vec());
        assert_eq!(normalize_stdout(b"1\t2345".to_vec()), b"1   2345".to_vec());
        assert_eq!(normalize_stdout(b"12\t345".to_vec()), b"12  345".to_vec());
        assert_eq!(normalize_stdout(b"123\t45".to_vec()), b"123 45".to_vec());
        assert_eq!(normalize_stdout(b"1234\t5".to_vec()), b"1234    5".to_vec());
        // Make sure we reset alignment on new lines.
        assert_eq!(
            normalize_stdout(b"123\t\n4\t5".to_vec()),
            b"123 \n4   5".to_vec()
        );
        assert_eq!(
            normalize_stdout(b"12\t3\n4\t5".to_vec()),
            b"12  3\n4   5".to_vec()
        );
        assert_eq!(
            normalize_stdout(b"1\t23\n4\t5".to_vec()),
            b"1   23\n4   5".to_vec()
        );
        assert_eq!(
            normalize_stdout(b"\t123\n4\t5".to_vec()),
            b"    123\n4   5".to_vec()
        );
    }
}
