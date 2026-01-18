#!/bin/bash
# Setup PATH for tools installed via nvm, homebrew, etc.
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[ -s "$NVM_DIR/nvm.sh" ] && source "$NVM_DIR/nvm.sh" 2>/dev/null
# Fallback: source shell profile if nvm not found
if ! command -v node &>/dev/null; then
  [ -f "$HOME/.zshrc" ] && source "$HOME/.zshrc" 2>/dev/null
  [ -f "$HOME/.bashrc" ] && source "$HOME/.bashrc" 2>/dev/null
fi

claude --dangerously-skip-permissions "$(cat <<'SIDECAR_PROMPT_EOF'
Start a td review session and do a detailed review of open reviews. If you find obvious fixes create a td bug task with a detailed description of the problem and fix them immediately. Ask questions if needed. Create new td tasks with detailed descriptions to fix bigger bugs or tasks if needed for fixing in a later session. Make sure the changes have tests, and if not, create tasks to test them. For tasks that can be reviewed in parallel, use subagents. Once tasks with reviews that are related to previously opened bugs are complete, make sure to close the in progress tasks.
SIDECAR_PROMPT_EOF
)"
rm -f "/Users/marcusvorwaller/code/sidecar-review-session/.sidecar-start.sh"
