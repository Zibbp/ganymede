# Ganymede contributor guide for coding agents

## Project intent

Ganymede is a self-hosted VOD and live-stream archiving system. It currently has deep Twitch support: it discovers channels and broadcasts, captures video and chat, post-processes media, stores archive metadata, and serves playback with synchronized chat. Archives must remain useful outside Ganymede, so stable files, metadata, and directory layouts are product behavior rather than implementation details.

The long-term direction is platform-generic. New domain code should use platform-neutral names and depend on `internal/platform.Platform` when practical. Do not imply that YouTube is fully supported merely because `utils.VideoPlatform` includes it; the current implementation, routes, archive services, download/chat tools, and tests still contain substantial Twitch-specific behavior.

## Repository map

- `cmd/server/main.go`: API process entrypoint and Swagger API metadata.
- `cmd/worker/main.go`: River worker process entrypoint and graceful shutdown.
- `internal/server/server.go`: API dependency composition, migrations, services, River insertion client/UI, and platform setup.
- `internal/worker/worker.go`: worker dependency composition and River execution client setup.
- `internal/transport/http/`: Echo routes, handlers, request/response types, auth middleware, and handler-local service interfaces.
- `internal/<domain>/`: services for archives, VODs, channels, live watches, queues, playlists, playback, users, auth, notifications, and administration.
- `internal/tasks/`: River job arguments/workers and archive pipeline logic.
- `internal/tasks/registry/registry.go`: authoritative list of executable River workers.
- `internal/tasks/worker/worker.go`: queues, concurrency, periodic jobs, middleware, and worker runtime.
- `internal/tasks/client/`: API-side insertion/administration client. It does not execute work.
- `internal/platform/`: platform-neutral models/interface and the Twitch implementation.
- `internal/exec/`: external process integration for yt-dlp, FFmpeg/ffprobe, TwitchDownloaderCLI, and live chat.
- `internal/database/`: PostgreSQL pools, Ent and River migrations, seeding, and shared transactions.
- `internal/config/`: environment configuration and persisted `config.json` application settings.
- `internal/storagetemplate/`, `internal/hls/`, `internal/chat/`, `internal/nfo/`: archive format and media support.
- `ent/schema/`: source of truth for database entities.
- `ent/`: mostly generated Ent code. Files marked generated must not be edited manually.
- `frontend/app/`: Next.js App Router pages, components, hooks, stores, and providers.
- `frontend/messages/`: next-intl locale JSON; English is the fallback/source locale.
- `tests/setup.go`: shared PostgreSQL Testcontainers application/HTTP/worker setup.
- `tests/shared/`: integration polling, archive completion, media, and River assertion helpers.
- `docs/`: generated Swag API artifacts.
- `Dockerfile`, `entrypoint.sh`, `supervisord.conf`: combined production image and process lifecycle.

Treat `dev/`, `.env`, generated binaries, `tmp/`, archive media, logs, and test working directories as local/runtime data. Do not commit or rewrite them as part of an unrelated change.

## Architecture and design conventions

### Backend boundaries

The API and worker are separate Go processes but share PostgreSQL and much of the service graph. The API validates requests, persists domain state, and inserts River jobs. The worker owns job execution and periodic scheduling. A feature that changes startup dependencies may need equivalent wiring in both `internal/server/server.go` and `internal/worker/worker.go`.

Domain packages generally expose a `Service` constructed with `*database.Database` and explicit collaborators, and use Ent directly. HTTP packages declare narrow service interfaces near their handlers; preserve this seam so handlers remain mockable. Pass `context.Context` from the request/job through new operations instead of introducing new `context.Background()` calls.

Use Zerolog for structured logging. Add useful fields such as queue ID, VOD ID, job kind, or channel rather than formatting them only into the message. Return errors with operation context. Avoid panics outside process startup paths.

Keep function and inline code comments concise and to the point. Explain intent, invariants, or non-obvious constraints that the code cannot express clearly; do not narrate straightforward code or write multi-paragraph essays for a single function. Prefer clearer names and smaller functions over lengthy explanatory comments.

### Persistence and transactions

PostgreSQL is the only supported application database. Ent owns application schema, while River owns job schema. Startup serializes migration with a PostgreSQL advisory lock.

When changing entities:

1. Edit `ent/schema/*.go`.
2. Run `make ent_generate`.
3. Commit the resulting generated Ent changes.
4. Add migration/backfill compatibility in `internal/database/` when automatic additive migration is insufficient. Ent auto-migration does not remove old columns.
5. Test both API-first and worker-first startup assumptions when migration behavior changes.

Do not hand-edit files beginning with `Code generated by ent`.

Use `Database.WithTx` when application state and a River job must be atomic. It binds Ent operations and `RiverClient.InsertTx` to the same `database/sql` transaction. Archive creation and status-plus-successor transitions intentionally use this pattern; do not split them into a database commit followed by a non-transactional enqueue.

### Archive and River job workflow

An archive is a persisted VOD plus queue state followed by a chain of retryable River workers. Job arguments normally carry the queue UUID, and workers reload current VOD/queue/channel data. Dedicated queues throttle expensive download, post-processing, chat, render, and sprite work; the default queue handles lightweight or time-sensitive jobs.

When adding or changing a job:

- Define stable JSON-tagged arguments and `Kind()`/`InsertOpts()` behavior.
- Make work safe to retry and resume after process death. Do not assume a failed command left no partial files or child processes.
- Update queue/task state consistently and preserve heartbeat/watchdog semantics.
- Enqueue successors transactionally with the state transition where applicable.
- Register every executable worker in `internal/tasks/registry/registry.go`.
- Add periodic scheduling in `internal/tasks/worker/worker.go` only when the task is genuinely periodic.
- Preserve archive tags, uniqueness rules, metadata correlation, cancellation, and recovery behavior.
- Use contexts and process-group cleanup for external commands so cancellation does not orphan FFmpeg, yt-dlp, or capture processes.

Do not replace hard-crash recovery tests with graceful shutdown tests. `tests/worker_process.go` deliberately launches and SIGKILLs a subprocess to exercise real recovery behavior.

### Platform-generic development

`internal/platform.Platform` is the primary provider seam for channels, VODs, clips, live streams, categories, badges, and emotes. For new platform support:

- Keep shared domain models, archive lifecycle, queue state, and storage behavior platform-neutral.
- Put provider API/GQL/auth details behind a platform implementation.
- Select providers by `utils.VideoPlatform`; avoid adding more `PlatformTwitch` fields or passing an assumed Twitch instance into new generic services.
- Isolate provider-specific URL construction, downloader flags, chat formats, badges/emotes, and credential requirements.
- Persist and validate the platform on channels/VODs/jobs so an external ID is never interpreted without its provider.
- Add provider-neutral tests with fakes at the interface seam, then provider-specific contract/E2E coverage separately.

Existing Twitch naming is technical debt, not a template for new abstractions. Refactor it incrementally only within the requested scope; a platform feature can span server/worker composition, archive/live services, HTTP routes, tasks, exec helpers, config, Ent enums, frontend DTOs, and tests.

### HTTP API and authorization

Echo routes live under `/api/v1` and are registered in `internal/transport/http/handler.go`. `/health`, `/metrics`, `/swagger/*`, River UI, direct media routes, and the catch-all Next.js proxy are outside that group.

Normal JSON responses use `{ "success": boolean, "data": ..., "message": string }` through `SuccessResponse` and `ErrorResponse`. Preserve that contract because frontend hooks unwrap `response.data.data` and display response messages.

For endpoint changes:

- Keep parsing/validation and HTTP status mapping in the handler; keep domain work in a service.
- Declare or extend the handler's narrow service interface rather than coupling it to a concrete service unnecessarily.
- Update the route, handler tests, Swag annotations, frontend hook/types, and translations/UI together when applicable.
- Treat public reads as intentional decisions, not defaults.
- For dual session/API-key endpoints, preserve middleware order: `AuthAPIKeyOrSessionMiddleware`, `AuthGetUserMiddleware`, then `RequireRoleOrScope`.
- Match the existing role hierarchy (`admin > editor > archiver > user`) and the equivalent resource scope (`read`, `write`, or `admin`). Backend authorization is authoritative even when the UI hides controls.
- API-key mint/update/revoke endpoints remain session-only and admin-only; a key must not be able to mint or escalate another key.

Swagger source annotations are in `cmd/server/main.go` and `internal/transport/http/*.go`. `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml` are generated and should not be edited directly. Regenerate them with the repository's Swag tooling when changing documented API behavior, and review the generated diff.

### Filesystem and media safety

Archive paths are configurable and may point at large local disks, NFS, or SMB mounts. Use storage-template helpers and `filepath` operations instead of assembling new layouts ad hoc. Keep temporary and final paths distinct, make moves/finalization restart-safe, and preserve sidecar metadata, chat, caption, thumbnail, HLS, and NFO compatibility.

Deletion code must validate that the resolved target belongs to the intended archive/channel before removing anything. Never weaken path containment/name checks or recursively delete a path derived only from unchecked input. Tests must use `t.TempDir()` or the directories created by `tests/setup.go`, never real archive locations.

External tool arguments are observable behavior. Preserve argument ordering, proxy/cookie handling, codec/container compatibility, progress/log parsing, signals, and timeouts. When changing media commands, test argument construction cheaply and add an actual media/process test when behavior depends on FFmpeg/ffprobe/yt-dlp.

## Frontend conventions

The frontend uses Next.js App Router, React, strict TypeScript, Mantine, CSS Modules, TanStack Query, Zustand, Axios, Vidstack, and next-intl. Most pages are client components. Global composition is in `frontend/app/layout.tsx` and `frontend/app/providers.tsx`.

- Build pages from Mantine primitives and colocated `*.module.css`; keep the dark-default theme and existing responsive breakpoints coherent.
- Put server data and mutations in domain hooks under `frontend/app/hooks/`. Hooks manually define API DTOs—there is no generated TypeScript client—so backend response changes require explicit type updates.
- Use stable domain-prefixed TanStack query keys containing every filter that affects the result. Invalidate all affected queries after mutations. Preserve deliberate polling intervals and `keepPreviousData` behavior.
- Use the public Axios singleton only for public endpoints. Use `useAxiosPrivate()` with credentials for authenticated operations and mutations.
- Reserve Zustand for client/persisted UI state and authentication. Do not duplicate server-owned data in a store when TanStack Query owns it.
- Keep loading/error early-return patterns and use the shared loading components where appropriate.
- Use Mantine forms with Zod validation for nontrivial forms; retain confirmation modals for destructive actions.
- Keep video/chat synchronization efficient. Player ticks, range-based chat fetching, clip offsets, resume progress, theater mode, and multistream synchronization are tightly coupled and should be tested together after changes.
- Never treat frontend role checks as security controls.

New user-visible text belongs in `frontend/messages/en.json`, then in each supported locale or via the English fallback. Run `node frontend/translation-coverage.js` from the repository root to inspect coverage. The `-u` option writes missing keys into locale files, so review its output. A new locale also requires the Navbar language list to be updated.

New browser-visible environment variables must be safe to expose and added consistently to `frontend/next.config.ts` (`makeEnvPublic`) and the `<EnvScript>` mapping in `frontend/app/layout.tsx`.

Follow local style in touched files; this codebase currently mixes relative/alias imports and semicolon styles. Avoid broad formatting-only churn. Note that `usePlayerSore.ts` is an existing misspelled filename; do not silently rename it without updating every import and checking persisted behavior.

## Testing

Ganymede has two meaningful test classes:

1. Unit tests exercise pure or contained behavior with standard `testing`, Testify, table-driven cases, `t.TempDir()`, and often `t.Parallel()`. Good examples are in `internal/utils`, `internal/config`, `internal/nfo`, `internal/hls`, and focused `internal/tasks` tests.
2. Integration/E2E tests import `github.com/zibbp/ganymede/tests` and use real PostgreSQL through Testcontainers. They may initialize the full application, start a River worker or HTTP server, invoke external media tools, and contact real Twitch APIs.

The filename or package alone does not determine cost: service tests under `internal/<service>` often call `tests.Setup(t)` and are integration tests. Some `internal/exec` tests are locally scoped but still require FFmpeg/ffprobe and POSIX process tools.

Shared setup variants in `tests/setup.go` are intentional:

- `tests.Setup(t)`: application plus an in-process River worker.
- `tests.SetupWithoutWorker(t)`: application without a worker, used by crash/recovery scenarios.
- `tests.SetupHTTP(t)`: real Echo server on a free port plus an `httpexpect` client.

Reuse a setup within a top-level test where the existing suite does so; every setup starts a fresh Postgres container and application. Use helpers in `tests/shared` for polling rather than fixed sleeps. Wait only for jobs belonging to the archive under test; unrelated periodic River jobs can remain scheduled.

`SKIP_SECRET_TESTS=true` skips setups that require trusted Twitch credentials. This is for fork CI and constrained environments, not a way to declare the full suite green. Never add secrets, tokens, real `.env` contents, downloaded media, or test artifacts to commits.

## Commands and validation

Run the narrowest useful checks while iterating, then expand according to risk.

```bash
# Go formatting and focused tests
gofmt -w path/to/changed.go
go test ./internal/<package>

# All Go tests: up to 30 minutes, Docker + external tools/credentials may be needed
make test

# Go lint
make lint

# Backend builds
make build_server
make build_worker

# Ent regeneration
make ent_generate

# Frontend development and validation
make dev_web
cd frontend && npx tsc --noEmit
cd frontend && npm run build
```

The full CI suite builds the `tests` target from `Dockerfile`, creates the `ganymede-tests` Docker network, mounts the Docker socket for Testcontainers, and runs `make test`. Reproduce that environment when a local host lacks the pinned FFmpeg, yt-dlp, TwitchDownloaderCLI, fonts, or ICU dependencies.

`frontend/package.json` currently defines `npm run lint` as `next lint`, but that command is incompatible with the installed Next.js 16 toolchain. Do not report it as a passing check unless the lint script has been repaired; use TypeScript and the production build meanwhile. There is currently no frontend test runner, so add targeted tests only together with an intentional test framework choice, or document manual UI verification.

Before handing off a change, report exactly which checks ran and which were skipped. For integration failures, distinguish code failures from missing Docker, unavailable live streams, rate limits, absent credentials, or missing external binaries.

## Development and deployment notes

- `make dev_setup` installs Air/godotenv tooling and frontend dependencies; `make dev_server`, `make dev_worker`, and `make dev_web` run the three development processes.
- Server/worker builds inject commit, tag, and build time into `internal/utils`; preserve these ldflags in release paths.
- The production image combines API, worker, standalone Next.js, FFmpeg/ffprobe, yt-dlp, TwitchDownloaderCLI, and fonts. Supervisor runs all three application processes and exits if one becomes fatally unhealthy.
- Data mounts and `VIDEOS_DIR`, `TEMP_DIR`, `LOGS_DIR`, and `CONFIG_DIR` must agree. `entrypoint.sh` changes ownership unless `SKIP_CHOWN=true`; network-storage compatibility matters.
- Runtime application settings live in persisted `config.json`, while connection/path/bootstrap settings come from environment variables. Consult both `internal/config/config.go` and `internal/config/env.go`; support existing `<VAR>_FILE` secret handling.
- Avoid unrelated dependency upgrades. Keep `go.mod`/`go.sum` or `frontend/package.json`/`package-lock.json` synchronized when dependencies intentionally change.

## Change checklists

For a backend API feature, usually inspect: domain service, transport interface/handler/route/middleware, response envelope, handler/service tests, Swagger annotations/generated docs, frontend DTO/hook/query invalidation, UI permissions, and translations.

For an archive/task feature, usually inspect: API-side atomic creation/enqueue, job args and stable JSON, worker registry, queue choice/concurrency, state transitions, retry/cancel/heartbeat/watchdog behavior, partial-file and process cleanup, filesystem layout, service/unit tests, and Docker-backed E2E coverage.

For a schema feature, usually inspect: `ent/schema`, regenerated Ent files, migration/backfill/drop compatibility, API DTOs, frontend manual DTOs, and both fresh-DB and upgrade behavior.

Keep patches scoped. Preserve user changes in a dirty worktree, do not edit generated artifacts by hand, and explain any verification that could not be completed.
