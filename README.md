# Linear Clock

A Raspberry Pi LED clock that displays time on a WS281x strip. A web UI lets you configure colors, brightness, and layout; the clock daemon reads a shared config file and stays lightweight for reliable real-time updates.

**Tested on:** Raspberry Pi Zero 2 (32-bit OS required)

---

## Architecture

| Component   | Description |
|------------|-------------|
| **clockd** | Daemon that drives the LED strip. Reads `config.gob`, refreshes at a set interval, and stays minimal for performance. |
| **configd** | HTTP server (port 8080) for the config UI. Serves templates and static files, and writes `config.gob` for clockd to read. |
| **config.gob** | Shared config file (gob format). configd writes it; clockd reads and hot-reloads it. |

Optional:

- **cli-clock** — Local CLI that runs the same clock logic against a mock (terminal) display for development.

---

## Prerequisites

- Raspberry Pi (e.g. Zero 2) running a **32-bit** OS
- WS281x-compatible LED strip
- Go 1.23+ (for building on your dev machine)

---

## Installation

### 1. Prepare the Pi

On the Pi, create the app directory:

```bash
mkdir -p /home/pi/go/github.com/jaredwarren/clock
```

### 2. Build and copy binaries

On your Mac (or other host):

```bash
# From repo root
make build-clockd-arm   # Produces clockd-armv7 (requires Docker + clock/docker builder)
make build-configd-arm # Produces configd-armv7
```

Then copy binaries and assets to the Pi (replace `pi@clock.local` with your Pi user and host):

```bash
scp clockd-armv7 configd-armv7 pi@clock.local:/home/pi/go/github.com/jaredwarren/clock/
scp -r templates public pi@clock.local:/home/pi/go/github.com/jaredwarren/clock/
```

Or use the Makefile (set `HOST` and `USER` as needed):

```bash
make push
```

### 3. Install systemd services

**Config server** (web UI):

```bash
sudo cp config.service /lib/systemd/system/
# or: rsync --rsync-path="sudo rsync" config.service pi@clock.local:/lib/systemd/system/config.service
sudo systemctl daemon-reload
sudo systemctl enable config.service
sudo systemctl start config.service
```

**Clock daemon**:

```bash
sudo cp clock.service /lib/systemd/system/
# or: rsync --rsync-path="sudo rsync" clock.service pi@clock.local:/lib/systemd/system/clock.service
sudo systemctl daemon-reload
sudo systemctl enable clock.service
sudo systemctl start clock.service
```

### 4. Open the config UI

In a browser: **http://clock.local:8080** (or your Pi’s hostname/IP).

---

## Building

### clockd (ARM, cross-compile via Docker)

clockd depends on the rpi_ws281x C library, so it is built in a Docker image that provides it.

**One-time: build the builder image** (from repo root):

```bash
make -C clock build-image
```

**Build the ARM binary** (from repo root):

```bash
make build-clockd-arm
```

This produces `clockd-armv7` in the repo root. Alternatively, from the `clock/` directory: `make build`.

### configd (ARM, pure Go)

No C deps; cross-compile from your machine:

```bash
make build-configd-arm
# or:
GOOS=linux GOARCH=arm GOARM=7 go build -o configd-armv7 -v ./cmd/configd
```

### Local / development

- **configd** (run locally): `go run ./cmd/configd` (serve from repo root so `templates/` and `public/` resolve).
- **cli-clock** (mock display in terminal): `go run ./cmd/cli-clock`.

### Verify before deploy

Run the pre-deploy verification target:

```bash
make verify
```

`make verify` runs selected package tests, a template parse smoke test, and `go vet` on the core runtime packages.

### Release package (`dist/`)

Build ARM binaries, package runtime assets, and generate checksums:

```bash
make release-dist
```

This creates:

- `dist/clockd-armv7`
- `dist/configd-armv7`
- `dist/clock.service`
- `dist/config.service`
- `dist/templates/`
- `dist/public/`
- `dist/SHA256SUMS` (SHA256 for both binaries)

---

## Updating a running Pi

**Option A — deploy script (recommended):** From the repo root, run the script for an interactive menu (what to deploy, whether to build, whether to restart):

```bash
./scripts/deploy.sh              # interactive menu
./scripts/deploy.sh both         # both clockd and configd (no menu)
./scripts/deploy.sh clock        # clockd only
./scripts/deploy.sh config       # configd only
```

Or via Make (uses `USER` and `HOST` from the Makefile, default `pi@clock.local`):

```bash
make deploy         # both
make deploy-clock   # clockd only
make deploy-config  # configd only
```

**Option B — manual:**  
1. **On Pi:** stop the service you’re updating (e.g. `sudo systemctl stop clock.service`).  
2. **On Mac:** build, then copy the new binary (and any changed `templates/` or `public/`) to the Pi.  
3. **On Pi:** `sudo systemctl start clock.service` (or `config.service`).

---

## Project layout

```
├── cmd/
│   ├── clockd/      # LED clock daemon (uses internal/hw + internal/display)
│   ├── configd/     # Config HTTP server (uses internal/server)
│   └── cli-clock/   # Local mock clock for development
├── internal/
│   ├── config/      # Shared config types and config.gob read/write
│   ├── display/     # Clock face logic (DisplayTime, Clear, Displayer interface)
│   ├── hw/          # WS281x hardware wrapper (used only by clockd)
│   └── server/      # HTTP handlers and templates for configd
├── lib/
│   └── mock/        # Mock display (e.g. terminal output for cli-clock)
├── templates/       # HTML for config UI
├── public/          # Static assets for config UI
├── clock.service    # systemd unit for clockd
├── config.service   # systemd unit for configd
└── clock/           # Docker builder and Makefile for cross-compiling clockd
```

---

## Events

The config UI has an **Events** page (nav link “Events”) where you can add an ordered list of events that override the tick colors (Past, Present, Future, Future B) when active.

- **One-time events** run only on the start date, between the start and end date-time you set.
- **Daily (repeating) events** run every day: the clock compares the current time-of-day to the event’s start/end time-of-day (no midnight wrap in v1).
- **Order matters**: events are applied in list order; later events override earlier ones for any color you set. Leave a color blank (or black) to mean “don’t override” that tick color.
- Events are stored in `config.gob` with the rest of the config, so clockd uses them automatically when it reloads.

Use the table on the Events page to reorder (up/down), delete, or add events with title, start/end date-time, repeat type, and optional color overrides.

---

## TODO / roadmap

- Fix tests (e.g. colors, icon)
- Tune refresh rate / performance
- Ensure brightness config is applied correctly

---

## Reference

- **SSH to Pi:** `ssh pi@clock.local`
- **Restart services:** `sudo systemctl restart clock.service` or `sudo systemctl restart config.service`
- **Config daemon source:** `cmd/configd/main.go`
- **Notes and design ideas:** see `notes.md`
