const TAB_SIZE: usize = 4;

pub fn normalize_stdout(b: Vec<u8>) -> Vec<u8> {
    // Naively replace tabs ('\t') with at most `TAB_SIZE` spaces (' ') while
    // maintaining the alignment / elasticity (see tests below).
    let mut b = b;
    let mut i = 0;
    while i < b.len() {
        if b[i] == b'\t' {
            b[i] = b' ';
            let r = TAB_SIZE - (i % TAB_SIZE);
            for _ in 1..r {
                b.insert(i, b' ');
            }
            i += r - 1;
        } else {
            i += 1;
        }
    }
    b
}

mod test {
    use super::*;

    #[test]
    fn test_normalize_stdout() {
        assert_eq!(normalize_stdout(b"\t12345".to_vec()), b"    12345".to_vec());
        assert_eq!(normalize_stdout(b"1\t2345".to_vec()), b"1   2345".to_vec());
        assert_eq!(normalize_stdout(b"12\t345".to_vec()), b"12  345".to_vec());
        assert_eq!(normalize_stdout(b"123\t45".to_vec()), b"123 45".to_vec());
        assert_eq!(normalize_stdout(b"1234\t5".to_vec()), b"1234    5".to_vec());
    }
}
