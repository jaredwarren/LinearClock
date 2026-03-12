# High-level repository analysis

Summary of potential issues, non-idiomatic patterns, and improvement areas across the clock repo.

---

## Critical / bug risks

### 1. Mutating the global default config (server)

**Where:** `internal/server/handler.go` (Home, UpdateConfig) and `internal/server/test_handler.go` (TestHandler).

**Issue:** When the config file is missing, the code does `c = config.DefaultConfig`. That assigns the **same pointer** as the package-level default. Any later mutation (e.g. in UpdateConfig: `c.Brightness = i`, `c.Tick.PastColor = color`, etc.) **mutates the global `DefaultConfig`**. After the first form submit with a missing config file, every future use of `DefaultConfig` (same process) sees the user’s values instead of the real defaults.

**Fix:** When falling back to default, use a **copy** of the config (e.g. a `Clone()` that returns a new `*Config` with the same values, or a struct literal copy), not the shared pointer.

---

### 2. `panic` in config write path

**Where:** `internal/config/config.go` — `WriteConfig`.

**Issue:** `os.Create(filepath)` failure leads to `panic(err)` instead of returning an error. Unidiomatic and can crash the process (e.g. configd or any tool that writes config).

**Fix:** Return the error: `if err != nil { return fmt.Errorf("create config file: %w", err) }`.

---

### 3. `hexStringToUint32` can panic on short input

**Where:** `internal/server/handler.go` — `hexStringToUint32`.

**Issue:** After `TrimPrefix(hexStr, "#")`, the code does `hexStr[0:2]`, `hexStr[2:4]`, `hexStr[4:6]` without checking length. A short or empty string (e.g. `"#"`, `"ab"`) causes a slice panic.

**Fix:** At the start, require `len(hexStr) >= 6` and return a clear error otherwise.

---

### 4. Events: global mutable slice with no synchronization

**Where:** `internal/server/handler_events.go` — `var Events = []*Event{}`.

**Issue:** `ListEvents` and `UpdateEvents` read and append to the same global slice. Concurrent requests (or even sequential ones with multiple tabs) can cause races. The Go race detector would flag this.

**Fix:** Either:
- Protect access with a `sync.RWMutex`, or
- Store events on the `Server` struct and pass the server (or an event store interface) into handlers so it can be tested and serialized in one place.

---

## Design / consistency

### 5. configd ignores `ListenAndServe` error

**Where:** `cmd/configd/main.go`.

**Issue:** `http.ListenAndServe(":8080", nil)` return value is ignored. If the server fails to bind (e.g. port in use, permission denied), the process exits silently without logging.

**Fix:** `log.Fatal(http.ListenAndServe(":8080", nil))` or assign to `err` and `log.Fatalf("server: %v", err)`.

---

### 6. Events not persisted (known gap)

**Where:** `internal/server/handler_events.go` — comment says “in-memory event list (for demo; replace with persistence later)”.

**Issue:** Events are lost on restart. `DeleteEvent` is also a no-op. README TODO mentions “persist for clock to read”.

**Recommendation:** Tracked in README; when adding persistence, consider file (e.g. JSON) or the same gob pattern as config, and wire deletion/update.

---

### 7. No authentication on config UI

**Where:** configd serves the config UI and POST endpoints on `:8080` with no auth.

**Issue:** Any client on the same network can change config and events. Acceptable for a home/LAN-only device (as in the README), but should be explicit.

**Recommendation:** Document “config UI has no authentication; only use on trusted networks.” If the device is ever exposed, add at least a simple auth or bind to localhost and use SSH/port forwarding.

---

## Minor / quality

### 8. Debug logging to stdout

**Where:** `internal/server/handler.go` — many `fmt.Println("Brightness:"+...)`, etc.

**Issue:** Noisy in production and mixes debug output with normal logs.

**Fix:** Remove or guard behind a debug flag / log level, and use a logger (e.g. `log.Printf`) if kept.

---

### 9. Config validation

**Where:** Server parses form values into config (brightness, refresh rate, colors, etc.).

**Issue:** Refresh rate is validated (1–900 seconds); brightness and other numeric/color fields are not. Invalid values (e.g. negative brightness, huge numbers) are written to config and then used by display/clock.

**Recommendation:** Add validation (ranges, non-negative where needed) before assigning to `c` and returning errors to the user (e.g. 400 + message).

---

### 10. Error responses and status codes

**Where:** Various handlers use `fmt.Fprintf(w, "error: ...")` without setting HTTP status.

**Issue:** Client sees a 200 OK with an error body. Harder for clients or future UI to detect failure.

**Fix:** Use `http.Error(w, message, http.StatusBadRequest)` (or 500 where appropriate) and optionally `w.WriteHeader` + body for more structured responses.

---

### 11. Test coverage

**Where:** Only `internal/display/display_test.go` exists.

**Issue:** No tests for `internal/server`, `internal/config`, or `cmd/` entry points. Regressions in config read/write, handlers, or hex parsing are not caught.

**Recommendation:** Add tests for: config Read/Write (and DefaultConfig copy behavior once fixed), `hexStringToUint32` (valid + invalid/short input), and at least one handler per endpoint (e.g. Home with missing config, UpdateConfig with valid/invalid form).

---

## What’s in good shape

- **Display package:** Clear interface, table-driven tests, nil and TicksPerHour guards already addressed.
- **Separation of concerns:** clockd vs configd vs cli-clock; display vs hw vs config.
- **Config sharing:** Single `config.gob` and hot-reload in clockd is straightforward.
- **Build/deploy:** Makefile and Docker for ARM and configd are documented and consistent with README.

---

## Suggested priority

1. **High:** Fix DefaultConfig mutation (copy when using default) and `WriteConfig` panic.
2. **High:** Add length check in `hexStringToUint32` to avoid panic.
3. **Medium:** Synchronize or refactor Events (mutex or server-owned store).
4. **Medium:** configd: log/fatal on `ListenAndServe` error; reduce or gate debug prints.
5. **Lower:** Config validation in server; HTTP status codes and error responses; more tests (config, server, hex parsing).
