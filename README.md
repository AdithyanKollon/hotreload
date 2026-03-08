# hotreload

A production-grade hot-reload CLI for Go (and any compiled language). Watches a project directory for file changes, automatically rebuilds, and restarts your server — all in under 2 seconds.

```
hotreload --root ./myproject --build "go build -o ./bin/server ./cmd/server" --exec "./bin/server"
```

---

## Features

| Feature | Details |
|---|---|
| **Fast** | File → rebuilt + running server in < 2 seconds |
| **Smart debouncing** | 300 ms quiet-period coalesces editor save storms |
| **Cancellable builds** | New change mid-build? The old build is killed immediately |
| **Process-tree killing** | Kills the full process group, not just the parent PID |
| **Crash-loop protection** | Exponential backoff if server dies within 2 seconds of starting |
| **Dynamic dir watching** | New subdirectories are auto-watched as they appear |
| **File filtering** | Ignores `.git`, `node_modules`, editor swap files, build artifacts |
| **inotify limit check** | Warns on Linux if `max_user_watches` is too low |
| **Structured logging** | Uses `log/slog` from the Go standard library |
| **Real-time logs** | Server stdout/stderr streamed immediately, not buffered |

---

## Installation

```bash
# Clone
git clone https://github.com/yourusername/hotreload
cd hotreload

# Build
make build          # produces ./bin/hotreload

# Or install globally
make install        # installs to $(GOPATH)/bin
```

**Requirements:** Go 1.22+

---

## Usage

```
hotreload [flags]

Flags:
  --root       <dir>     Directory to watch (default: ".")
  --build      <cmd>     Command to build the project (required)
  --exec       <cmd>     Command to run after a successful build (required)
  --log-level  <level>   debug | info | warn | error (default: info)
  --version              Print version and exit
```

### Examples

**Go project:**
```bash
hotreload \
  --root ./myproject \
  --build "go build -o ./bin/server ./cmd/server" \
  --exec "./bin/server"
```

**With build flags:**
```bash
hotreload \
  --root . \
  --build 'go build -ldflags "-s -w" -o ./bin/api .' \
  --exec "./bin/api --port 8080"
```

---

## Demo

Run the included test server:

```bash
make demo
```

Then:
1. Open `http://localhost:8080` — you'll see `Hello from testserver v1 👋`
2. Edit `testserver/main.go` — change the `message` constant
3. Save the file
4. Within ~1 second, the server restarts with your new message

---

## Architecture

```
main.go          CLI flags, signal handling, wires components together
│
├── watcher/     fsnotify wrapper
│   └─ Recursive directory watching
│   └─ Dynamic new-directory detection
│   └─ inotify watch limit warning
│
├── filter/      Path-based ignore rules
│   └─ Glob pattern matching on every path component
│
├── debouncer/   Rapid event coalescing
│   └─ 300 ms quiet window before triggering build
│
├── builder/     Build command runner
│   └─ context.Context cancellation of in-flight builds
│   └─ Real-time build log streaming
│
└── runner/      Server process lifecycle
    └─ Process group creation (Setpgid)
    └─ SIGTERM → SIGKILL escalation
    └─ Crash loop detection + exponential backoff
    └─ Real-time server log streaming
```

### Key Design Decisions

**1. Context-based build cancellation**

The builder holds a `context.CancelFunc` for the current build. When a new file event triggers while a build is running, we cancel that context immediately — `exec.CommandContext` propagates the cancellation to the `go build` subprocess. A fresh build then starts from the latest file state.

**2. Process group killing**

When the server starts, it's placed in its own process group via `syscall.SysProcAttr{Setpgid: true}`. On restart, we send `SIGTERM` (then `SIGKILL` after 3 seconds) to `-pgid`, which kills the entire group — including any child processes the server may have spawned.

**3. Crash loop detection**

If the server exits within 2 seconds of starting, it's considered a crash. The runner tracks consecutive crashes and applies exponential backoff (1s → 2s → 4s … → 30s max) before restarting. A healthy run resets the backoff.

**4. Debouncing**

Editors like Vim and VSCode generate multiple events per save (write, rename, chmod). The debouncer accumulates events and fires only after a 300 ms quiet period, preventing multiple redundant builds per save.

**5. Real-time log streaming**

Build output and server logs are streamed line-by-line as they arrive. No buffering means you see `fmt.Println` output immediately even in long-running servers.

---

## Running Tests

```bash
make test          # all tests with race detector
make test-short    # faster, no race detector
```

Tests cover:
- Debouncer timing and cancellation
- File filter glob matching
- Build command string splitting (quoted arguments etc.)

---

## inotify Limits (Linux)

The OS limits how many file watches can be open. If you have a large project, increase the limit:

```bash
echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

hotreload will warn you if the current limit looks too low.

---

## What hotreload does NOT use

- `air` / `realize` / `reflex` — not used, full original implementation
- Any hot-reload framework — only `fsnotify` as an event source, per the assignment rules

---

## License

MIT
