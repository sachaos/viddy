pub fn normalize_stdout(b: Vec<u8>) -> Vec<u8> {
    // To fix '\t' width issue replace '\t' with '    '
    let mut b = b;
    let mut i = 0;
    while i < b.len() {
        if b[i] == b'\t' {
            b[i] = b' ';
            b.insert(i, b' ');
            b.insert(i, b' ');
            b.insert(i, b' ');
            i += 4;
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
        let b = b"hello\tworld".to_vec();
        let b = normalize_stdout(b);
        assert_eq!(b, b"hello    world".to_vec());
    }
}
