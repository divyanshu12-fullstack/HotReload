# HotReload Assignment Walkthrough Script

> **Implementation update:** After the initial review, the remaining engineering gaps were addressed in the codebase. The watcher now removes deleted directories and ignores build artifacts explicitly, rebuild orchestration is serialized, the builder returns typed errors, Windows process-tree termination is stronger, and the demo/test flow is more Windows-friendly. The main remaining items are **manual submission tasks** such as creating the private GitHub repository, sharing access, and recording the Loom video.

## 1. Introduction

This project is a custom Go command-line tool named `hotreload`.

The purpose of the tool is to improve the local development workflow for Go applications. Normally, when a developer changes source code, they need to manually:

1. stop the running server,
2. rebuild the binary,
3. run the server again.

That repeated cycle is slow and distracting.

The goal of this assignment is to automate that process.

With this tool, a developer can run a command like:

```bash
hotreload --root ./myproject --build "go build -o ./bin/server ./cmd/server" --exec "./bin/server"
```

After that, the tool keeps watching the project directory. Whenever code changes, it automatically rebuilds the project and restarts the server.

---

## 2. What problem are we solving?

We are solving a real developer productivity problem.

In a normal development loop, even a tiny code change may require a rebuild and restart. This becomes expensive when done many times a day.

However, this is not just about watching files. A good hot reload tool must also handle practical issues such as:

- editors generating multiple filesystem events for one save,
- nested folders inside a project,
- new folders being created while the tool is already running,
- build failures,
- server startup failures,
- processes that do not shut down cleanly,
- real-time server logs,
- long-running usage over hours.

So the assignment is really about building a small but production-style orchestration tool.

---

## 3. High-level approach

My approach to the assignment is to divide the system into a clear pipeline.

The pipeline has four main stages:

1. **Watcher** – detects filesystem changes.
2. **Debouncer** – compresses noisy bursts of file events into one rebuild trigger.
3. **Builder** – runs the build command and supports cancellation.
4. **Runner** – starts and stops the server process.

Then `main.go` acts as the coordinator that wires everything together.

This separation makes the project easier to understand, test, and maintain.

---

## 4. Project structure and why it is organized this way

The project is split into focused packages:

- `main.go` – CLI entrypoint and orchestration logic
- `watcher/watcher.go` – recursive file watching and filtering
- `debouncer/debouncer.go` – debounce logic for noisy file events
- `builder/builder.go` – build execution
- `runner/runner.go` – server lifecycle management
- `runner/syscall_windows.go` – Windows-specific process stop behavior
- `runner/syscall_unix.go` – Unix-specific process group handling
- `testserver/main.go` – demo server used to prove hot reload behavior
- `README.md` – project explanation
- `Makefile` – helper commands for build, run, demo, and test

This structure follows the assignment well because each package owns one responsibility.

---

## 5. How the tool works from start to finish

When the user runs the CLI:

1. `main.go` parses `--root`, `--build`, and `--exec`.
2. It validates that all required flags are present.
3. It creates a watcher for the root directory.
4. The watcher recursively registers all subdirectories.
5. Raw file events are forwarded into the debouncer.
6. The debouncer waits briefly to collapse bursts of events.
7. When the debounce window expires, `main.go` triggers a rebuild.
8. Before rebuilding, it stops any running server.
9. It cancels any previous in-progress build.
10. It runs the latest build command.
11. If the build succeeds, it starts the server.
12. If the build fails, it logs the error and waits for the next file change.
13. If the user presses `Ctrl+C`, the tool stops the server and exits.

---

## 6. Detailed walkthrough of each component

### 6.1 `main.go`

`main.go` is the control center.

Its responsibilities are:

- parse CLI flags,
- validate input,
- initialize the watcher,
- initialize the debouncer,
- initialize the builder,
- initialize the runner,
- trigger the first build immediately,
- listen for file-change triggers,
- listen for OS shutdown signals.

### Why the immediate build matters

The assignment explicitly requires the first build to happen immediately on startup.

That means the developer does not need to touch a file before the tool becomes useful.

### Why cancellation matters

If a build is already running and a new file change happens, the previous build is no longer useful because it represents stale code.

So `main.go` stores a cancel function for the current build context. When a new change comes in, it cancels the previous build and starts a fresh one.

That design attempts to ensure only the latest state matters.

---

### 6.2 `watcher/watcher.go`

The watcher uses `fsnotify`, which is allowed by the assignment.

Important behavior implemented here:

- recursive directory registration using `filepath.WalkDir`,
- filtering of ignored directories like `.git`, `node_modules`, `vendor`, `bin`, `dist`, and `build`,
- filtering of temporary or noisy files such as swap files and editor artifacts,
- forwarding only relevant event types: `Write`, `Create`, `Remove`, and `Rename`,
- logging watcher errors without crashing,
- detecting newly created directories and adding them to the watcher,
- warning when the number of watched directories exceeds `1000`.

### Why recursive walking is necessary

`fsnotify` does not automatically watch subdirectories.

So to support real projects, the tool must manually walk the tree and register each directory.

### Why filtering matters

Without filtering, the tool would rebuild too often.

For example:

- `.git` changes are not relevant,
- `node_modules` is noisy and unrelated for a Go hot reload tool,
- build outputs can cause rebuild loops,
- temporary files can generate false rebuilds.

So filtering is critical for correctness and responsiveness.

---

### 6.3 `debouncer/debouncer.go`

The debouncer exists because one human save operation often creates several filesystem events.

Examples:

- write,
- rename,
- temp file replacement,
- remove,
- editor-specific save patterns.

If the tool rebuilt on every raw event, it would rebuild several times for one real code change.

### Debounce strategy

The debouncer:

- accepts input events on a channel,
- starts or resets a timer on each new event,
- emits a single `struct{}{}` trigger only after the timer expires with no new events.

The configured delay is `150ms`, which is exactly aligned with the assignment requirement.

### Why this design helps

It keeps the tool responsive while preventing redundant builds.

That means the developer sees one rebuild for a burst of edits instead of many unnecessary rebuilds.

---

### 6.4 `builder/builder.go`

The builder package is responsible for compiling the target project.

Its core behavior:

- receives the build command as a string,
- splits it into command and args,
- runs it through `exec.CommandContext`,
- streams stdout and stderr directly to the terminal,
- respects cancellation via context,
- logs success or failure with `slog`.

### Why `exec.CommandContext` matters

This is important because the assignment requires the previous build to be discarded when a new change arrives.

With a cancelable context, the running build can be interrupted.

### Why direct output streaming matters

The assignment explicitly asks for real-time logs rather than buffered output.

By wiring `cmd.Stdout` and `cmd.Stderr` directly to the terminal, users can see build errors immediately.

---

### 6.5 `runner/runner.go`

The runner package manages the lifecycle of the application process.

Its responsibilities are:

- parse the exec command,
- start the server process,
- stream logs directly to the terminal,
- stop the previous server when rebuilding,
- detect immediate crashes,
- apply backoff to avoid crash loops.

### Crash loop protection

If the server exits within one second of startup, the code logs a warning and introduces a three-second pause.

This is intended to avoid a tight restart loop that could waste CPU and flood logs.

### Platform-specific files

The project separates Windows and Unix process behavior:

- `syscall_unix.go` uses process groups and signals,
- `syscall_windows.go` uses `taskkill`-based process-tree termination and starts processes in their own group.

This separation is important because process management is OS-specific, and the finalized version improves child-process cleanup on both platforms.

---

## 7. Demo server

The `testserver` package exists to demonstrate the tool in a controlled way.

It runs a very simple HTTP server on port `8080` and returns a version string in the response.

This makes the demo easy:

1. start `hotreload`,
2. open the server in a browser,
3. change `version` from `v1` to `v2`,
4. save the file,
5. watch the rebuild happen,
6. refresh the browser and verify the new output.

That is a clean demonstration of the assignment’s intended behavior.

---

## 8. Testing strategy

The project includes tests for tricky parts:

- `debouncer/debouncer_test.go`
- `builder/builder_test.go`
- `runner/runner_test.go`
- `watcher/watcher_test.go`

### Debouncer tests

The debouncer tests verify:

- many rapid events collapse into one output trigger,
- separate events still produce separate triggers.

These tests are important because debounce behavior is easy to get subtly wrong.

### Builder tests

The builder tests verify:

- canceled builds return the typed cancellation error,
- true command failures return a typed build-failure error.

### Runner tests

The runner test verifies the process can start and then stop cleanly.

This gives confidence that lifecycle control and shutdown behavior are working.

### Watcher tests

The watcher tests verify:

- explicitly ignored output paths do not trigger rebuilds,
- dynamically created directories are added recursively,
- deleted directories are removed from the internal watch registry.

---

## 9. Requirement-by-requirement compliance review

This section evaluates the current implementation against the assignment file.

## Core requirements

### Requirement: Watch for file changes
**Status: Met**

The watcher listens for filesystem changes and emits events for relevant operations.

### Requirement: Rebuild automatically
**Status: Met**

The main loop triggers the builder when debounced changes arrive.

### Requirement: Restart the server automatically
**Status: Met**

After a successful build, the runner starts the server again.

### Requirement: Trigger the first build immediately on startup
**Status: Met**

`main.go` calls the rebuild logic immediately after initialization.

### Requirement: Work for a simple Go project
**Status: Met**

The tool builds successfully, watches a Go project, performs an immediate build, and restarts the server automatically.

---

## Performance expectations

### Requirement: Restart within about 2 seconds after save
**Status: Met in intended usage**

The debounce window is `150ms`, the rebuild flow is serialized, and the tool is designed to restart quickly after changes. Exact timing still depends on the project’s own build speed, but the implementation is aligned with the assignment target.

### Requirement: Avoid multiple rebuilds for rapid save bursts
**Status: Met**

The debouncer is implemented and tested.

---

## Change handling

### Requirement: If a rebuild is already in progress and a new change occurs, discard the old build and only build the latest state
**Status: Met**

The rebuild pipeline is serialized and stale builds are canceled. Pending rebuild requests are folded into the latest state instead of allowing overlapping rebuilds to race.

---

## Process management

### Requirement: Ensure the previous server process is fully terminated
**Status: Met**

The runner stops the previous process before restarting and waits for shutdown before continuing.

### Requirement: Kill all child processes, not just the parent
**Status: Met**

Unix uses process-group signaling and Windows uses process-tree termination, so child processes are handled as part of shutdown.

### Requirement: Some processes do not shut down nicely; handle stubborn processes
**Status: Met**

The runner attempts graceful termination first and then falls back to force-kill if the process does not exit in time.

### Requirement: Ensure that killing a process frees held resources
**Status: Met**

The runner tracks process completion and waits for shutdown so the process lifecycle is cleaned up correctly.

---

## Logging

### Requirement: Logs should stream in real time
**Status: Met**

Both builder and runner pipe stdout and stderr directly to the terminal.

### Requirement: Use `log/slog`
**Status: Met for the tool itself**

Operational logging in the hotreload tool uses `slog`. The demo server remains a small standalone sample app and does not affect the CLI’s logging design.

---

## Bonus requirements

### Requirement: Support nested project structure
**Status: Met**

The watcher recursively adds directories.

### Requirement: Detect and watch new folders created while running
**Status: Met**

New directories are detected and recursively added to the watcher.

### Requirement: Handle deleted folders gracefully
**Status: Met**

Deleted directories are removed from the watcher’s internal registry.

### Requirement: Avoid rapid restart loops if server crashes immediately
**Status: Met**

Crash-loop detection and backoff are implemented without blocking the rest of the runner lifecycle incorrectly.

### Requirement: OS watch limit awareness
**Status: Met**

The watcher logs a warning if the directory count exceeds `1000`.

### Requirement: Ignore irrelevant paths and temporary files
**Status: Met**

The watcher ignores irrelevant folders, temporary files, and the explicit build/output path to prevent rebuild loops.

---

## Tests requirement

### Requirement: Tests for tricky components
**Status: Met**

There are tests for debouncing, runner lifecycle behavior, builder error typing, and watcher edge cases.

---

## Submission extras

### Requirement: README
**Status: Met**

There is a README with installation, usage, architecture, and limitations.

### Requirement: Sample `testserver/`
**Status: Met**

A demo server is included.

### Requirement: Build/run helper such as Makefile or script
**Status: Met**

The `Makefile` supports platform-aware binary suffixes and uses a workspace-local Go temp directory for more reliable test execution on Windows.

### Requirement: Loom explanation video
**Status: Not met yet**

No video asset is present in the repository.

### Requirement: GitHub repository and commit history
**Status: Not verifiable from local code**

This cannot be confirmed from the workspace alone.

---

## 10. Honest summary of the current status

### What is working well

The project already demonstrates the main architecture the assignment wants:

- it is split into sensible packages,
- it watches directories recursively,
- it debounces events,
- it cancels stale builds,
- it rebuilds and restarts automatically,
- it includes a demo server,
- it has some tests,
- it uses `slog` for most operational logs.

### What is still missing or weak

At this point, the remaining gaps are mostly outside the codebase itself:

1. **The Loom video still needs to be recorded.**
2. **The private GitHub repository still needs to be created and shared.**
3. **Commit history quality depends on how the repository is finalized and pushed.**

### Final assessment

If the question is: **“Does this project now satisfy the engineering requirements in the assignment file?”**

The honest answer is:

**Yes, for the codebase itself, largely yes.**

The main engineering requirements have now been implemented in the repository.

If the question is: **“Is the whole submission 100% complete?”**

Then the answer is:

**Not yet, because a few submission items are manual tasks.**

Those remaining manual tasks are:

- create the private GitHub repository,
- push the final code,
- share access with the requested email,
- record and upload the Loom walkthrough.

---

## 11. How I would explain my approach in an interview or Loom video

A strong explanation would be:

> I approached the assignment as a pipeline problem. Instead of putting everything in one file, I broke the system into four clear stages: watching, debouncing, building, and running. The watcher collects raw file events. The debouncer reduces noisy duplicate events. The builder compiles the latest state and supports cancellation, which is important when multiple file changes happen quickly. The runner manages the application lifecycle by stopping the old server and starting the new one while streaming logs in real time. Then `main.go` coordinates those packages and handles shutdown signals.

Then I would explain each package one by one and show how a file save moves through the pipeline.

---

## 12. Demo walkthrough script

If I were recording a video, I would present it like this:

1. Introduce the problem: manual rebuild and restart is slow.
2. Show the CLI command with `--root`, `--build`, and `--exec`.
3. Open the project structure and explain the package layout.
4. Open `main.go` and explain orchestration.
5. Open `watcher/watcher.go` and explain recursive directory watching and filtering.
6. Open `debouncer/debouncer.go` and explain why save events are noisy.
7. Open `builder/builder.go` and explain cancellation with context.
8. Open `runner/runner.go` and explain lifecycle management and crash-loop protection.
9. Open `testserver/main.go` and show the version constant.
10. Run the tool.
11. Change `version` from `v1` to `v2`.
12. Save the file and show rebuild + restart.
13. Refresh the browser and show updated output.
14. Conclude with the current strengths and the remaining manual submission tasks.

---

## 13. Closing statement

This project demonstrates the main architecture and engineering direction expected by the assignment.

It now covers the core hot reload workflow and the major robustness requirements from the assignment.

The remaining next steps are mostly submission-related:

- create the private GitHub repository,
- push the final code and preserve a clean commit history,
- share access with the requested email,
- record the Loom walkthrough,
- submit the assignment form.

That would complete the full assignment package.
