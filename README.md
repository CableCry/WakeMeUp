<div align="center">


# WakeMeUp

> _Wake Me Up inside... your terminal!_

<img src="assets/emo-emoji.png" alt="WakeMeUp logo" width="160" />
</div>

---

## ✨ Overview

I recently started to do some selfhosting which led me to using WOL. The problem I ran into was that most of the apps were ancient feeling and had entire GUIs. I dont need a GUI (I say that as I gave this app a TUI), what I need to to stay in the terminal. So I made this; it uses charms BubbleTea as the TUI and some of the basic UDP stuff to send magic packets.

---

## 🚀 Features

- ⚡ **Wake any saved device** by pressing enter 0o0
- ➕ **Add, edit, and remove** devices, just by typing!!!!
- 💾 **Persistent storage** in super high tech and ground breaking JSON
- 🎯 **Targeted or broadcast** the magical packet that turns computers on (͡o‿O͡)
- 🎨 **Adaptive theme** that has a light or dark mode!!! take that react devs!
- 🖥️ **Scriptable CLI** for if you are insane and dont want the amazing UI

---

## 📦 Installation

### From source

```sh
go install github.com/CableCry/WakeMeUp@latest
```

Or build it locally:

```sh
git clone https://github.com/CableCry/WakeMeUp.git
cd WakeMeUp
go build -o wake .
```

### Pre-built binaries

Download the latest binary for your platform from the [Releases](https://github.com/CableCry/WakeMeUp/releases) page.

> **Requirements:** Go 1.26+ to build from source.

---

## 🧑‍💻 Usage

Launch the interactive manager:

```sh
wake
```

### Keybindings

| Key            | Action               |
| -------------- | -------------------- |
| `↑` / `↓`      | Move between devices |
| `enter`        | Wake selected device |
| `a`            | Add a device         |
| `e`            | Edit selected device |
| `d`            | Remove selected device |
| `/`            | Filter devices       |
| `q`            | Quit                 |

### Command line

For scripting or quick access without the UI:

```sh
wake list              # List saved devices
wake wake <name>...    # Wake one or more devices by name
wake wake all          # Wake every saved device
wake version           # Print the version
wake help              # Show help
```

---

## ⚙️ Configuration

Devices are stored as JSON at:

| OS      | Path                                          |
| ------- | --------------------------------------------- |
| Windows | `%AppData%\wakemeup\devices.json`             |
| macOS   | `~/Library/Application Support/wakemeup/devices.json` |
| Linux   | `~/.config/wakemeup/devices.json`             |

---

## 🛠️ Built With

- [Go](https://go.dev)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — styling

---

## 🤝 Contributing

I dont really have any requirements, just make a PR and give a decent description at least.

---

## 📄 License

MIT so do what you want

---

<div align="center">
<sub>"Made with an odd blue gopher"</sub>
</div>
