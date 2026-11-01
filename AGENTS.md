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
The authoritative contract is the OpenAPI 3.1 bundle in the server repo
(`openapi/dist/wariwari.bundled.yaml`) and its prose spec (`docs/specs/api.md`).
Check a route there before adding or changing one here.

### Wire format

**Every field is camelCase**: `lockVersion`, `amountCents`, `splitType`,
`paidByToken`, `participantToken`, `emailAddress`, `groupCategoryToken`,
`{ "pagination": { "hasMore", "cursor" } }`.

Two carve-outs:

- Maps keyed by data pass through untouched — `fxRates`,
  `spendingByCurrency`, `netByCurrency` (keyed by ISO-4217 code; `netByCurrency`'s
  inner objects by participant token), and on requests `splitPercentages` /
  `splitShares` (keyed by participant token).
- Values are never converted. Enum strings stay snake_case (`read_only`), as do
  URL path segments (`/fx_rate`) and path parameters (`:group_token`).

### Routes the CLI uses

| Command                      | Route                                                         |
| ---------------------------- | ------------------------------------------------------------- |
| `account show` / `me`        | `GET /v1/account`                                             |
| `account update`             | `PATCH /v1/account` (flat body, no `user` wrapper)            |
| `search`                     | `GET /v1/search` (`q`)                                        |
| `feedback`                   | `POST /v1/feedback`                                           |
| `group list`                 | `GET /v1/groups` (`archived`, `take`, `cursor`)               |
| `group create`               | `POST /v1/groups` (+ `Idempotency-Key`)                       |
| `group show` / `snapshot`    | `GET /v1/groups/:group_token`                                 |
| `group update`               | `PATCH /v1/groups/:group_token`                               |
| `group delete`               | `DELETE /v1/groups/:group_token` (+ expected revision)        |
| `group activity`             | `GET /v1/groups/:group_token/activity`                        |
| `group archive` / `unarchive`| `POST` / `DELETE /v1/groups/:group_token/archival`            |
| `group stats`                | `GET /v1/groups/:group_token/stats` (filters below)           |
| `group fx-rate`              | `GET /v1/groups/:group_token/fx_rate` (`from`)                |
| `group export`               | `GET /v1/groups/:group_token/csv_export` (`text/csv`)         |
| `group settings`             | `GET /v1/groups/:group_token/settings`                        |
| `group settings update`      | `PATCH /v1/groups/:group_token/settings`                      |
| `balance`                    | `GET /v1/groups/:group_token` (derived)                       |
| `participant list`           | `GET /v1/groups/:group_token` (derived)                       |
| `participant balance`        | `GET /v1/groups/:g/participants/:token/balance`               |
| `participant create`         | `POST /v1/groups/:group_token/participants`                   |
| `participant update`         | `PATCH /v1/groups/:group_token/participants/:token`           |
| `participant delete`         | `DELETE /v1/groups/:group_token/participants/:token` (+ rev)  |
| `category list`              | `GET /v1/groups/:group_token` (derived)                       |
| `category create`            | `POST /v1/groups/:group_token/categories`                     |
| `category update`            | `PATCH /v1/groups/:group_token/categories/:token`             |
| `category delete`            | `DELETE /v1/groups/:group_token/categories/:token` (+ rev)    |
| `transaction list`           | `GET /v1/groups/:g/transactions` (filters below)              |
| `transaction show`           | `GET /v1/groups/:g/transactions/:transaction_token`           |
| `transaction create`         | `POST /v1/groups/:g/transactions`                             |
| `transaction update`         | `PATCH /v1/groups/:g/transactions/:transaction_token`         |
| `transaction delete`         | `DELETE /v1/groups/:g/transactions/:transaction_token` (+ rev)|
| `transaction activity`       | `GET /v1/groups/:g/transactions/:t/activity`                  |
| `payment create`             | `POST /v1/groups/:g/payments`                                 |
| `payment update`             | `PATCH /v1/groups/:g/payments/:payment_token`                 |
| `invitation create`          | `POST /v1/groups/:g/invitations`                              |
| `invitation revoke`          | `DELETE /v1/groups/:g/invitations/:invitation_token`          |
| `access-request status`      | `GET /v1/groups/:g/access_request`                            |
| `access-request create`      | `POST /v1/groups/:g/access_request`                           |
| `access-request list`        | `GET /v1/groups/:g/access_requests`                           |
| `access-request approve`     | `POST /v1/groups/:g/access_requests/:id/approval`             |
| `access-request decline`     | `POST /v1/groups/:g/access_requests/:id/decline`              |

`transaction list` filters: `from`, `to` (inclusive ISO dates), `category`
(group category token), `participant` (paid-for **or** owes-a-share), plus
`take` / `cursor`. `group stats` takes `dateRange`, `participant` and `category`
(both tokens), `date`, `month` (`YYYY-MM`), and `week`.

### Routes that do not exist

Do not reintroduce these — the API has no such surface:

- `GET …/snapshot` — the snapshot **is** `GET /v1/groups/:group_token`.
- `GET …/balances` — derive from the snapshot's `netByCurrency` and
  `suggestedPayments`.
- `GET …/participants`, `GET …/categories` — no list endpoints; read
  `participants` / `groupCategories` off the snapshot.
- `/v1/api_keys` (any verb) — API-key management is deliberately web-only. The
  `token` command exists only to say so.
- `GET /v1/me` — removed; `GET /v1/account` returns the same user payload.

### Session-only routes

The CLI authenticates with a developer API key (`wari_…`), and these routes
answer it with `403 ACCESS_DENIED`, so the CLI does not wrap them:
`/v1/account/sessions*`, `/v1/account/identities*`,
`/v1/account/payment_methods*`, `/v1/account/device_tokens*`,
`/v1/account/email_change`, `DELETE /v1/account`, the imports, and
`POST /v1/groups`.

Updates are **PATCH**, never PUT. There is no PUT anywhere in the API, and
`client.Client` has no `Put` method.

### Semantics worth remembering

- **Group-resource writes return the refreshed group snapshot** (`201`/`200`),
  including deletes of nested resources and access-request approvals. Group
  delete, archive/unarchive, invitation revoke and access-request decline are
  `204`; the client returns nil and the command prints its own result.
  `account update` returns the account, `invitation create` the invitation.
- **`lockVersion` is required on every update.** Omitting it is
  `400 VALIDATION_FAILED`; a stale value is `409 CONFLICT`, with the fresh record
  under `current` for groups, categories, transactions and participants
  (`current` is absent for other records).
- **Deletes are guarded too.** Group, participant, category and transaction
  deletes need the record's `lockVersion` in the `WariWari-Expected-Revision`
  header (`--lock-version`); `client.DeleteRevision` sends it.
- **`POST /v1/groups` is session-credential only.** A `wari_` developer key gets
  `403 ACCESS_DENIED`.
- **The CLI sends `Idempotency-Key` on `group create` only.** The API accepts it
  on most group writes, but participant, transaction and payment creates are
  already retry-safe through client-supplied tokens.
- **Errors are RFC 9457 Problem Details**: `type`, `title`, `status`, `detail`,
  `code`, `instance`, plus `errors` (validation) and `current` (409). Switch on
  `code`; `VALIDATION_FAILED` is a `400`, not a `422`. A `429` is
  `RATE_LIMIT_EXCEEDED` or `QUOTA_EXCEEDED`, with `Retry-After`.
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

1. `git tag vX.Y.Z && git push origin vX.Y.Z`. The release workflow publishes
   the archives and `checksums.txt` to GitHub Releases.
2. `scripts/homebrew-formula.sh X.Y.Z > Formula/wariwari.rb` in a checkout of
   `lostcoords/homebrew-tap`, then commit and push the tap.

`scripts/install.sh` (served at `https://wariwari.jp/install.sh`) and
`wariwari upgrade` read the release's `checksums.txt`, so archive names must
stay `wariwari_<version>_<os>_<arch>.tar.gz`. Both honour
`WARIWARI_RELEASE_URL` to test against a local `goreleaser release --snapshot
--clean` build in `dist/`.

## Checks

```bash
make build       # go build -o bin/wariwari ./cmd/wariwari
make check       # gofmt -l, go vet, go test ./internal/...
```

`internal/commands/contract_test.go` pins the command↔route table above. Update
it whenever a route moves.
