#![allow(dead_code)]
#![allow(unused_imports)]
#![allow(unused_variables)]
#![feature(iter_intersperse)]

pub mod action;
pub mod app;
pub mod cli;
pub mod components;
pub mod config;
mod diff;
mod exec;
pub mod mode;
mod old_config;
mod runner;
mod search;
mod store;
mod termtext;
pub mod tui;
mod types;
pub mod utils;
mod widget;

use chrono::Duration;
use clap::Parser;
use cli::Cli;
use color_eyre::eyre::Result;

use crate::{
    app::App,
    utils::{initialize_logging, initialize_panic_handler, version},
};

async fn tokio_main() -> Result<()> {
    initialize_logging()?;

    initialize_panic_handler()?;

    let args = Cli::parse();
    let interval = Duration::from(args.interval);
    let store = store::MemoryStore::new();
    let mut app = App::new(args, store)?;
    app.run().await?;

    Ok(())
}

#[tokio::main]
async fn main() -> Result<()> {
    if let Err(e) = tokio_main().await {
        eprintln!("{} error: Something went wrong", env!("CARGO_PKG_NAME"));
        Err(e)
    } else {
        Ok(())
    }
}
