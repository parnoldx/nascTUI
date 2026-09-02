
```
  _   _        _____  _____ 
 | \ | |      / ____|/ ____|
 |  \| | __ _| (___ | |     
 | . ` |/ _` |\___ \| |     
 | |\  | (_| |____) | |____ 
 |_| \_|\__,_|_____/ \_____|

```               
<h3>Do maths like a normal person</h3>


![NASC TUI Demo](demo.gif)

[![Release](https://img.shields.io/github/release/parnoldx/nascTUI.svg)](https://github.com/parnoldx/nascTUI/releases)
[![License: GPL v2](https://img.shields.io/badge/License-GPL%20v2-blue.svg)](https://www.gnu.org/licenses/gpl-2.0)
[![libqalculate](https://img.shields.io/badge/Powered%20by-libqalculate-green)](https://github.com/Qalculate/libqalculate)

## 
NaSC is an app where you do maths like a normal person. It lets you type whatever you want and smartly figures out what is math and spits out an answer on the right pane. Then you can plug those answers in to future equations and if that answer changes, so does the equations it's used in.

**Features:**
- 🧮 Advanced mathematical expressions and functions
- 🔄 Real-time unit conversions
- 📊 Multiple input lines with live results
- 🗂️ Named sessions with autosave, fuzzy switching, rename and duplicate
- ⚡ Instant calculation results
- 🎨 Beautiful terminal UI
- 🔍 Auto-completion for functions and variables
- ↩️ Undo/redo functionality
- 🚀 Fast and lightweight

## Installation

Works on Arch Linux, Omarchy, and other Arch derivatives, plus Debian/Ubuntu, Fedora, and openSUSE.

```bash
bash -c "$(curl -sLo- https://raw.githubusercontent.com/parnoldx/nascTUI/refs/heads/master/install.sh)"
```

Or from a checkout:

```bash
git clone https://github.com/parnoldx/nascTUI.git
cd nascTUI
./install.sh
```

Arch / Omarchy / AUR:

```bash
# official extra packages used at runtime / build time
omarchy pkg add libqalculate go   # or: sudo pacman -S --needed libqalculate go

# AUR package
yay -S nasc-tui
```

## Usage

Simply run the calculator:
```bash
nasc
```

Your sheet is saved automatically as you type and when you quit, so `nasc` always
comes back with the session you last worked on. Sessions are plain text files in
`~/.local/share/nasc-tui/sessions/`.

```bash
nasc                  # resume the last session
nasc -s               # start with the fuzzy session picker
nasc -n               # start a new session
nasc "taxes 2026"     # open (or create) a session by name
nasc --help           # print the help text
```

Press <kbd>Ctrl</kbd>+<kbd>O</kbd> inside the app to switch sessions — type to filter,
<kbd>Enter</kbd> to open, <kbd>Ctrl</kbd>+<kbd>N</kbd> new, <kbd>Ctrl</kbd>+<kbd>R</kbd>
rename, <kbd>Ctrl</kbd>+<kbd>U</kbd> duplicate, <kbd>Ctrl</kbd>+<kbd>D</kbd> delete.

## Contributing

Please feel free to submit a Pull Request. For major changes, open an issue first to discuss it.

## License

This project is licensed under the GPL v2 License - see the [LICENSE](LICENSE) file for details.

---
