# Supplier Catalog Updates

A Go application with an interactive Web UI (React + Ant Design + Tailwind CSS) and CLI tool for managing a shop's local copy of its supplier catalog.

It handles delta imports delivered in batches, survives supplier outages or crashes mid-update without losing progress, prevents duplicates, and guarantees that older or repeated data never overwrites newer information or brings back withdrawn products.

---

## Key Features

- **Robust Versioning (`updated_at` timestamps):**
  - Evaluates each record's supplier timestamp. Older records arriving out of order are safely skipped.
  - Repeating the same update will never produce duplicate records (`supplier_sku` as natural primary key).
  - Withdrawn items are retained as tombstones with their withdrawal timestamp, preventing earlier updates from accidentally restoring them.
- **Resilient Batch Checkpointing:**
  - Each batch of products and its run checkpoint are committed together in **a single SQLite transaction**.
  - If the supplier drops offline or the application stops mid-batch, zero corrupted state is saved.
- **Resume Incomplete Updates:**
  - An interrupted update can be resumed seamlessly from its last confirmed batch without re-applying or duplicating earlier work.
  - Partially completed updates are clearly marked as `interrupted` and are never reported as `completed`.
- **Triage & Change Log:**
  - Staff can view what changed in every run (new products, updated prices/names, withdrawals, skipped stale records).
  - Malformed records or supplier connection failures are isolated as **Problems** requiring staff attention, without failing the entire run.
- **Dual Interface:**
  - Interactive Web Dashboard built with React, Ant Design, and Tailwind CSS.
  - Comprehensive command-line interface (`catalog` CLI) for terminal usage and scripting.
- **Built-in Fake Supplier:**
  - Provides realistic camping gear data with pre-built scenarios (repeated records, out-of-order deliveries, malformed records, and an outage toggle).

---

## Architecture Overview

```
                      +-----------------------------+
                      |   Fake Supplier HTTP API    |
                      | (:8081, simulated scenarios)|
                      +--------------+--------------+
                                     ^
                                     | HTTP (Batches)
                                     v
+------------------+         +-------+-------+         +---------------------+
| Staff CLI Tool   | <-----> |   Go Backend  | <-----> |  SQLite (WAL mode)  |
| (cmd/catalog)    |  REST   |  (cmd/server) |  Tx/Tx  |  catalog.db         |
+------------------+         +-------+-------+         +---------------------+
                                     ^
                                     | Serves API & Static UI
                                     v
                      +--------------+--------------+
                      |   React Web Dashboard       |
                      |   (Ant Design + Tailwind)   |
                      +-----------------------------+
```

### Core Design Rules (`updated_at` ordering)

When a product record arrives with a supplier `updated_at`:
1. **Never Seen SKU + Active:** Insert as `active`.
2. **Never Seen SKU + Withdrawn:** Insert as `withdrawn` tombstone (to guard against delayed out-of-order older inserts).
3. **Incoming `updated_at` <= Stored `updated_at`:** Record is stale or a duplicate. **Skipped** and recorded in change log.
4. **Incoming `updated_at` > Stored `updated_at`:**
   - If incoming is active: Update fields and set status to `active`.
   - If incoming is withdrawn: Mark as `withdrawn` tombstone.
5. **Validation Error (e.g., negative price, missing name, invalid timestamp):** Logged into `problems` table for triage. The stored catalog product remains untouched.

---

## Project Structure

```
.
├── cmd/
│   ├── server/           # Main Go server (JSON API + embedded web UI)
│   ├── fakesupplier/     # Standalone fake supplier HTTP service with outage controls
│   └── catalog/          # Staff CLI tool
├── internal/
│   ├── catalog/          # Domain types, pure merge rules & unit tests
│   ├── store/            # SQLite repository (WAL mode, single-conn, transactions)
│   ├── importer/         # Update orchestrator, retry logic, recovery & checkpoints
│   ├── supplier/         # Supplier HTTP client and wire structs
│   ├── fakesupplier/     # Demo dataset and scenario definitions
│   └── api/              # REST API handlers
├── web/                  # React + Ant Design + Tailwind CSS frontend
│   ├── src/              # React components (StartUpdate, RunsTable, ProblemsTable, etc.)
│   └── dist/             # Production build served by the Go backend
├── README.md
└── go.mod
```

---

## Prerequisites

- **Go 1.22+** (Go 1.23 recommended)
- **Node.js 18+** & **npm** (only needed if modifying the frontend)

> Note: SQLite is run using `modernc.org/sqlite`, a pure Go implementation that requires **no CGO and no GCC compiler**, working out of the box on Windows, macOS, and Linux.

---

## Instructions to Run

> **Tip (Reset Database):** To start fresh with a clean catalog at any time:
> - **Windows (PowerShell):** `Remove-Item -Force -ErrorAction SilentlyContinue catalog.db, catalog.db-shm, catalog.db-wal`
> - **Linux / macOS:** `rm -f catalog.db*`

### 1. Run the Fake Supplier
In your first terminal, start the fake supplier service:
```bash
go run ./cmd/fakesupplier -addr :8081
```
*Options:*
- `-delay 1s`: Adds a 1-second pause between batches so you can observe live progress or trigger interruptions.
- `-outage=true`: Starts the supplier in an outage state (HTTP 503).

### 2. Run the Main Server (Backend & Web UI)
In your second terminal:
```bash
go run ./cmd/server -addr :8080 -supplier http://localhost:8081 -static web/dist
```
Open **[http://localhost:8080](http://localhost:8080)** in your browser to view the Web UI.

*(Optional: If running the frontend in Vite live-reload mode, run `npm run dev` in `web/` and open http://localhost:5173).*

---

## Using the CLI Tool

You can also run tasks from the terminal:

```bash
# Build the CLI
go build -o catalog.exe ./cmd/catalog

# View available commands
./catalog.exe

# Start an update
./catalog.exe start NW-2025-09-01

# List update runs and current statuses
./catalog.exe runs

# Inspect detailed changes for a run
./catalog.exe changes 1

# List problems that need staff attention
./catalog.exe problems open

# Mark a problem as resolved
./catalog.exe resolve 1

# Inspect the local catalog
./catalog.exe products active

# Resume an interrupted run
./catalog.exe resume 1
```

---

## Running the Tests

To run all unit and integration tests:

```bash
go test -v ./internal/catalog ./internal/importer
```

### What the Tests Verify:
1. **`TestDecide` / `TestDiff`:** Validates pure business merge rules (idempotency, out-of-order rejection, tombstone retention, revival with newer timestamp).
2. **`TestImportCompletes`:** Verifies clean end-to-end multi-batch import with counts and problem detection.
3. **`TestImportingSameUpdateTwiceCreatesNoDuplicates`:** Confirms identical updates create zero new rows and are 100% skipped.
4. **`TestOutOfOrderAndOlderDataDoNotWin`:** Confirms older updates delivered late cannot overwrite newer prices or revive withdrawn items.
5. **`TestSupplierOutageKeepsCatalogAndCanResume`:** Simulates HTTP 503 mid-run, ensures existing catalog is preserved, marks run `interrupted`, then resumes to completion.
6. **`TestResumeAfterApplicationRestart`:** Simulates abrupt app shutdown (`Ctrl+C`), restarts database connection, runs crash recovery, and resumes from the exact batch checkpoint.
7. **`TestAbandonedRunIsNeverCompleted`:** Verifies abandoned runs remain incomplete and keep previously committed batches intact.

---

---

## Assumptions

1. **Monotonic Timestamps:** The supplier provides an ISO-8601 `updated_at` timestamp per product that accurately reflects when each change occurred.
2. **Delta Feeds:** Products omitted from a batch are unaffected; removals must be explicitly communicated with `withdrawn: true`.
3. **Sequential Batches:** Batches in an update are numbered sequentially starting from 1 with the final batch indicating `last: true`.
4. **Single Worker:** Only one update run can be actively importing at any given time to avoid concurrent lock contention and race conditions.
5. **Natural Keys:** `supplier_sku` is stable and uniquely identifies a product across all updates.

---

## Unfinished Work & Future Improvements

- **Multi-Supplier Support:** Add a `supplier_id` foreign key column to allow multiple suppliers with isolated catalogs.
- **Periodic Background Reconciliation:** Full periodic catalog syncs to detect orphaned products if the supplier ever stops sending explicit withdrawal events.
- **Authentication & RBAC:** Add role-based authentication (e.g. JWT / session cookies) for staff users on the web dashboard and CLI.
- **Streaming WebSockets / SSE:** Replace the 2.5s dashboard polling loop with Server-Sent Events (SSE) for instant real-time progress updates.

---

## AI Tools Usage

During the preparation of this submission:
- **Claude / Gemini (Antigravity):** Used as an architectural sounding board to discuss edge cases around out-of-order delta imports, verify SQLite WAL durability settings, scaffold boilerplate for Ant Design table definitions, and assist in drafting test scenarios.
- **Manual Verification:** All business merge rules, database constraints, state transition guards, and test suites were reviewed and verified through automated test runs and manual execution.
