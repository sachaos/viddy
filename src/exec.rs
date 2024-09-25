use color_eyre::eyre::{OptionExt, Result};
use tokio::process::Command;

pub async fn exec(
    command: Vec<String>,
    shell: Option<(String, Vec<String>)>,
) -> Result<(Vec<u8>, Vec<u8>, i32)> {
    let (command, args) = prepare_command(command, shell);
    let mut command = Command::new(command);

    let (width, height) = crossterm::terminal::size()?;
    command.env("COLUMNS", width.to_string());
    command.env("LINES", height.to_string());
    command.args(args);
    let result = command.output().await?;

    Ok((
        result.stdout,
        result.stderr,
        result.status.code().ok_or_eyre("failed to get exit code")?,
    ))
}

fn prepare_command(
    command: Vec<String>,
    shell: Option<(String, Vec<String>)>,
) -> (String, Vec<String>) {
    if cfg!(target_os = "windows") && !is_pwsh(&shell) {
        let cmd_str = command.join(" ");
        return ("cmd".to_string(), vec!["/C".to_string(), cmd_str]);
    }

    if let Some((shell, mut shell_options)) = shell {
        if !shell_options.contains(&"-c".to_string()) {
            shell_options.push("-c".to_string());
        }
        shell_options.push(command.join(" "));
        (shell, shell_options)
    } else {
        (command[0].clone(), command[1..].to_vec())
    }
}

fn is_pwsh(shell: &Option<(String, Vec<String>)>) -> bool {
    if let Some((shell, _)) = shell {
        shell == "pwsh"
    } else {
        false
    }
}
