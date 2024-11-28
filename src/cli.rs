use std::path::PathBuf;

use chrono::{format::Parsed, Duration};
use clap::Parser;
use color_eyre::eyre::{bail, eyre, Result};
use serde_with::serde_as;

use crate::utils::version;

#[cfg(not(target_os = "windows"))]
const SHELL_HELP: &str = "Shell [default: sh]";

#[cfg(target_os = "windows")]
const SHELL_HELP: &str = "Shell [default: cmd]";

#[serde_as]
#[derive(Parser, Debug, Clone)]
#[command(author, version = version(), about)]
pub struct Cli {
    #[arg(
    short = 'n',
    long = "interval",
    value_parser = validate_duration,
    default_value = "2s",
    help = "Seconds to wait between updates (>= 100ms)",
  )]
    pub interval: Duration,

    #[arg(
        name = "differences",
        short = 'd',
        long = "differences",
        help = "Highlight changes between updates"
    )]
    pub is_diff: bool,

    #[arg(
        short = 'D',
        long = "deletion-differences",
        conflicts_with = "differences",
        help = "Highlight deletion changes between updates"
    )]
    pub is_deletion_diff: bool,

    #[arg(
        short = 'p',
        long = "precise",
        help = "Attempt run command in precise intervals"
    )]
    pub is_precise: bool,

    #[arg(short = 't', long = "no-title", help = "Turn off header")]
    pub is_no_title: bool,

    #[arg(
        short = 'w',
        long = "unfold",
        alias = "no-wrap",
        help = "Turn off line wrapping"
    )]
    pub is_unfold: bool,

    #[arg(long = "shell", help = SHELL_HELP)]
    pub shell: Option<String>,

    #[arg(
        short = 's',
        long = "skip-empty-diffs",
        help = "Skip snapshots with no changes (±0) in history"
    )]
    pub is_skip_empty_diffs: bool,

    #[arg(
    long = "shell-options",
    num_args(0..),
    help = "Additional shell options"
  )]
    pub shell_options: Option<Vec<String>>,

    #[arg(
        short = 'b',
        long = "bell",
        help = "Ring terminal bell changes between updates"
    )]
    pub is_bell: bool,

    #[arg(value_name = "COMMAND", num_args(0..), allow_hyphen_values = true, help = "Command to run")]
    pub command: Vec<String>,

    #[arg(
        short = 'x',
        long = "exec",
        help = "Pass command to exec instead of \"sh -c\"",
        conflicts_with = "shell"
    )]
    pub is_exec: bool,

    #[arg(long = "debug")]
    pub is_debug: bool,

    #[arg(
        long = "save",
        value_name = "FILE",
        help = "Path to the backup file. If not provided, a temporary file will be created",
        conflicts_with_all = ["disable_auto_save", "load"]
    )]
    pub save: Option<PathBuf>,

    #[arg(
        long = "disable_auto_save",
        help = "Disable to save automatically",
        conflicts_with_all = ["save", "load"]
    )]
    pub disable_auto_save: bool,

    #[arg(long = "disable_mouse", help = "Stop handling mouse events")]
    pub disable_mouse: bool,

    #[arg(
        long = "load",
        alias = "lookback",
        value_name = "FILE",
        help = "Path to the backup file",
        conflicts_with_all = ["save", "disable_auto_save", "shell", "shell_options", "is_exec", "is_bell", "is_precise", "interval"]
    )]
    pub load: Option<PathBuf>,
}

fn validate_duration(s: &str) -> Result<Duration> {
    let d = parse_duration_from_str(s)?;
    if d < Duration::milliseconds(100) {
        bail!("The short interval is not allowed (less than 100ms)");
    }

    Ok(d)
}

fn parse_duration_from_str(s: &str) -> Result<Duration> {
    match humantime::parse_duration(s) {
        Ok(d) => Ok(Duration::from_std(d)?),
        Err(_) => {
            // If the input is only a number, we assume it's in seconds
            let n = s.parse::<f64>()?;
            Ok(Duration::milliseconds((n * 1000.0) as i64))
        }
    }
}
