# Archive reliability plan

Date: 2026-09-17

Status: Status implementation authorized. Deliver the work in separate functional PRs; merge and validate each step in production before starting the next one.

## Objective

Replace misleading, persistent `processing` flags with an explicit archive
lifecycle. Prevent low disk space from causing repeated failed recordings, and
preserve usable media when recording or finalization fails.

The analysis covers release v4.20.0 and the current repository. Individual
production incidents have not yet been verified against their job records and
logs.

## Agreed direction

- Start with the archive status replacement.
- Add a configurable free-space threshold and account for the additional space
  needed to finalize recordings.
- On insufficient space, stop recordings and block new starts. Do not implement
  recording pause/resume for storage pressure.
- Automatically allow new recordings after storage has recovered for a sustained
  period. Recordings stopped for storage pressure remain stopped; do not restart
  those stream IDs automatically, including after application restarts.
- Use the existing PostgreSQL/River infrastructure.
- Separate processing lifecycle, termination reason, and capture coverage. Omit
  `partial` from the lifecycle: a finalized, playable recording can be completed
  even when some broadcast content is missing.
- Document a recording timeline and gap map as a future feature outside this
  reliability fix. Do not add its schema, capture instrumentation, or UI as part
  of the current work.
- Use the five lifecycle states below. Intentional termination without playable
  output uses `failed`; do not add `partial` or `cancelled`.
- Keep `stop_reason` and a durable archive event history separate from lifecycle
  status. Implement them in a follow-up PR, not the initial status replacement.

## Findings that motivate the work

- Videos and queues have separate processing booleans. Queue steps already have
  their own status values. The normal completion path clears processing only
  after all required video and chat steps succeed.
- Terminal job errors mark a queue step as failed without ending the archive's
  processing state.
- In v4.20.0, the archive heartbeat indicates that the worker is alive, not that
  media is advancing. The current branch adds live media stall detection.
- Live download errors generally trigger partial-media finalization and clear
  the watched channel's live flag. Another check can create another archive for
  the same stream instead of continuing the original archive.
- A live MP4 retry treats an existing nonempty conversion output as completed.
  A damaged output can therefore prevent a fresh conversion from a valid source.
- Orphan recovery focuses on live video/chat download steps marked running; it
  does not reconcile every unfinished pipeline stage.

Relevant implementation locations:

- `internal/tasks/shared.go`: completion checks and terminal error handling.
- `internal/tasks/live_video.go`: recording termination and finalization handoff.
- `internal/tasks/video.go`: conversion retries and moving archive files.
- `internal/tasks/watchdog.go`: stale job and orphan recovery.
- `internal/live/live.go`: stream detection and duplicate checks.
- `internal/tasks/client/client.go`: cancellation of all jobs for an archive.
- `ent/schema/vod.go`, `ent/schema/queue.go`: current persisted state.

## Phase 1: Replace processing flags with archive status

### Status decision

Use one authoritative archive lifecycle status. Keep individual task statuses
for pipeline execution and diagnostics; do not maintain independently writable
copies of the overall lifecycle in both video and queue records.

Agreed lifecycle:

| Status | Meaning |
| --- | --- |
| `queued` | Accepted and awaiting execution, including a scheduled retry. |
| `running` | Video recording or download is in progress. |
| `finalizing` | Capture has ended; conversion, media validation, or final placement is pending or running. |
| `completed` | The required archive processing has completed successfully. |
| `failed` | Processing ended unsuccessfully. Recoverable source files may still exist. |

Lifecycle completion describes successful processing of the captured material,
not capture of the entire broadcast. Video download, conversion, and placement
must succeed. Enabled chat stages must finish or reach a terminal failure; a chat
failure does not invalidate a finished video. Task-level failures remain visible
in queue details. While remaining chat work is active, the archive is finalizing.
Required video/preparation failures produce failed. Automatic retries retain an
unfinished lifecycle and reset their task to pending.

### Separate lifecycle from capture coverage

Use `completed` for successfully finalized, playable captured media, including
recordings ended early by storage pressure or containing network-related gaps.
Do not introduce a `partial` lifecycle status for those cases.

Store the termination reason separately, such as normal stream end, user stop,
or storage pressure. Video and chat results remain separately diagnosable. The UI
must communicate known early termination or missing components without implying
that processing is still running or the video cannot be played.

Detailed capture coverage belongs to the future recording timeline feature
below. The current status replacement does not attempt to locate or reconstruct
missing intervals.

### Follow-up: termination reason and archive history

Store an optional structured `stop_reason` on the archive (for example
`stream_ended`, `user_stop`, or `storage_low`). It explains why capture ended and
must not be overwritten when finalization succeeds or fails.

Add a compact, durable archive event history in PostgreSQL. Events record archive
ID, time, event type, stage, applicable status transition, reason, and bounded
technical details such as job ID and attempt number. Save status transitions and
their events in the same transaction. Retain history with the archive rather
than tying it to process log or River job retention. Record meaningful changes
and errors, not progress ticks or full FFmpeg output.

Existing application and per-stage process logs remain the detailed diagnostic
source. They cannot replace this history: process logs can be overwritten by
retries or pruned, and River jobs have their own retention policy. This history
is not the future recording timeline/gap map.

### Implementation scope

- Define allowed transitions, terminal outcomes, and how video/chat task results
  determine the overall state. A chat error must not make finished video unusable.
- Add structured termination reasons and durable events in the next PR.
- Commit stage transitions and successor jobs transactionally.
- Connect existing terminal error, panic, cancellation callback, and exhausted
  retry handling to the lifecycle. Keep live stop/finalization behavior unchanged;
  redesigning cancellation and recovering cancelled pending jobs belongs to the
  later finalization/reconciliation work.
- Replace processing-based API fields, filters, UI controls, and player decisions
  together. Update Swagger, frontend DTOs, query invalidation, and translations.
- Backfill existing records from persisted video/task state: previously finished
  videos remain completed; pending, active, finalizing, and failed stages map to
  their lifecycle; processing videos without a queue become failed. This does
  not validate files or prove that each historical running/pending task still
  has a live River job. Reconcile those historical cases in Phase 4.
- Regenerate Ent and define explicit migration handling for obsolete fields. Do
  not introduce permanent legacy API behavior or dual writable status models.

Acceptance: terminal failures reported through the existing task handlers update
archive status; retries remain unfinished; completed video with terminal chat
failure can finish. Status, API filters, queue details, admin controls, and player
behavior use the same vocabulary. Broad crash/orphan reconciliation is separate.

The first PR removes the old API fields and the `processing` query filter, replacing
it with `status` (comma-separated values). Update API, worker, and frontend together;
old and new processes must not share the migrated database during rollout. Database
migration removes the obsolete columns; rollback to an old release requires a
pre-upgrade database backup. No compatibility flags or dual writes are introduced.

## Phase 2: Make finalization safe to repeat

- Treat conversion output as incomplete until FFmpeg succeeds and media validation
  passes. File existence and nonzero size are insufficient evidence.
- Preserve source media until the final output is validated and safely committed.
- Make conversion and file placement recoverable across process death, including
  a crash after moving a file but before updating the database.
- Add a targeted stop-and-finalize operation. Do not use blanket cancellation of
  all archive jobs for storage stops, since finalization jobs must still run.
- Produce an explicit terminal outcome when salvage fails; preserve remaining
  recovery material instead of retrying indefinitely or deleting it blindly.

Acceptance: interruption during capture, remux, or placement results in either a
validated archive or an explained failure with remaining sources preserved.

## Phase 3: Add storage protection

### Budget and monitored paths

Proposed threshold model:

`stop threshold = configured free-space floor + remaining finalization budget + shutdown margin`

The configured X GB is the free-space floor. The additional reserve is dynamic;
stopping only when X GB remains would be too late for large pending remuxes.

- Inspect both temporary and archive storage, grouping paths on the same
  filesystem so free space is not independently allocated twice.
- Account for all accepted work that still needs finalization, existing output
  bytes, capture growth during detection/shutdown, and concurrent writers.
- Include MP4/remux outputs, HLS handling, live-preview files, cross-filesystem
  copies, and chat-related output. Respect configured conversion parameters;
  arbitrary transcoding cannot be budgeted as a fixed copy of source size.
- Start with conservative estimates and bounded finalization concurrency.
- Apply the same storage admission decision to automatic starts, manual starts,
  already queued jobs, and VOD/clip downloads so they cannot consume the reserve.
- Prevent concurrent starters from independently spending the same capacity.

### Stop and release behavior

1. Persist the global admission block and storage reason before stopping capture.
2. Stop active recordings and their chat capture in a controlled way.
3. Finalize captured data within the reserved budget. Gate other media writers
   so optional work cannot consume that budget.
4. Persist the identities of streams stopped for storage pressure, scoped by
   platform and channel, and exclude them from automatic recording restarts.
5. Release admission automatically only after sufficient capacity, including the
   reserve and a recovery margin, has remained available for a configured period.

Check storage periodically and immediately before execution. Monitoring must
remain responsive when recording queues are saturated. Measurement errors and
unexpected write failures also need explicit handling; foreign disk usage can
invalidate any estimate between checks.

Expose the block reason, free capacity, reserve, and release condition in the
admin UI. Exact defaults for X, monitoring interval, recovery margin, recovery
duration, and output estimation remain to be chosen.

Acceptance: one low-space event stops recordings without generating replacement
archives. Restarting the application cannot bypass the block or restart stopped
streams. Recovery permits new streams automatically without resuming old ones.

## Phase 4: Reconcile all stages and inspect existing stuck archives

Build the reconciliation rules alongside the new lifecycle and expand them to
all archive stages. At startup and periodically, compare lifecycle state, River
jobs, and file evidence. Each unfinished archive needs valid running/scheduled
work or a justified terminal outcome. Isolate individual failures so one bad
archive cannot stop the entire reconciliation pass.

For existing incidents, first produce an inventory distinguishing final media,
recoverable sources, incomplete intermediates, and absent media. Present concrete
repair candidates before executing production repairs. Do not perform blanket
status resets, deletions, or automatic merges of old duplicate recordings.

Acceptance: existing stuck archives can be explained and repairable cases can be
handled without treating all failed jobs as lost recordings.

## Phase 5: Improve stream startup and transient interruptions

This is a later work package. Status and storage protection do not by themselves
prevent duplicate archives caused by transient stream availability.

- Retry temporary playlist unavailability during initial capture setup within
  the same archive.
- Distinguish an unavailable status check from confirmed stream termination.
- Correlate attempts by platform, channel, and immutable external stream ID;
  do not use the mutable external VOD ID as session identity.
- Continue network-interrupted captures within the same archive using safe
  recording parts and keep video, chat, and chapters synchronized. The future
  gap map is a separate feature, not an acceptance requirement for this fix.
- Keep different stream identities separate and preserve explicit stop intent.

Storage-triggered stops remain terminal; this phase does not introduce storage
pause/resume. Missing media cannot be promised recoverable unless it is still
available from the provider.

## Future feature: Recording timeline and gap map

This section records a later feature idea, originally called a "holes map".
It is outside the current reliability fix and requires its own design and
implementation scope. The status replacement and storage protection must not
depend on implementing it.

### Purpose

Describe which parts of a broadcast were captured and where content is missing
inside one archive. A recording may contain several capture attempts after
network interruptions and still finish with lifecycle status `completed`.
Coverage information supplements that status instead of changing it.

### Proposed model

Record captured intervals with a mapping between two time axes:

- Stream time: the position in the original broadcast.
- Playback time: the position in the saved, playable archive.

Derive gaps from the captured intervals where there is enough evidence. For
example, if stream minutes 10 through 12 are missing and the remaining media is
joined directly, playback minute 10 continues with stream minute 12. The gap
does not itself occupy two minutes in the saved video.

This mapping can support correct chat and chapter alignment as well as later
player markers showing interruptions. Whether playback should skip gaps or
represent them with placeholders is a future design decision.

### Evidence and limits

- Use captured media boundaries and trustworthy timing information to establish
  coverage. A connection failure's duration alone does not prove the duration of
  missing media: buffering or a successful retry may recover some content.
- Distinguish known missing intervals from estimated boundaries or unknown
  coverage. Do not claim exact gaps when the recording evidence is insufficient.
- Distinguish an internal interruption from starting an archive late or stopping
  it early; do not invent a broadcast endpoint when it is unknown.
- Keep video coverage separate from chat availability and chat capture gaps.
- Preserve stream identity: different broadcasts must not be merged merely to
  make a continuous timeline.
- The map documents gaps; it does not promise recovery of missing content.

When this feature is scheduled, design evidence capture alongside the recording
parts and reconnection workflow. Exact timing may no longer be reconstructable
from an already joined video. Do not backfill historical archives with guessed
intervals presented as facts.

Future validation should cover multiple interruptions, buffered or recovered
content, uncertain boundaries, video/chat time mapping, and separate stream
identities. These are feature tests, not requirements for the current fix.

## PR delivery sequence

1. **Archive status (current PR):** replace processing booleans with the five-state
   lifecycle, wire transitions and terminal failures, migrate persisted state,
   and update API/UI/filters/tests. Keep current capture and recovery mechanics.
   Do not include `stop_reason`, event history, disk monitoring, general orphan
   reconciliation, media salvage changes, or reconnect behavior in this PR.
2. **Termination reasons and history:** implement the agreed structured reason
   and transactional archive event history.
3. **Safe finalization:** make interrupted remux and file placement retryable.
4. **Storage protection:** add reserve accounting, controlled stops, and automatic
   admission recovery without restarting storage-stopped stream identities.
5. **Broader recovery:** reconcile all pipeline stages and separately inspect
   historical stuck archives before proposing production repairs.
6. **Stream continuity:** address transient startup failures and reconnection.

Describe the first PR as the foundation of this larger reliability plan while
stating its exact scope and limitations. Merge and verify the first step in
production before proceeding with subsequent PRs. The recording timeline and gap
map remains an independent future feature outside these fixes.

## Validation and delivery order

Follow the PR delivery sequence above, keeping each change independently reviewable.
Develop reconciliation with these changes, then inspect historical incidents.
Treat robust network reconnection as a separate follow-up.

Add focused tests with each implementation phase:

- State transitions, exhausted retries, chat-only failure, and migration of old
  records; API-first and worker-first startup.
- Invalid MP4 intermediates and crashes during remux or file placement.
- Concurrent recordings, shared/separate filesystems, low-space admission races,
  unexpected write failures, and restart during storage shutdown.
- Sustained recovery and exclusion of storage-stopped stream identities.
- Playlist failure at stream start, media stalls, same/new stream identity, and
  chat timing across capture gaps for the later reconnection work.

Use fake capacity readings, injected errors, isolated media, and existing
Testcontainers infrastructure. Preserve hard-crash tests rather than replacing
them with graceful shutdown tests. Do not fill or manipulate production storage
to test these behaviors.
