# NASC - TUI Calculator Specification

## Overview
A terminal-based calculator using Charm's Bubbletea framework with libqalculate integration for mathematical expression evaluation.

## Build Configuration
- **Binary name**: `nasc`

## Architecture
- **src/main.go**: Core application logic
- **src/calculator.go**: All the calculator integration
- **src/ui.go**: UI handling and message routing
- **src/events.go**: Event handling and key bindings
- **src/rendering.go**: UI rendering and viewport management
- **src/input.go**: Input processing and line management
- **src/ui_utils.go**: UI utilities and command functions
- **src/undo.go**: Undo/redo system implementation
- **src/session.go**: Session storage (list, load, save, rename, duplicate, delete)
- **src/picker.go**: Fuzzy session picker popup
- **src/style.go**: Theme definitions and color management
- **src/textutil.go**: Unicode-safe string helpers and visual truncation
- **src/calc_wrapper.cpp**: C++ wrapper for libqalculate library
- **Makefile**: Build configuration

## Performance Guidelines
Based on Bubbletea best practices from https://leg100.github.io/en/posts/building-bubbletea-programs/#keepfast:

### Keep the Event Loop Fast
- Never block the `Update()` method with expensive operations
- Offload time-consuming tasks to `tea.Cmd` functions that run in separate goroutines
- Make sure the ui updates are done in the correct fashion so no conflicts or crashes can occur
- Process messages sequentially to maintain responsive UI

### Avoid Race Conditions
- Only modify model state within the `Update()` method
- Never modify the model from outside the event loop
- Use commands for concurrent operations that need to update state

### Message Processing
- Messages from commands may not be processed in order
- Use `tea.Sequence()` if order matters
- Keep the update method lightweight and fast

## Features
- Multi-line calculator with line-by-line evaluation
- Variable references (`ans`, `ans1`, `ans2`, etc.)
- Mouse click support for result insertion
- Terminal color palette theming
- Real-time expression evaluation
- Interactive help system with scrollable content
- Function completion with descriptions
- Auto-completion for functions, variables, and answer references
- Comprehensive undo/redo system with 50-level history
- Named sessions persisted to disk with autosave and a fuzzy picker

## Key Bindings

### Main Interface
- **Enter**: Add new input line
- **Up/Down**: Navigate between lines
- **Backspace**: Delete empty line (when multiple lines exist)
- **Ctrl+D**: Delete line
- **Ctrl+N**: New session (prompts for a name)
- **Ctrl+O**: Open the session picker
- **Ctrl+Z**: Undo last action
- **Ctrl+Y**: Redo last undone action
- **Ctrl+L**: Go to line (opens line number input dialog)
- **Ctrl+T**: Insert the built-in example sheet
- **Ctrl+S**: Copy the focused line's result to the clipboard
- **Tab/Ctrl+Space**: Show completion proposals
- **Ctrl+H**: Show help popup
- **Esc**: Quit application (or close active popup)
- **Ctrl+C**: Force quit application

### Help System
- **Ctrl+H**: Toggle help popup
- **Up/Down**: Scroll help content line by line
- **Page Up/Down**: Scroll help content by half page
- **Esc**: Close help popup

### Special Input
- **Ctrl+P**: Insert π symbol
- **Ctrl+R**: Insert √ symbol
- **Ctrl+A**: Insert "ans" (last answer reference)
- **Ctrl+T**: Insert example expressions
- **Ctrl+S**: Copy focused result

## Completion Proposals
The calculator provides intelligent function and variable completion through a popup interface.

### Activation
- **Tab**: Show completion popup for current word
- **Ctrl+Space**: Show completion popup for current word

### Navigation
- **Up/Down** or **Ctrl+P/Ctrl+N**: Navigate through completion options
- **Enter/Tab/Ctrl+Y**: Accept selected completion
- **Esc**: Close completion popup
- **Any other key**: Continue typing and filter completions

### Completion Order
1. **Answer references**: `ans`, `ans1`, `ans2`, etc. (most commonly used)
2. **Basic functions**: Core mathematical functions (sin, cos, log, sqrt, etc.)
3. **Advanced functions**: Specialized functions (physics, statistics, etc.)

### Function Categorization
- **Basic Functions**: Essential math functions from categories like:
  - Basic trigonometry (sin, cos, tan, asin, acos, atan)
  - Basic logarithms (log, ln, exp, sqrt)
  - Basic arithmetic and algebra functions
  - Simple number theory (abs, gcd, lcm)

- **Advanced Functions**: Specialized functions from categories like:
  - Physical Constants (alpha_particle, speed_of_light, etc.)
  - Statistics and probability functions
  - Advanced trigonometry and hyperbolic functions
  - Matrix and vector operations
  - Number theory (prime functions, advanced arithmetic)
  - Calculus (derivatives, integrals)
  - Special mathematical functions

### Filtering
- Only active functions and variables are shown (using libqalculate's `isActive()`)
- Completions are filtered by prefix matching as you type
- Case-insensitive matching for better usability

## Help System
The application includes a comprehensive help system accessible via Ctrl+H.

### Features
- **Scrollable content**: Help text automatically scrolls when content exceeds available height
- **Adaptive sizing**: Help popup adjusts to terminal size with sensible constraints
- **Visual indicators**: Title shows scroll status and available actions
- **External content**: Help text loaded from `help.txt` file at compile time
- **Overlay design**: Help popup centers over the main interface without disrupting state

### Navigation
- **Up/Down arrows**: Scroll content line by line
- **Page Up/Down**: Scroll content by half page increments
- **Mouse wheel**: Scroll content up/down (3 lines per scroll)
- **Esc**: Close help and return to calculator
- **Dynamic feedback**: Title shows "(↑↓ to scroll, Esc to close)" when scrollable

### Content Areas
- Overview and basic usage instructions
- Mathematical expression examples
- Complete keyboard shortcut reference
- Feature explanations and tips

## Undo/Redo System
The application provides comprehensive undo/redo functionality to recover from mistakes and experiment safely.

### Functionality
- **50-level history**: Maintains up to 50 previous states for undo
- **State preservation**: Saves input text, results, cursor positions, and focus state
- **Smart triggering**: Automatically saves state before significant changes:
  - Line deletion (Ctrl+D, Backspace on empty line)
  - New sheet creation (Ctrl+N)
  - New line creation (Enter)
  - Multi-line paste operations
  - Template insertion (Ctrl+T)
  - Result click insertions (clicking results to insert ans references)
  - Symbol insertions (Ctrl+P for π, Ctrl+R for √, Ctrl+A for ans)
  - Auto-completion insertions (Tab/Enter on completions)

### Key Bindings
- **Ctrl+Z**: Undo last action
- **Ctrl+Y**: Redo last undone action

### Behavior
- **Undo stack**: Each action that modifies content saves the previous state
- **Redo stack**: Undoing an action enables redo; new actions clear the redo stack
- **Cursor restoration**: Undo/redo preserves exact cursor positions and focus
- **Results restoration**: Calculated results are restored along with input text

## Session Persistence
Sheets are stored as named sessions so work survives quitting or a crash.

### Storage
- Location: `$XDG_DATA_HOME/nasc-tui/sessions/<name>.nasc` (falls back to `~/.local/share`)
- Format: plain text, one input line per file line — nothing else
- Blank lines are preserved on load and save; `ans1`, `ans2`, ... address lines by
  position, so dropping a blank line would break every reference below it
- Results are not stored; they are recalculated by the normal calculation chain on load
- Metadata comes from the filesystem: name = file name, "last used" = modification time
- Writes are atomic (temp file + rename), so a crash mid-write cannot truncate a session
- An all-blank session is never created on disk

### Command Line
- `nasc`: resume the most recently used session
- `nasc -s`: open the session picker on startup (skipped when no sessions exist yet)
- `nasc -n`: start a new session named after today's date
- `nasc <name>`: open, or create, a session by name
- Piped input (`echo "2+2" | nasc`) stays a throwaway scratch sheet and is never saved

### Autosave
- A `tea.Tick` fires every 30 seconds; the sheet is written only when its text changed
- Writing happens in a `tea.Cmd`, never in `Update()`
- Quitting (Esc / Ctrl+C) and switching sessions save synchronously, because a command
  would race with `tea.Quit`

### Session Picker (Ctrl+O)
Fuzzy filter over the session list, most recently used first.

- **Type**: filter (subsequence match, bonuses for prefix, word starts and unbroken runs)
- **Up/Down**: move the selection
- **Enter**: save the current session and switch to the selected one
- **Ctrl+N**: new session (name prompt, prefilled with today's date)
- **Ctrl+R**: rename the selected session
- **Ctrl+U**: duplicate the selected session (`<name> copy`, `<name> copy 2`, ...)
- **Ctrl+D**: delete the selected session, confirmed with `y`
- **Esc**: close the picker, or leave the name prompt

Deleting the open session falls back to the next most recent one, or a new empty session.
The name of the open session is shown in the top border of the input pane.

## Mouse Actions
- **Click result**: Insert corresponding `ans<N>` reference at cursor
- **Click input line**: Focus that line and position cursor at click location
- **Click gutter**: Focus line with cursor at end (when clicking line numbers)
- **Mouse wheel in help**: Scroll help content up/down (3 lines per scroll)