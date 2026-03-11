# 32HybridHV — Test Plan

## Objectives

1. Validate that the runtime appliance correctly executes Win32 binaries inside the guest.
2. Confirm that the **time shim and guest OS** behave correctly for dates past **2038-01-19** (and further into the future).
3. Establish a repeatable regression baseline so regressions are caught before release.

## Post-2038 Test Strategy

### Why virtualized time?

Running tests by waiting until 2038 is not practical. Instead, we use two approaches:

1. **Clock offset injection** — the time shim accepts an environment variable `HYBRIDHV_TIME_OFFSET_SECONDS` that adds a fixed offset to all intercepted time calls. This shifts the effective time seen by Wine and the Win32 app without changing the host clock.
2. **Fake system time via `libfaketime`** — for integration tests, the guest runs Wine under `libfaketime` to simulate an arbitrary date.

Both approaches are complementary: clock offset injection tests the shim itself; `libfaketime` tests the full stack.

### Test date targets

| Test date | Significance |
|---|---|
| 2038-01-19 03:14:07 UTC | 32-bit signed `time_t` overflow boundary |
| 2038-01-20 00:00:00 UTC | One day after overflow |
| 2042-01-01 00:00:00 UTC | Far-future sanity check (some other format overflows) |
| 2106-02-07 06:28:15 UTC | 32-bit unsigned `time_t` overflow boundary |

## Smoke Tests

These are the minimum tests that must pass for every build.

### ST-01: Health check

- **What:** Call `HealthCheck` RPC.
- **Expected:** Returns `status: "ok"` and a valid server timestamp.
- **Validates:** Guest agent is up, gRPC is reachable, auth token works.

### ST-02: RunExe — hello world (Win32)

- **What:** Submit a minimal Win32 console binary that prints `hello` to stdout and exits 0.
- **Expected:** `RunExe` returns `exit_status: 0`; `StreamLogs` streams a line containing `hello`.
- **Validates:** Wine launches, the binary executes, stdout is captured.

### ST-03: RunExe — exit code passthrough

- **What:** Submit a binary that exits with code 42.
- **Expected:** `RunExe` returns `exit_status: 42`.
- **Validates:** Exit code is correctly propagated through Wine → guest agent → RPC response.

### ST-04: PutFile / GetFile round-trip

- **What:** Upload a file via `PutFile`, then download it via `GetFile`.
- **Expected:** Downloaded content is byte-for-byte identical to uploaded content.
- **Validates:** Chunked streaming works; no corruption.

### ST-05: Auth rejection

- **What:** Send a `HealthCheck` RPC with an invalid token.
- **Expected:** gRPC returns `UNAUTHENTICATED`.
- **Validates:** Token validation is enforced.

### ST-06: Allowlist rejection

- **What:** Connect from an IP not on the allowlist.
- **Expected:** Connection is refused or gRPC returns `PERMISSION_DENIED`.
- **Validates:** IP allowlist is enforced.

### ST-07: Time shim — post-2038 time() call

- **What:** Run a Win32 test binary that calls `time()` and prints the result. Inject `HYBRIDHV_TIME_OFFSET_SECONDS` to simulate 2038-01-20.
- **Expected:** The binary prints a value ≥ `2147483648` (i.e., a value that would overflow a 32-bit signed int); no crash or wrap-around.
- **Validates:** Time shim intercepts `time()` and returns a 64-bit-safe value.

### ST-08: Time shim — `gettimeofday()` and `clock_gettime()`

- **What:** Same as ST-07 but for `gettimeofday` and `clock_gettime(CLOCK_REALTIME)`.
- **Expected:** Both return consistent, correct post-2038 values.
- **Validates:** Shim covers all common time entry points.

## Test Matrix

| Test ID | Component | Date context | Automation |
|---|---|---|---|
| ST-01 | Guest agent | Current | CI |
| ST-02 | Wine runner | Current | CI |
| ST-03 | Wine runner | Current | CI |
| ST-04 | File transfer | Current | CI |
| ST-05 | Auth | Current | CI |
| ST-06 | Allowlist | Current | CI |
| ST-07 | Time shim | Post-2038 (offset) | CI |
| ST-08 | Time shim | Post-2038 (offset) | CI |
| IT-01 | Full stack | 2038-01-19 (libfaketime) | Nightly |
| IT-02 | Full stack | 2042-01-01 (libfaketime) | Nightly |
| IT-03 | Full stack | 2106-02-07 (libfaketime) | Nightly |

## Test Infrastructure Notes

- Smoke tests (`ST-*`) run on every CI push using Go's `testing` package and a test helper that starts the guest agent in-process (no VM required).
- Integration tests (`IT-*`) require a provisioned Hyper-V VM and run nightly on a dedicated runner.
- Test binaries used as targets are stored in `hv/tests/fixtures/` and are minimal, purpose-built Win32 PE binaries compiled from source (C).
- The post-2038 fixtures must be compilable with `mingw-w64` (`i686-w64-mingw32-gcc`) to produce 32-bit PE executables.
