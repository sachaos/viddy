use color_eyre::eyre::Result;
use serde::{Deserialize, Serialize};

use crate::utils;

#[derive(Debug, Serialize, Deserialize)]
pub struct OldConfig {
    #[serde(default)]
    pub general: Option<General>,
    #[serde(default)]
    pub keymap: Option<Keymap>,
    #[serde(default)]
    pub color: Option<Color>,
}

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct General {
    pub no_shell: Option<bool>,
    pub shell: Option<String>,
    pub shell_options: Option<String>,
    pub skip_empty_diffs: Option<bool>,
}

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct Keymap {
    pub toggle_timemachine: Option<String>,
    pub timemachine_go_to_past: Option<String>,
    pub timemachine_go_to_future: Option<String>,
    pub timemachine_go_to_more_past: Option<String>,
    pub timemachine_go_to_more_future: Option<String>,
    pub timemachine_go_to_now: Option<String>,
    pub timemachine_go_to_oldest: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct Color {
    pub background: Option<String>,
}

impl OldConfig {
    pub fn new_from_str(s: &str) -> Result<Self> {
        let config: OldConfig = toml::from_str(s)?;
        Ok(config)
    }

    pub fn new() -> Result<Self> {
        let config_dir = utils::get_old_config_dir();
        let file_path = config_dir.join("viddy.toml");
        let config_str = std::fs::read_to_string(file_path)?;
        let config = OldConfig::new_from_str(&config_str)?;
        Ok(config)
    }
}

#[cfg(test)]
mod test {
    #[test]
    fn test_old_config() {
        let config_str = r#"
[color]
background = "white"
"#;
        let config = super::OldConfig::new_from_str(config_str).unwrap();
        assert_eq!(config.color.unwrap().background, Some("white".to_string()));
        assert!(config.general.is_none());
    }

    #[test]
    fn test_old_config_skip_empty_diffs() {
        let config_str = r#"
[general]
skip_empty_diffs = true
"#;
        let config = super::OldConfig::new_from_str(config_str).unwrap();
        assert_eq!(config.general.unwrap().skip_empty_diffs, Some(true));
    }
}
