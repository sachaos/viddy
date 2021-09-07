# Viddy

<p align="center">
<img src="images/logo.png" width="200" alt="viddy" title="viddy" />
</p>

Modern `watch` command.

Viddy well, gopher. Viddy well.

## Demo

<p align="center">
<img src="images/demo.gif" width="100%" alt="viddy" title="viddy" />
</p>


## Features

* Basic features of original watch command.
    * Execute command periodically, and display the result.
    * color output.
    * diff highlight.
* Time machine mode. 😎
    * Rewind like video.
    * Go to the past, and back to the future.
* See output in pager.
* Vim like keymaps.
* Search text.
* Suspend and restart execution.
* Run command in precise intervals forcibly.

## Install

### Mac

...via [Homebrew](https://brew.sh):

```shell
brew install sachaos/tap/viddy
```

...via [MacPorts](https://www.macports.org):

```shell
sudo port install viddy
```

### Linux

```shell
wget -O viddy.tar.gz https://github.com/sachaos/viddy/releases/download/v0.1.10/viddy_0.1.10_Linux_x86_64.tar.gz && tar xvf viddy.tar.gz && mv viddy /usr/local/bin
```

#### ArchLinux ( AUR )
```shell
yay -S viddy
```
Alternatively you can use the [AUR Git repo](https://aur.archlinux.org/packages/viddy/) directly
### Go

```shell
go install github.com/sachaos/viddy@latest
```

Download from [release page](https://github.com/sachaos/viddy/releases)

## Keymaps

| key       |                                            |
|-----------|--------------------------------------------|
| SPACE     | Toggle time machine mode                   |
| s         | Toggle suspend execution                   |
| d         | Toggle diff                                |
| t         | Toggle header display                      |
| /         | Search text                                |
| j         | Pager: next line                           |
| k         | Pager: previous line                       |
| Control-F | Pager: page down                           |
| Control-B | Pager: page up                             |
| g         | Pager: go to top of page                   |
| Shift-G   | Pager: go to bottom of page                |
| Shift-J   | (Time machine mode) Go to the past         |
| Shift-K   | (Time machine mode) Back to the future     |
| Shift-F   | (Time machine mode) Go to more past        |
| Shift-B   | (Time machine mode) Back to more future    |
| Shift-O   | (Time machine mode) Go to oldest position  |
| Shift-N   | (Time machine mode) Go to current position |

## Configuration

Install your config file on `$XDG_CONFIG_HOME/viddy.toml`
On macOS, the path is `~/Library/Application\ Support/viddy.toml`.

```toml
[keymap]
timemachine_go_to_past = "Down"
timemachine_go_to_more_past = "Shift-Down"
timemachine_go_to_future = "Up"
timemachine_go_to_more_future = "Shift-Up"

[color]
background = "white" # Default value is inherit from terminal color.
```

## What is "viddy" ?

"viddy" is Nadsat word meaning to see.
Nadsat is fictional argot of gangs in the violence movie "A Clockwork Orange".

## Credits

The gopher's logo of viddy is licensed under the Creative Commons 3.0 Attributions license.

The original Go gopher was designed by [Renee French](https://reneefrench.blogspot.com/).
