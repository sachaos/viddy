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

## Viddy v1.0.0 RC is available

Viddy was originally a program written in Go, but starting from version 1.0.0, it is being reimplemented in Rust. For more details, please see [the announcement](https://github.com/sachaos/viddy/issues/117). We are currently distributing a release candidate (RC) version of v1.0.0.

Since the build methods will change for each package management system, please take note and make the necessary adjustments.

We would greatly appreciate it if as many people as possible could test the RC version. If you are willing to help, we would be very grateful if you could install and test the RC version. Please refer to [the RC installation section](#install-v100-rc-version) for details.

If you find any bugs or areas for improvement, feel free to submit an issue.

## Features

* Basic features of original watch command.
    * Execute command periodically, and display the result.
    * color output.
    * diff highlight.
* Time machine mode. 😎
    * Rewind like video.
    * Go to the past, and back to the future.
* Look back history.
    * Save and load history.
* See output in pager.
* Vim like keymaps.
* Search text.
* Suspend and restart execution.
* Support shell alias
    * See detail https://github.com/sachaos/viddy/issues/2#issuecomment-904002053
* Customize keymappings.
* Customize color.

## Install

### Mac

#### [Homebrew](https://brew.sh)

```shell
brew install viddy
```

#### [MacPorts](https://www.macports.org)

```shell
sudo port install viddy
```

### Windows

#### [Scoop](https://scoop.sh/)

To install Viddy on Windows, first install the Scoop package manager, and then run the commands below.

**NOTE**: The git package is required in order to add additional Scoop "buckets".

```
scoop install git
scoop bucket add extras
scoop install extras/viddy
```

### Linux

```shell
wget -O viddy.tar.gz https://github.com/sachaos/viddy/releases/download/v0.4.0/viddy_Linux_x86_64.tar.gz && tar xvf viddy.tar.gz && mv viddy /usr/local/bin
```

#### ArchLinux ( AUR )

```shell
yay -S viddy
```
Alternatively you can use the [AUR Git repo](https://aur.archlinux.org/packages/viddy/) directly

#### Alpine Linux

After [enabling the community repository](https://wiki.alpinelinux.org/wiki/Enable_Community_Repository):

```shell
apk add viddy
```

### [asdf version manager](https://asdf-vm.com)

```shell
asdf plugin add viddy
asdf install viddy latest
asdf global viddy latest
```

### Other

Download from [release page](https://github.com/sachaos/viddy/releases).

## Install v1.0.0 RC version

### Mac

```shell
brew install sachaos/tap/viddy-rc
```

### Linux

```shell
wget -O viddy.tar.gz https://github.com/sachaos/viddy/releases/download/v1.0.0-rc.3/viddy-v1.0.0-rc.3-linux-x86_64.tar.gz && tar xvf viddy.tar.gz && mv viddy /usr/local/bin
```

### Other

Download from [release page](https://github.com/sachaos/viddy/releases/tag/v1.0.0-rc.3).

## Keymaps

| key       |                                            |
|-----------|--------------------------------------------|
| SPACE     | Toggle time machine mode                   |
| s         | Toggle <ins>s</ins>uspend execution                   |
| b         | Toggle ring terminal <ins>b</ins>ell                  |
| d         | Toggle <ins>d</ins>iff                                |
| t         | Toggle header/<ins>t</ins>itle display                      |
| ?         | Toggle help view                           |
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

Viddy can be used without any configuration.
However, if you want to customize the keybindings or default behavior, you can do so.

Install your config file on `$XDG_CONFIG_HOME/viddy.toml`
On macOS, the path is `~/Library/Application\ Support/viddy.toml`.

```toml
[general]
no_shell = false
shell = "zsh"
shell_options = ""
skip_empty_diffs = false

[keymap]
timemachine_go_to_past = "Down"
timemachine_go_to_more_past = "Shift-Down"
timemachine_go_to_future = "Up"
timemachine_go_to_more_future = "Shift-Up"
timemachine_go_to_now = "Ctrl-Shift-Up"
timemachine_go_to_oldest = "Ctrl-Shift-Down"

[color]
background = "white" # Default value is inherit from terminal color.
```

## What is "viddy" ?

"viddy" is Nadsat word meaning to see.
Nadsat is fictional argot of gangs in the violent book and movie "A Clockwork Orange".

## Credits

The gopher's logo of viddy is licensed under the Creative Commons 3.0 Attributions license.

The original Go gopher was designed by [Renee French](https://reneefrench.blogspot.com/).
