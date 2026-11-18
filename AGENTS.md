# WariWari CLI Development Context

This repository mirrors the Fizzy CLI pattern for WariWari.

## Structure

```
wariwari-cli/
├── cmd/wariwari/       # Main entrypoint
├── internal/config/    # Config files, profiles, credential storage
├── internal/client/    # Direct WariWari HTTP API client
├── internal/commands/  # Cobra command implementations
└── skills/             # Embedded agent skill
```

## API

The CLI talks to WariWari's `/v1` JSON API with `Authorization: Bearer <token>`.
The token is a developer API key (`wari_…`). `auth login TOKEN` saves one, and
`auth login` with no argument gets one through the browser (below).
The authoritative contract is the OpenAPI 3.1 bundle in the server repo
(`openapi/dist/wariwari.bundled.yaml`). Check a route there before adding or
changing one here.

### Wire format

**Every field is camelCase**: `lockVersion`, `amountCents`, `splitType`,
`paidByParticipantToken`, `participantToken`, `emailAddress`, `categoryToken`,
`{ "pagination": { "hasMore", "cursor" } }`.

Two carve-outs:

- Maps keyed by data pass through untouched — `fxRates`,
  `spendingByCurrency`, `netByCurrency` (keyed by ISO-4217 code; `netByCurrency`'s
  inner objects by participant token).
- Values are never converted. Enum strings stay snake_case (`read_only`), as do
  URL path segments (`/fx_rate`) and path parameters (`:group_token`). Enums
  are open: pass an unknown value through, never fail on it.

### Routes the CLI uses

| Command                      | Route                                                         |
| ---------------------------- | ------------------------------------------------------------- |
| `auth login` (no token)      | `POST /v1/cli_authorizations`, then poll `…/token` (no auth)  |
| `account show` / `me`        | `GET /v1/account`                                             |
| `account update`             | `PATCH /v1/account` (flat body, no `user` wrapper)            |
| `search`                     | `GET /v1/search` (`q`)                                        |
| `feedback`                   | `POST /v1/feedback`                                           |
| `group list`                 | `GET /v1/groups` (`archived`, `take`, `cursor`)               |
| `group create`               | `POST /v1/groups`                                             |
| `group show` / `snapshot`    | `GET /v1/groups/:group_token`                                 |
| `group update`               | `PATCH /v1/groups/:group_token` (+ rev; also `visibility`)    |
| `group delete`               | `DELETE /v1/groups/:group_token` (+ expected revision)        |
| `group activity`             | `GET /v1/groups/:group_token/activity`                        |
| `group archive` / `unarchive`| `POST` / `DELETE /v1/groups/:group_token/archival`            |
| `group duplicate`            | `POST /v1/groups/:group_token/duplication`                    |
| `group join` / `leave`       | `POST` / `DELETE /v1/groups/:group_token/membership`          |
| `group notifications`        | `GET` / `PUT /v1/groups/:g/notification_settings` (`muted`)   |
| `group stats`                | `GET /v1/groups/:group_token/stats` (filters below)           |
| `group fx-rate`              | `GET /v1/groups/:group_token/fx_rate` (`from`, `date`)        |
| `group export`               | `GET /v1/groups/:group_token/csv_export` (`text/csv`)         |
| `balance`                    | `GET /v1/groups/:group_token` (derived)                       |
| `participant list`           | `GET /v1/groups/:group_token` (derived)                       |
| `participant balance`        | `GET /v1/groups/:g/participants/:token/balance`               |
| `participant create`         | `POST /v1/groups/:group_token/participants`                   |
| `participant update`         | `PATCH /v1/groups/:group_token/participants/:token` (+ rev; also `membershipRole`) |
| `participant delete`         | `DELETE /v1/groups/:group_token/participants/:token` (+ rev)  |
| `participant merge`          | `POST /v1/groups/:g/participants/:token/merge` (+ rev)        |
| `participant move`           | `PATCH /v1/groups/:g/participants/:token/position` (`position`, no rev) |
| `participant claim`          | `POST /v1/groups/:g/participants/:token/claim`                |
| `category list`              | `GET /v1/groups/:group_token` (derived)                       |
| `category create`            | `POST /v1/groups/:group_token/categories`                     |
| `category update`            | `PATCH /v1/groups/:group_token/categories/:token` (+ rev)     |
| `category delete`            | `DELETE /v1/groups/:group_token/categories/:token` (+ rev)    |
| `transaction list`           | `GET /v1/groups/:g/transactions` (filters below)              |
| `transaction show`           | `GET /v1/groups/:g/transactions/:transaction_token`           |
| `transaction create`         | `POST /v1/groups/:g/transactions`                             |
| `transaction update`         | `PATCH /v1/groups/:g/transactions/:transaction_token` (+ rev) |
| `transaction delete`         | `DELETE /v1/groups/:g/transactions/:transaction_token` (+ rev)|
| `transaction activity`       | `GET /v1/groups/:g/transactions/:t/activity`                  |
| `transaction bulk-edit`      | `PATCH /v1/groups/:g/transaction_batch` (`expectedRevision` per item) |
| `transaction bulk-delete`    | `POST /v1/groups/:g/transaction_deletions` (same items)       |
| `payment create`             | `POST /v1/groups/:g/transactions` (`kind: payment`)           |
| `payment update`             | `PATCH /v1/groups/:g/transactions/:transaction_token` (`kind: payment`, + rev) |
| `invitation list`            | `GET /v1/groups/:g/invitations`                               |
| `invitation create`          | `POST /v1/groups/:g/invitations`                              |
| `invitation revoke`          | `DELETE /v1/groups/:g/invitations/:invitation_token`          |
| `invitation accept`          | `POST /v1/groups/:g/invitations/:invitation_token/acceptance` |
| `access-request status`      | `GET /v1/groups/:g/access_request`                            |
| `access-request create`      | `POST /v1/groups/:g/access_request`                           |
| `access-request list`        | `GET /v1/groups/:g/access_requests`                           |
| `access-request approve`     | `POST /v1/groups/:g/access_requests/:id/approval`             |
| `access-request decline`     | `DELETE /v1/groups/:g/access_requests/:id`                    |
| `transaction receipt show`   | `GET /v1/groups/:g/transactions/:t/receipt_photo`             |
| `transaction receipt upload` | `PUT /v1/groups/:g/transactions/:t/receipt_photo` (multipart) |
| `transaction receipt delete` | `DELETE /v1/groups/:g/transactions/:t/receipt_photo`          |
| `file list` / `upload`       | `GET` / `POST /v1/groups/:g/attachments` (upload multipart)   |
| `file show` / `delete`       | `GET` / `DELETE /v1/groups/:g/attachments/:id`                |
| `recurring list` / `create`  | `GET` / `POST /v1/groups/:g/recurring_expenses`               |
| `recurring show` / `update` / `delete` | `GET` / `PATCH` / `DELETE /v1/groups/:g/recurring_expenses/:token` |

`transaction list` filters: `from`, `to` (inclusive ISO dates), `categoryToken`,
`participantToken` (paid-for **or** owes-a-share), plus `take` / `cursor`.
`group stats` takes `dateRange`, `participantToken`, `categoryToken`, `date`,
`month` (`YYYY-MM`), and `week`. The flags stay `--category` and
`--participant`.

### Routes that do not exist

Do not reintroduce these — the API has no such surface:

- `GET …/snapshot` — the snapshot **is** `GET /v1/groups/:group_token`.
- `GET …/balances` — derive from the snapshot's `netByCurrency` and
  `suggestedPayments`.
- `GET …/participants`, `GET …/categories` — no list endpoints; read
  `participants` / `categories` off the snapshot.
- `/payments`, `/settings`, `/dismissal`, `…/participants/:token/link` — a
  payment is a transaction, visibility is on the group, a role is on the
  participant, and leaving is `DELETE …/membership`.
- `/v1/api_keys` (any verb) — API-key management is deliberately web-only. The
  `token` command exists only to say so.
- `GET /v1/me` — removed; `GET /v1/account` returns the same user payload.

### Browser login

`auth login` without a token runs the RFC 8628 device authorization grant in
`internal/commands/browser_login.go`. Both requests are unauthenticated, even
when a token is configured. `POST /v1/cli_authorizations` with `{clientName}`
(the hostname, at most 80 characters) returns `deviceCode`, `userCode`,
`verificationUriComplete`, `expiresAt` and `interval`. The CLI then polls
`POST /v1/cli_authorizations/token` with `{deviceCode}` every `interval`
seconds. The poll returns 200 `{token, account}` once approved. A 400 problem
`code` means: `AUTHORIZATION_PENDING`, keep polling; `SLOW_DOWN`, add 5s to the
interval for good; `AUTHORIZATION_DENIED` or `EXPIRED_TOKEN`, stop. Anything
else is fatal, and so is passing `expiresAt`. The token is saved through the same
`saveLogin` path as `auth login TOKEN`. Prompts go to stderr, and stdout carries
only the JSON envelope.

### Session-only routes

The CLI authenticates with a developer API key (`wari_…`), and these routes
answer it with `403 ACCESS_DENIED`, so the CLI does not wrap them:
`/v1/account/sessions*`, `/v1/account/identities*`,
`/v1/account/payment_methods*`, `/v1/account/device_tokens`,
`/v1/account/email_change`, `DELETE /v1/account`, and the imports
(`/v1/import_drafts*`, `…/groups/:g/imports`). They are absent from the public
bundle the CLI vendors.

Partial updates are **PATCH**. **PUT** replaces a whole singleton: notification
settings and the multipart receipt photo. Uploads go
through `client.Client.Upload`.
`receipt show` reads a 404 as "none" only after the parent expense reads
(`absentOn404`).

### Semantics worth remembering

- **A write returns the resource it changed, and a delete answers `204`.**
  Only `GET /v1/groups/:group_token` returns the snapshot. On a `204` the
  client returns nil and the command prints its own result. `group create`
  returns the group without participants; join, claim, invitation accept and
  access-request approval return the participant.
- **Every update, delete and merge of a record with a `lockVersion` sends it in
  the `Expected-Revision` header** (`client.Write.ExpectedRevision`), never in
  the body: groups, categories, participants, transactions. `revisedWrite`
  sends `--lock-version` when given, otherwise reads the record first; a record
  the read cannot find goes at revision 0, so the server answers (an already
  deleted record succeeds again, an unknown one is 404). A stale value is
  `409 CONFLICT` with the fresh record under `current`, keyed by its kind
  (`current.transaction`). Creates and reorders send none, a recurring expense
  has no `lockVersion`, and bulk edits and deletes carry `expectedRevision` on
  each item of `transactions`.
- **`POST /v1/groups` works with developer keys**, 20 groups an hour.
- **`--idempotency-key` exists exactly where the operation declares
  `Idempotency-Key`**: every write under a group and `group create`, not
  account update or feedback. `spec_test.go` enforces it.
- **The CLI sends no `Recorded-At`.** The header dates a write a client queued
  offline, and the CLI writes as it runs. It reads no `Read-Model-Version`
  either: it holds no group between commands.
- **Every failure is one Problem body**: `code`, `title`, `detail`, `status`,
  `instance`, plus `errors` (validation, keyed by wire field path or header
  name) and `current` (409 CONFLICT). There is no `type`. Switch on `code`;
  `VALIDATION_FAILED` is a `400`, not a `422`. A `429` is
  `RATE_LIMIT_EXCEEDED` or `QUOTA_EXCEEDED`, with `Retry-After`.
  `403 PRO_REQUIRED` and `429 QUOTA_EXCEEDED` carry `upgradeUrl`, and their
  `detail` states the price and limit: print it as sent, never a copy.
  `/v1/account/purchase_sync` is session-only and absent from the public spec.
- **Pagination is keyset**: `take` defaults to 50 and caps at 100; read
  `pagination.hasMore` / `pagination.cursor`.

## Conventions

- Keep command behavior thin and predictable.
- Put config and credential behavior in `internal/config`.
- Keep HTTP mechanics in `internal/client`.
- Return JSON envelopes by default, and raw data with `--quiet`.
- Preserve Fizzy's setup pattern: `setup`, `doctor`, `auth`,
  `config`, `skill`, `commands`.
- Keep `skills/wariwari/SKILL.md` in lockstep with the commands — it is what an
  agent follows, so every example must run and every field must be camelCase.

## Releasing

The version lives in the tag; GoReleaser stamps it into `main.version`, which
`wariwari --version` prints. The skill ships inside the binary, so a release
also ships the skill; `wariwari upgrade` reinstalls it.

Releasing is one step: `git tag vX.Y.Z && git push origin vX.Y.Z`.

- `release.yml` publishes the archives and `checksums.txt` to GitHub Releases.
- `lostcoords/homebrew-tap` runs `.github/workflows/wariwari.yml` every 15
  minutes: it regenerates `Formula/wariwari.rb` from the latest release with
  `scripts/homebrew-formula.sh` and commits any change. The tap pulls because
  the org disables deploy keys, so the CLI can't push to it.
- Tags with a `-` (`v0.3.0-rc1`) publish as prereleases and stay off Homebrew.
- `nightly.yml` rebuilds the `nightly` prerelease from every main commit that
  passes Test, versioned `<next patch>-nightly.<sha>`.

`scripts/install.sh` (served at `https://wariwari.jp/install.sh`) and
`wariwari upgrade` read the release's `checksums.txt`, so archive names must
stay `wariwari_<version>_<os>_<arch>.tar.gz`. Both honour
`WARIWARI_RELEASE_URL` to test against a local `goreleaser release --snapshot
--clean` build in `dist/`.

## Checks

```bash
make build       # go build -o bin/wariwari ./cmd/wariwari
make check       # gofmt -l, go vet, go test ./internal/...
make sync-spec   # copy $WARIWARI_SERVER/openapi/dist/wariwari.bundled.yaml into testdata (default ../wariwari)
```

`contract_test.go` lists every command with the request it ends with and the
spec `operationId` it calls; add a row for each new command. `spec_test.go`
checks the rows against the vendored bundle
(`internal/commands/testdata/wariwari.openapi.yaml`): method, path template,
required headers, the `--idempotency-key` flag, and that every public operation
has a command. It also fails when the server checkout beside this repo changed
an operation since the vendored copy; run `make sync-spec` and update the
commands.
