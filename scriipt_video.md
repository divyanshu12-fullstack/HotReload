# HotReload Demo Video Script

This version is written to sound more natural on camera and fit a tighter 5 to 6 minute submission video.

## Before You Start Recording

Set up your screen like this before you begin:

- VS Code open at the HotReload repository.
- Terminal open in the repository root. On Windows, use a `Command Prompt` terminal for the live demo instead of Git Bash.
- Browser tab ready for `http://localhost:8080`.
- Keep `README.md`, `main.go`, `watcher/watcher.go`, `debouncer/debouncer.go`, `builder/builder.go`, `runner/runner.go`, and `testserver/main.go` easy to switch to.
- In `testserver/main.go`, keep the line `const version = "v1"` visible before the live demo section.
- Make sure nothing else is already using port `8080` before you start the demo.
- On this Windows setup, build the CLI once before recording so you can run the compiled binary instead of `go run`:

```bash
go build -o ./bin/hotreload.exe .
```

If possible, zoom the editor and terminal slightly so logs are readable in the video.

## Video Script

### 0:00 - 0:25

**What to show on screen:**
Open the repository root in VS Code. Show the file tree on the left. Keep the terminal visible at the bottom.

**What to say:**

"Hi, this is my submission for the HotReload backend engineering assignment. I built a custom Go CLI called HotReload that watches a project directory, rebuilds the app when files change, and restarts the server automatically. This project does not use existing hot reload tools like Air or Reflex. The only external watcher dependency I used is fsnotify, which is allowed in the assignment."

### 0:25 - 0:55

**What to show on screen:**
Click into `README.md`. Slowly scroll through the overview, features, usage example, and demo section.

**What to say:**

"The problem I am solving is the usual backend dev loop where even a small code change means stopping the server, rebuilding, and starting it again by hand. This tool automates that. The user passes a root directory, a build command, and an exec command. From there, the CLI watches the project, triggers the initial build right away, debounces noisy file events, cancels stale builds, and restarts the server only if the latest build succeeds."

### 0:55 - 1:20

**What to show on screen:**
Show the repository tree. Briefly click `main.go`, `watcher`, `debouncer`, `builder`, `runner`, and `testserver`.

**What to say:**

"I split the code into small packages with clear responsibilities. The watcher handles recursive directory watching and filtering. The debouncer turns multiple save events into one rebuild trigger. The builder runs the build command with cancellation support. The runner manages the server process. Then main.go ties everything together. I also included a small test server that I will use for the live demo."

### 1:20 - 1:55

**What to show on screen:**
Open `main.go`. Keep the flag parsing, watcher setup, debouncer setup, rebuild queue logic, and initial build section visible as you scroll.

**What to say:**

"This file is the orchestration layer. It parses and validates the flags, starts the watcher and debouncer, creates the builder and runner, and manages the rebuild cycle. One important design choice here is that rebuilds are serialized. If a build is already running and a new change comes in, the older build is cancelled and only the newest state is processed. That avoids overlapping rebuilds and makes sure the tool focuses on the latest code. It also triggers the initial build immediately on startup, which is a required part of the assignment."

### 1:55 - 2:25

**What to show on screen:**
Open `watcher/watcher.go`. Point to the recursive walk, ignore rules, new directory handling, and relevant event filtering.

**What to say:**

"The watcher uses fsnotify, but fsnotify does not automatically watch subdirectories, so I register directories recursively. I also filter out paths that should not trigger rebuilds, like .git, node_modules, vendor, build folders, editor temp files, and generated binaries. It also handles newly created directories while the tool is already running, and removes deleted ones from the watch registry. That makes it more practical for real development usage."

### 2:25 - 2:50

**What to show on screen:**
Open `debouncer/debouncer.go` and then `builder/builder.go`.

**What to say:**

"The debouncer is there because one save action can produce several filesystem events. Instead of rebuilding on every event, I wait for a short quiet window and then trigger one rebuild. The debounce window is 150 milliseconds, which matches the assignment. The builder uses a cancelable context, so if a newer change comes in, the stale build can be stopped. Build logs are streamed directly to the terminal so errors are visible immediately."

### 2:50 - 3:15

**What to show on screen:**
Open `runner/runner.go`. Briefly point to `Start`, `Stop`, and the crash-loop protection logic.

**What to say:**

"The runner manages the application process itself. It starts the server, streams stdout and stderr in real time, stops the previous process before a rebuild, and force-kills the process tree if a normal stop does not finish in time. I also added crash-loop protection, so if the server exits almost immediately after startup, the tool backs off instead of restarting aggressively."

### 3:15 - 3:25

**What to show on screen:**
Switch to the terminal at the repository root.

**What to say:**

"Now I will run the live demo using the sample test server in the repository. I am using a standard Windows command prompt here so the compiled executable runs directly. This will show the full flow from initial build to automatic rebuild and restart after a code change."

### 3:25 - 3:55

**What to show on screen:**
Run this command in the terminal:

```bash
.\bin\hotreload.exe --root .\testserver --build "go -C .\testserver build -o ..\bin\testserver.exe ." --exec ".\bin\testserver.exe"
```

Pause for the logs to appear. Make sure the terminal shows the initial build and server startup.

**What to say:**

"I already built the CLI once, and now I am running the compiled HotReload binary. It watches the testserver directory, builds the sample server from its own module, and then runs it. In the terminal, we can see the initial build happen immediately, followed by the server starting. That confirms the startup flow is working before any file change happens."

### 3:55 - 4:10

**What to show on screen:**
Open the browser at `http://localhost:8080`. Show the response text.

**What to say:**

"In the browser, the sample server is responding on port 8080. It returns a version string, which makes the code change very easy to verify. Right now the output shows version v1."

### 4:10 - 4:45

**What to show on screen:**
Go back to VS Code. Open `testserver/main.go`. Change this line:

```go
const version = "v1"
```

to:

```go
const version = "v2"
```

Save the file. Keep the terminal visible so the rebuild logs can be seen.

**What to say:**

"Now I am editing the test server source code. I am changing the version constant from v1 to v2 and saving the file. After the save, the watcher picks up the change, the debouncer reduces noisy events to one rebuild trigger, the old server is stopped, the project is rebuilt, and the server starts again automatically. In the terminal, we can see that rebuild and restart happen without any manual steps."

### 4:45 - 5:00

**What to show on screen:**
Return to the browser and refresh the page.

**What to say:**

"After refreshing the browser, the response now shows version v2. That confirms the tool detected the source code change, rebuilt the application, restarted the running server, and served the updated code. This is the core requirement of the assignment, and the demo shows it working end to end."

### 5:00 - 5:30

**What to show on screen:**
Switch back to VS Code and briefly show `README.md` or the repository tree again.

**What to say:**

"To summarize, this project includes recursive directory watching, file filtering, a 150 millisecond debounce window, cancelable builds, serialized rebuild orchestration, real-time logs, process shutdown with fallback kill behavior, and automatic restart behavior. I also added tests for the watcher, debouncer, builder, and runner packages. Thank you for reviewing my submission."

## Optional Retake Notes

If you want the demo to feel cleaner during recording, use this exact sequence:

1. Start with the repository tree already expanded.
2. Keep the browser tab pinned and ready before you run the demo command.
3. Wait one second after saving `testserver/main.go` so the rebuild logs are clearly visible.
4. Refresh the browser only after the terminal shows that the server has started again.
5. End the video immediately after the final summary sentence to keep the pacing tight.
6. If the server crashes immediately during the demo, first check whether another process is already using port `8080`.

## Delivery Notes

To make this sound natural during recording:

1. Do not read every sentence with the same pace. Slow down slightly during the live demo.
2. When you switch files, pause for a second so the reviewer can see what is on screen.
3. Treat the script as guidance, not something you have to recite perfectly word for word.
4. Keep the live demo section unchanged if you shorten anything further, because that is the strongest proof that the tool works.