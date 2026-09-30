use std::process::{Command, Stdio};

/// Regression test: `viddy` used to abort when stdout was a closed pipe.
/// `Tui::drop` unwrapped the result of `exit()`, which fails when the
/// terminal cannot be entered, and the panic hook then constructed a
/// second `Tui` whose drop panicked *inside* the hook, so the process
/// died with a double panic (Unix `SIGABRT`, Windows `0xC0000409`)
/// instead of reporting the error.
///
/// A closed stdout must now produce a normal, non-panicking error exit.
#[test]
fn closed_stdout_does_not_abort() {
    let mut child = Command::new(env!("CARGO_BIN_EXE_viddy"))
        .args(["echo", "hi"])
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .expect("failed to spawn viddy");

    // Close the read end of stdout so that the first write fails.
    drop(child.stdout.take());

    let output = child.wait_with_output().expect("failed to wait on viddy");
    let stderr = String::from_utf8_lossy(&output.stderr);

    assert!(
        !stderr.contains("panic"),
        "viddy panicked on closed stdout:\n{stderr}"
    );
    // Rust panic exit code.
    assert_ne!(output.status.code(), Some(101));
    // SIGABRT (double panic) terminates the process without an exit code.
    #[cfg(unix)]
    assert!(
        output.status.code().is_some(),
        "viddy was killed by a signal: {output:?}"
    );
    // STATUS_STACK_BUFFER_OVERRUN, produced by abort() on Windows.
    #[cfg(windows)]
    assert_ne!(output.status.code(), Some(0xC000_0409u32 as i32));
    // Entering the TUI is impossible on a closed stdout, so a normal
    // failure is expected.
    assert!(!output.status.success());
}
