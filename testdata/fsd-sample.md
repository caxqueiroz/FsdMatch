# Notebook Service — Functional Specification (sample fixture)

This is a synthetic FSD authored for fsdtrace's smoke tests. It is not
derived from any real product.

## Authentication

### FR-001 — User signs in with email and password

**Description**: A registered user authenticates by submitting an email
and a password. On success the system returns a short-lived bearer
token. Failed logins increment a per-account counter.

**Acceptance criteria**:
- POST `/api/v1/auth/login` with valid credentials returns 200 and a JWT.
- Invalid credentials return 401 with no token leaked in the body.
- Five consecutive failures within ten minutes lock the account for
  fifteen minutes.

**Actor**: end user.
**Non-functional**: median latency under 200 ms.

### FR-002 — Session token refresh

**Description**: A client exchanges a refresh token for a new bearer
token without re-prompting the user. The refresh token is rotated on
each call.

**Acceptance criteria**:
- POST `/api/v1/auth/refresh` with a valid refresh token returns 200
  and a new bearer + refresh pair.
- An already-used refresh token returns 401 and revokes the family.

## Notes

### FR-010 — Create a note

**Description**: An authenticated user creates a note. Notes consist of
a title and a markdown body. The system records the author and a
created-at timestamp.

**Acceptance criteria**:
- POST `/api/v1/notes` with `{title, body}` returns 201 and the new
  note with a server-assigned id.
- Titles longer than 200 characters return 400.
- Bodies up to 10 MiB are accepted; larger bodies return 413.

**Side effects**: emits `notes.created` to the `notes-events` Kafka
topic.

### FR-011 — List my notes

**Description**: An authenticated user lists notes they authored, most
recent first. Results are paginated.

**Acceptance criteria**:
- GET `/api/v1/notes?cursor=...` returns 200 with `{items, next_cursor}`.
- Page size defaults to 20 and is capped at 100.
- Notes authored by other users are never returned.

## Scheduled jobs

### FR-020 — Nightly note compaction

**Description**: A scheduled job runs nightly to compact soft-deleted
notes older than thirty days, freeing storage. The job logs the number
of rows reclaimed.

**Acceptance criteria**:
- Runs at 02:00 UTC every day.
- Logs a summary line including `compacted_count`.
- Tolerates concurrent invocations: a leader-election lock prevents
  double execution.

**Non-functional**: completes within fifteen minutes for one million
candidate rows.
