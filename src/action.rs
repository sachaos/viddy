use std::{fmt, string::ToString};

use chrono::{DateTime, Local};
use crossterm::event::{KeyEvent, MouseEvent};
use serde::{
    de::{self, Deserializer, Visitor},
    Deserialize, Serialize,
};
use strum::Display;

use crate::{mode::Mode, termtext::Text, types::ExecutionId};

#[derive(Debug, Clone, Eq, PartialEq, Copy, Serialize, Deserialize)]
pub enum DiffMode {
    Add,
    Delete,
}

#[derive(Debug, Clone, PartialEq, Eq, Display, Serialize, Deserialize)]
pub enum Action {
    Tick,
    Render,
    Resize(u16, u16),
    Suspend,
    Resume,
    Quit,
    Refresh,
    MouseEvent(MouseEvent),
    Error(String),
    Help,
    StartExecution(ExecutionId, DateTime<Local>),
    FinishExecution(ExecutionId, DateTime<Local>, Option<(u32, u32)>, i32),
    ShowExecution(ExecutionId, ExecutionId),
    SetClock(DateTime<Local>),
    SetResult(Option<Text>),
    SetMode(Mode),
    SwitchTimemachineMode,
    SetTimemachineMode(bool),
    EnterSearchMode,
    ExecuteSearch,
    ExitSearchMode,
    SetSearchQuery(String),
    KeyEventForPrompt(KeyEvent),
    GoToPast,
    GoToFuture,
    GoToMorePast,
    GoToMoreFuture,
    GoToOldest,
    GoToCurrent,
    ScrollLeft,
    ScrollRight,
    ResultScrollDown,
    ResultScrollUp,
    HelpScrollDown,
    HelpScrollUp,
    ResultPageDown,
    ResultPageUp,
    HelpPageDown,
    HelpPageUp,
    ResultHalfPageDown,
    ResultHalfPageUp,
    HelpHalfPageDown,
    HelpHalfPageUp,
    BottomOfPage,
    TopOfPage,
    SwitchFold,
    SetFold(bool),
    SetDiff(Option<DiffMode>),
    SwitchDiff,
    SwitchDeletionDiff,
    SwitchSuspend,
    SetSuspend(bool),
    SwitchBell,
    SetBell(bool),
    DiffDetected,
    SetNoTitle(bool),
    SwitchNoTitle,
    InsertHistory(ExecutionId, DateTime<Local>),
    UpdateHistoryResult(ExecutionId, Option<(u32, u32)>, i32),
    UpdateLatestHistoryCount,
    ShowHelp,
    ExitHelp,
}
