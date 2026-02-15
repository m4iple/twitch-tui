# Twitch TUI

A terminal user interface application for interacting with Twitch chat built with Go.

## Overview

Twitch TUI is a command-line application that provides a rich terminal interface for connecting to and viewing Twitch chat. The application uses the Charmbracelet ecosystem for UI rendering and the Twitch IRC protocol for real-time chat interaction.

## Features

- Real-time Twitch chat viewing in the terminal
- Terminal User Interface built with Bubbletea and Bubbles
- Configurable theme support
- Message formatting and display
- Command support for chat interaction

## Configuration

Configuration is managed automatically through `config.toml`:

- **Twitch Settings**: Channel name, OAuth token, and refresh credentials
- **Theme**: Customizable color palette for the interface

Configuration updates are saved automatically as you use the application.

## Commands

Commands are prefixed with a colon:

- **:login** or **:l** - Authenticate with Twitch
  - Usage: `:login <username> <oauth_token> <refresh_token>`
  
- **:join** or **:j** - Switch to a different channel
  - Usage: `:join <channel_name>`
  
- **:find** or **:f** - Filter messages by search term
  - Usage: `:find <search_string>` (use `:find` with no args to clear filter)
  
- **:quit** or **:q** - Exit the application
  - Usage: `:quit`

## Keyboard Shortcuts

- **Ctrl+Q** - Quit the application
- **Ctrl+F** - Open find command (prefills `:find `)
- **Ctrl+J** - Open join command (prefills `:join `)