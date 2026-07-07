# 🍅 TaskMax

A cosy, polished desktop **task manager with a built-in Pomodoro timer**, built with **Go + Wails v2 + Svelte**. Runs fully offline on a local SQLite database out of the box, and can connect to an external PostgreSQL or MySQL database via a simple config file.

---

## ✨ Features

- **Task management** — create, edit, delete, and organise tasks with titles, descriptions, priorities, tags, and due dates.
- **Drag-to-reorder** task list with inline status toggling (todo → in progress → done).
- **Pomodoro timer** — a large circular countdown that tracks focused work against a selected task.
- **Automatic cycle** — work → short break → work → … → long break, following your configured cadence.
- **Desktop notifications** when a session completes, plus an in-app event so the UI auto-advances.
- **Focus statistics** — today's completed sessions, total focus time, and per-task session history.
- **Three cosy themes** — `cosy`, `dark`, and `light`, swappable live from Settings.
- **Pluggable database** — SQLite by default; switch to PostgreSQL or MySQL with a "Test connection" button in Settings.

---

## 🧱 Tech Stack

| Layer | Technology |
|---|---|
| Desktop framework | [Wails v2](https://wails.io) |
| Frontend | Svelte 4 + Vite 5 |
| Language | Go 1.22+ |
| ORM | [GORM](https://gorm.io) |
| Local DB | SQLite |
| External DB | PostgreSQL / MySQL |
| Notifications | [beeep](https://github.com/gen2brain/beeep) |
| Config | [Viper](https://github.com/spf13/viper) (`config.yaml`) |

---

## 📁 Project Structure

```
TaskMax/
├── main.go                     # Wails entry point — embeds frontend, opens DB, runs the app
├── app.go                      # App struct: methods bound to the Svelte frontend
├── config.yaml                 # User config (DB, Pomodoro durations, theme)
├── wails.json                  # Wails project config
├── Makefile                    # setup / dev / build / check targets
├── go.mod / go.sum
│
├── internal/
│   ├── config/config.go        # Load & save config with Viper (creates defaults on first run)
│   ├── db/db.go                # DB factory: returns a *gorm.DB for sqlite/postgres/mysql
│   ├── models/
│   │   ├── task.go             # Task model
│   │   └── pomodoro.go         # PomodoroSession model
│   └── services/
│       ├── task_service.go     # Task CRUD + reordering
│       └── pomodoro_service.go # Timer engine (goroutine + ticker + context), stats
│
└── frontend/                   # Svelte + Vite single-page app
    ├── index.html
    ├── vite.config.js
    ├── package.json
    ├── src/
    │   ├── main.js
    │   ├── style.css           # Theme tokens (CSS custom properties)
    │   ├── App.svelte          # Three-panel layout
    │   ├── lib/
    │   │   ├── TaskList.svelte
    │   │   ├── TaskForm.svelte
    │   │   ├── PomodoroTimer.svelte
    │   │   ├── Settings.svelte
    │   │   └── StatsPanel.svelte
    │   └── stores/
    │       ├── tasks.js
    │       └── timer.js
    └── wailsjs/                # Auto-generated Go↔JS bindings (regenerated on build)
```

---

## 🚀 Getting Started

### Prerequisites

- **Go** 1.22+
- **Node.js** 18+ and npm
- **Wails CLI** v2 — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Platform webview:
  - **Windows** — WebView2 (preinstalled on Windows 10/11)
  - **macOS** — WebKit (built in)
  - **Linux / WSL** — `libgtk-3-dev` and `libwebkit2gtk-4.0-dev` (see below)

### Install dependencies

```bash
make setup          # installs the Wails CLI, Go modules, and npm packages
```

### Run in development (hot-reload)

```bash
make dev
```

This opens the app window and live-reloads on changes to Go or Svelte code.

### Build a distributable binary

```bash
make build          # output: ./build/bin/TaskMax (.exe on Windows, .app on macOS)
make run            # build, then launch the binary
```

---

## 🐧 Linux / WSL notes

On Linux the Wails webview needs GTK + WebKit. Install them once:

```bash
make system-deps    # sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev …
```

If your distro ships **webkit2gtk 4.0** (e.g. Ubuntu 22.04), the build needs a matching tag. The Makefile adds it automatically on Linux:

```bash
wails build -tags webkit2_40      # done for you by `make build` / `make dev`
```

If you have **webkit2gtk 4.1**, override it:

```bash
make build WAILS_TAGS=webkit2_41
```

> **WSL tip:** running from the Windows-mounted `/mnt/c/...` path works but is slow (disk I/O + hot-reload). For a snappier dev loop, clone into the WSL-native filesystem (e.g. `~/TaskMax`). The GUI window is provided by **WSLg** on Windows 11.

---

## ⚙️ Configuration

Settings live in `config.yaml` (created automatically on first run) and are also editable from the in-app **Settings** panel.

```yaml
database:
  type: sqlite          # sqlite | postgres | mysql
  dsn: tasks.db         # file path for sqlite, connection string for others

pomodoro:
  work_duration: 25     # minutes
  short_break: 5
  long_break: 15
  sessions_before_long: 4

app:
  theme: cosy           # cosy | dark | light
  minimize_to_tray: true
```

### External databases

Switch `database.type` and provide a DSN:

- **PostgreSQL**
  ```yaml
  database:
    type: postgres
    dsn: "host=localhost user=taskmax password=secret dbname=taskmax port=5432 sslmode=disable"
  ```
- **MySQL**
  ```yaml
  database:
    type: mysql
    dsn: "taskmax:secret@tcp(127.0.0.1:3306)/taskmax?charset=utf8mb4&parseTime=True&loc=Local"
  ```

Use the **Test connection** button in Settings to validate a DSN before saving. Changing the database driver takes effect after an app restart; tables are auto-migrated on startup.

---

## 🛠️ Makefile Targets

| Target | Description |
|---|---|
| `make setup` | Install Wails CLI + Go modules + npm packages |
| `make system-deps` | *(Linux/WSL)* Install GTK + WebKit libraries |
| `make dev` | Run with hot-reload |
| `make build` | Build a production desktop binary |
| `make build-debug` | Build with the debug console + devtools |
| `make run` | Build, then launch the binary |
| `make frontend` / `make frontend-dev` | Build / dev-serve the Svelte UI only |
| `make check` | `go vet` + backend build + `go test` |
| `make doctor` | Wails toolchain health check |
| `make clean` / `make clean-all` | Remove build artifacts / node_modules |

---

## 🏗️ Architecture

- **`main.go`** loads config, opens the database via the `db` factory, auto-migrates the models, then hands an `*App` to Wails.
- **`app.go`** is the binding surface. Every exported method is callable from Svelte as `window.go.main.App.MethodName()` (Wails generates typed wrappers in `frontend/wailsjs`).
- **Services** hold the logic:
  - `TaskService` — CRUD and drag-reorder persistence.
  - `PomodoroService` — owns a countdown goroutine driven by a `time.Ticker`, cancellable via `context`. On completion it marks the session done, fires a desktop notification, emits the `pomodoro:complete` Wails event, and computes the next session in the cycle.
- **Frontend** polls `GetTimerState()` once per second (the authoritative countdown lives in Go) and listens for `pomodoro:complete` to auto-advance and refresh stats.

### Bound methods

**Tasks:** `GetAllTasks`, `GetTasksByStatus`, `CreateTask`, `UpdateTask`, `DeleteTask`, `ReorderTasks`
**Pomodoro:** `StartPomodoro`, `StopPomodoro`, `GetTimerState`, `GetSessionsForTask`, `GetTodayStats`
**Config:** `GetConfig`, `SaveConfig`, `TestConnection`

### Data models

```go
type Task struct {
    gorm.Model
    Title         string
    Description   string
    Priority      string      // "low" | "medium" | "high"
    Status        string      // "todo" | "in_progress" | "done"
    Tags          string      // comma-separated
    DueDate       *time.Time
    PomodoroCount int
    Position      int         // manual sort order
}

type PomodoroSession struct {
    gorm.Model
    TaskID      uint
    Type        string        // "work" | "short_break" | "long_break"
    Duration    int           // minutes
    Completed   bool
    StartedAt   time.Time
    CompletedAt *time.Time
}
```

---

## 📄 License

MIT — see [LICENSE](LICENSE).
