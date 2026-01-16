---
sidebar_position: 1
---

# Introduction

Sidecar is a terminal UI for monitoring AI coding agent sessions.

## Overview

Sidecar provides a unified terminal interface for viewing Claude Code conversations, git status, and task progress. Built for developers who want visibility into their AI coding sessions without leaving the terminal.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/marcus/sidecar/main/scripts/setup.sh | bash
```

## Requirements

- Go 1.21+
- macOS, Linux, or WSL

## Quick Start

After installation, run from any project directory:

```bash
sidecar
```

## Features

- **Git Status** - View staged, modified, and untracked files with syntax-highlighted diffs
- **Conversations** - Browse Claude Code session history with message content and token usage
- **TD Monitor** - Integration with TD task management for AI agents
- **File Browser** - Navigate project files with tree view and preview
- **Worktrees** - Manage git worktrees for parallel development

For more details, see the [GitHub repository](https://github.com/marcus/sidecar).
