# WariWari CLI

`wariwari` is a command-line interface for WariWari. It follows the same shape as Fizzy CLI: token authentication, named profiles, JSON envelopes, breadcrumbs for next commands, an embedded agent skill, and Claude plugin metadata.

## Quick Start

```bash
make build
./bin/wariwari setup
./bin/wariwari doctor
./bin/wariwari group list --json
./bin/wariwari transaction example create --json
```

## Installation

From source:

```bash
go install github.com/lostcoords/wariwari-cli/cmd/wariwari@latest
```

Local build:

```bash
make build
```

## Authentication

Create an API key on the WariWari web account page (Account → API keys) — there
is no API route to mint one. The key is shown once; save it with:

```bash
wariwari auth login wari_xxxxxxxx
```

A developer key can read and edit existing groups but **cannot create a group**;
`POST /v1/groups` is session-credential only and answers `403 ACCESS_DENIED`.

## Configuration

```
~/.config/wariwari/
├── config.yaml        # Global fallback settings
├── config.json        # Named profiles
└── credentials/       # Fallback token storage when keyring is disabled/unavailable

.wariwari.yaml         # Per-repo local config
```

Configuration precedence, highest to lowest:

1. CLI flags: `--token`, `--profile`, `--api-url`, `--group`
2. Environment variables: `WARIWARI_TOKEN`, `WARIWARI_PROFILE`, `WARIWARI_API_URL`, `WARIWARI_GROUP`
3. Named profile settings in `~/.config/wariwari/config.json`
4. Local project config, `.wariwari.yaml`
5. Global config, `~/.config/wariwari/config.yaml` or `~/.wariwari/config.yaml`
6. Defaults, including `https://api.wariwari.jp`

Credentials are stored in the OS keyring by default. Set `WARIWARI_NO_KEYRING=1` to force file fallback.

## Reading

```bash
wariwari me
wariwari group list
wariwari group list --archived --take 100
wariwari group show GROUP_TOKEN          # the full snapshot
wariwari group activity GROUP_TOKEN
wariwari balance GROUP_TOKEN
wariwari participant list GROUP_TOKEN
wariwari category list GROUP_TOKEN
wariwari transaction list GROUP_TOKEN
wariwari transaction show TRANSACTION_TOKEN GROUP_TOKEN
wariwari transaction activity TRANSACTION_TOKEN GROUP_TOKEN
```

`group show` returns the whole group render state — metadata, `participants`,
`groupCategories`, `transactions`, `netByCurrency`, `suggestedPayments`, and
spending totals. `balance`, `participant list`, and `category list` read that
same snapshot; the API has no separate endpoints for them.

### Filtering transactions

```bash
wariwari transaction list GROUP_TOKEN \
  --from 2026-01-01 --to 2026-01-31 \
  --category CATEGORY_TOKEN \
  --participant PARTICIPANT_TOKEN
```

`--participant` matches transactions that participant paid for **or** owes a
share of. Filters narrow rather than assert: a well-formed token that matches
nothing returns an empty page.

### Pagination

`group list`, `transaction list`, and the activity feeds are keyset paginated.
Responses carry `pagination.hasMore` and `pagination.cursor`; pass the cursor
back with `--cursor`. `--take` defaults to 50 and caps at 100.

```bash
wariwari group list --take 100 --json
wariwari group list --take 100 --cursor eyJraW5kIjoia2V5c2V0In0 --json
```

## Writing

Write commands accept a JSON body via `--data` or `--data-file`. All fields are
camelCase, and every write returns the refreshed group snapshot.

```bash
wariwari group create --idempotency-key "$(uuidgen)" \
  --data '{"name":"Trip","icon":"🧳","currency":"JPY","participants":[{"name":"Ada"}]}'
wariwari group update GROUP_TOKEN --data '{"name":"Trip 2026","lockVersion":1}'
wariwari participant create GROUP_TOKEN --data '{"name":"Ada"}'
wariwari participant update PARTICIPANT_TOKEN GROUP_TOKEN --data '{"name":"Ada","lockVersion":0}'
wariwari participant delete PARTICIPANT_TOKEN GROUP_TOKEN
wariwari category create GROUP_TOKEN --data '{"name":"Food","emoji":"🍜"}'
wariwari category update CATEGORY_TOKEN GROUP_TOKEN --data '{"name":"Food","emoji":"🍜","lockVersion":0}'
wariwari category delete CATEGORY_TOKEN GROUP_TOKEN
wariwari transaction create GROUP_TOKEN --data-file transaction-create.json
wariwari transaction update TRANSACTION_TOKEN GROUP_TOKEN --data-file transaction-update.json
wariwari transaction delete TRANSACTION_TOKEN GROUP_TOKEN
```

Updates require `lockVersion` from the record you just read. A stale value
returns `409 CONFLICT` with the fresh record under `current` — re-apply your
change on top of it and retry.

A transaction needs `description`, `amountCents`, `currency`, `splitType`,
`date`, `paidByToken`, and `splits`. `splitType` is one of `equal`,
`percentage`, `shares`, `amount`.

```json
{
  "kind": "expense",
  "description": "Team dinner",
  "amountCents": 12000,
  "currency": "JPY",
  "splitType": "equal",
  "date": "2026-05-23",
  "paidByToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
  "splits": [
    { "participantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj", "amountCents": 6000 },
    { "participantToken": "8mPNkR4qQXtLbJR5CcxT6yWa", "amountCents": 6000 }
  ]
}
```

## Errors

Failures are RFC 9457 Problem Details. The CLI prints `CODE: detail (HTTP nnn)`,
lists any `errors` fields, and with `--json` emits the whole problem document —
including `current` on a `409` — so a caller can switch on `code`.

## Agent Integration

```bash
wariwari skill
wariwari skill install
wariwari setup claude
```

The embedded skill installs to `~/.agents/skills/wariwari/SKILL.md`.
`wariwari skill install` also installs the skill to the Codex skill directory.
`wariwari setup claude` also links it into `~/.claude/skills/wariwari`.

Example payloads:

```bash
wariwari group example create --json
wariwari participant example create --json
wariwari category example create --json
wariwari transaction example create --json
```

## Development

```bash
make build
make test-unit
make check
```
