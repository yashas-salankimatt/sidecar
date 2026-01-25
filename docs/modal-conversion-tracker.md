# Modal Conversion Tracker

This document tracks the conversion of all modals in Sidecar to the new `internal/modal` library system.

**Legend:**
- ✓ = Converted to `internal/modal`
- ○ = Not yet converted

---

## App-Level Modals

| Modal | Type | Converted |
|-------|------|-----------|
| Quit Confirm | Confirmation | ✓ |
| Command Palette | Search | ○ |
| Help Modal | Info | ○ |
| Diagnostics | Info | ○ |
| Project Switcher | Selection | ○ |
| Project Add | Form | ○ |
| Theme Switcher | Selection | ○ |
| Community Browser | Search | ○ |

---

## Workspace Plugin Modals

| Modal | Type | Converted |
|-------|------|-----------|
| Create Worktree | Form | ✓ |
| Task Link | Dropdown | ✓ |
| Confirm Delete | Confirmation | ○ |
| Confirm Delete Shell | Confirmation | ○ |
| Rename Shell | Form | ○ |
| Prompt Picker | Selection | ○ |
| Agent Choice | Selection | ○ |
| Merge Workflow | Multi-step | ○ |
| Commit for Merge | Form | ○ |
| Type Selector | Selection | ○ |

---

## Git Status Plugin Modals

| Modal | Type | Converted |
|-------|------|-----------|
| Confirm Discard | Confirmation | ✓ |
| Confirm Stash Pop | Confirmation | ✓ |
| Pull Menu | Selection | ○ |
| Pull Conflict | Menu | ○ |
| Push Menu | Selection | ○ |
| Commit Message | Form | ✓ |
| Branch Picker | Selection | ○ |

---

## File Browser Plugin Modals

| Modal | Type | Converted |
|-------|------|-----------|
| Blame View | Info | ○ |
| File Info | Info | ○ |
| Project Search | Results | ○ |

---

## Summary

**Total Modals:** 28
**Converted:** 6 (21%)
**Remaining:** 22 (79%)
