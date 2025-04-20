# DevDeck Server

A Go-based WebSocket server for DevDeck, a customizable desktop control deck application.

## Overview

DevDeck Server is the backend component of the DevDeck ecosystem, designed to facilitate communication between the DevDeck app and your computer. It enables users to create custom control buttons for launching applications, running commands, and organizing workflows through a configurable interface.

## Features

- WebSocket-based communication
- Command execution on the host machine
- Application launching
- mDNS service discovery
- Configurable layout and button configurations
- Context-based command grouping

## Configuration

DevDeck uses TOML configuration files stored at `~/.config/devdeck/devdeck.toml`. The configuration includes:

### Layout

```toml
[layout]
columns = 4
background_color = "#000000"
button_size = 60
```

### Commands

```toml
[[commands]]
uuid = "unique-id" # Optional; auto-generated if not provided
description = "Launch Application"
app = "Application Name" # For 'open' action
action = "open" # Or specify a command to run
icon = "icon-name" # Icon from https://ionic.io/ionicons to display on button
type = "action" # Or "context" to show sub-commands
context = "main" # Or custom context name
main = true # Whether to show on main screen
```

## Development

### Prerequisites

- Go 1.24+

### Installation

```bash
brew tap devdeck-app/homebrew-devdeck-server
brew install devdeck-server
```

### Running

You can either run it manually:

```bash
devdeck-server
```

Or you can run it on the background:

```bash
brew services start devdeck-server
```

The server will start on port 4242 by default, but you can specify the port in the config file: `$HOME/.config/devdeck/config.toml`

```toml
[server]
port=4242
```

## Examples

### Contexts

#### Creating a System Preferences context and adding two commands

```toml
[[commands]]
app = "System Preferences"
type = "context"
icon = "cog-outline"
context = "system"
main = true

[[commands]]
description = "Volume Down"
action = "aerospace volume down"
type = "action"
context = "system"

[[commands]]
action = "aerospace volume up"
description = "Volume Up"
type = "action"
context = "system"
```

### Opening an application

```toml
[[commands]]
app = "Obsidian"
icon = "document-outline"
action = "open"
type = "action"
main = true
```

## License

[MIT License](LICENSE)
