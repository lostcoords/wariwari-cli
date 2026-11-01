# WariWari CLI

`wariwari` is a command-line interface for WariWari. It follows the same shape as Fizzy CLI: token authentication, named profiles, JSON envelopes, breadcrumbs for next commands, and an embedded agent skill.

## Quick Start

```bash
curl -fsSL https://wariwari.jp/install.sh | sh
wariwari auth login wari_xxxxxxxx
wariwari doctor
wariwari group list --json
wariwari skill install
```

## Installation

Install script (Linux and macOS, into `~/.local/bin`):

```bash
curl -fsSL https://wariwari.jp/install.sh | sh
```

Pin a release with `WARIWARI_VERSION`:

```bash
curl -fsSL https://wariwari.jp/install.sh | WARIWARI_VERSION=v0.1.0 sh
```

The script checks the archive against the release's `checksums.txt` and prints
a `PATH` hint if `~/.local/bin` is not on it. `WARIWARI_BIN_DIR` installs
somewhere else.

Homebrew:

```bash
brew install lostcoords/tap/wariwari
```

From source, with Go:

```bash
go install github.com/lostcoords/wariwari-cli/cmd/wariwari@latest
```

### Upgrading

```bash
wariwari upgrade --check     # only report whether an update exists
wariwari upgrade             # install the latest release
wariwari upgrade --version v0.1.0
```

`upgrade` verifies the download against `checksums.txt` and replaces the binary
in place. A Homebrew install runs `brew upgrade lostcoords/tap/wariwari`
instead. If the agent skill is installed, `upgrade` refreshes it from the new
binary.

## Authentication

Create an API key on the WariWari web account page (Account → API keys) — there
is no API route to mint one. The key is shown once; save it with:

```bash
wariwari auth login wari_xxxxxxxx
```

A developer key can read and edit existing groups, including creating
participants and transactions, but **cannot create a group**; `POST /v1/groups`
is session-credential only and answers `403 ACCESS_DENIED`. Full access is
currently enabled, so developer keys have no monthly usage quota; short-term
per-IP and per-credential rate limits still apply.

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
wariwari account show                    # also: wariwari me
wariwari search "ramen"
wariwari group list
wariwari group list --archived --take 100
wariwari group show GROUP_TOKEN          # the full snapshot
wariwari group activity GROUP_TOKEN
wariwari group stats GROUP_TOKEN --date-range this_month
wariwari group fx-rate GROUP_TOKEN --from USD
wariwari group settings GROUP_TOKEN
wariwari balance GROUP_TOKEN
wariwari participant list GROUP_TOKEN
wariwari participant balance PARTICIPANT_TOKEN GROUP_TOKEN
wariwari category list GROUP_TOKEN
wariwari transaction list GROUP_TOKEN
wariwari transaction show TRANSACTION_TOKEN GROUP_TOKEN
wariwari transaction activity TRANSACTION_TOKEN GROUP_TOKEN
```

`group show` returns the whole group render state — metadata, `shareUrl` (the
invite link), `participants`,
`groupCategories`, `transactions`, `netByCurrency`, `suggestedPayments`, and
spending totals. `balance`, `participant list`, and `category list` read that
same snapshot; the API has no separate endpoints for them.

`participant balance` shows how one participant's balance adds up, transaction
by transaction, in the group currency.

In `netByCurrency`, positive means the participant is owed money and negative
means they owe. Transaction reads always contain `timeOfDay`, but it can be
`null` for an import whose source did not include a time. `paidByToken` can also
be `null` for historical data whose payer was removed.

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

`group stats` takes `--date-range` (`all_time`, `this_month`, `last_month`,
`last_3_months`, `last_6_months`, `last_12_weeks`, `this_year`),
`--participant` and `--category` (tokens), `--month YYYY-MM`, `--week DATE`, and
`--date DATE`.

### Exporting

```bash
wariwari group export GROUP_TOKEN > trip.csv
wariwari group export GROUP_TOKEN --output trip.csv
```

The export is the `wariwari.v1` CSV the web app produces and the importers
accept. It needs settings access to the group.

### Pagination

`group list`, `transaction list`, and the activity feeds are keyset paginated.
Responses carry `pagination.hasMore` and `pagination.cursor`; pass the cursor
back with `--cursor`. Generated next-page breadcrumbs preserve the current
filters, archived mode, and page size. `--take` defaults to 50 and caps at 100.

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
wariwari participant create GROUP_TOKEN --data '{"token":"qm1mVhS7nyyF99vfPYwfF3nN","name":"Ada"}'
wariwari participant update PARTICIPANT_TOKEN GROUP_TOKEN --data '{"name":"Ada","lockVersion":0}'
wariwari participant delete PARTICIPANT_TOKEN GROUP_TOKEN --lock-version 0
wariwari category create GROUP_TOKEN --data '{"name":"Food","emoji":"🍜"}'
wariwari category update CATEGORY_TOKEN GROUP_TOKEN --data '{"name":"Food","emoji":"🍜","lockVersion":0}'
wariwari category delete CATEGORY_TOKEN GROUP_TOKEN --lock-version 0
wariwari transaction create GROUP_TOKEN --data-file transaction-create.json
wariwari transaction update TRANSACTION_TOKEN GROUP_TOKEN --data-file transaction-update.json
wariwari transaction delete TRANSACTION_TOKEN GROUP_TOKEN --lock-version 2
wariwari payment create GROUP_TOKEN --data '{"fromToken":"FROM","toToken":"TO","amountCents":4000}'
wariwari payment update PAYMENT_TOKEN GROUP_TOKEN --data '{"fromToken":"FROM","toToken":"TO","amountCents":4500,"date":"2026-05-24","lockVersion":0}'
wariwari group settings update GROUP_TOKEN --data '{"visibility":"private_registered","lockVersion":3}'
wariwari group archive GROUP_TOKEN
wariwari group unarchive GROUP_TOKEN
wariwari group delete GROUP_TOKEN --lock-version 3
wariwari account update --data '{"locale":"ja"}'
```

Updates require `lockVersion` from the record you just read, and deletes take it
as `--lock-version`. A stale value returns `409 CONFLICT`, usually with the
fresh record under `current`. Re-apply your change on top of it and retry.

`payment update` replaces the payment: resend `date` and the other fields you
want to keep, or the date resets to today. `group settings update` changes
currency (existing transactions are rebased in the background), visibility,
existing categories by token (`categories: [{token, name, emoji, position}]`),
and roles of linked participants (`participants: [{token, membershipRole}]`).

A transaction needs `description`, `amountCents`, `currency`, `splitType`,
`date`, `paidByToken`, and `splits`. `splitType` is one of `equal`,
`percentage`, `shares`, `amount`. With `equal`, the splits may omit
`amountCents` and the server divides the amount evenly.

```json
{
  "token": "9AHTHuAGT7U2CpSfktrfUAYA",
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

The optional 24-character base58 `token` makes participant and transaction
creates safe to retry. Generate it once for the new resource and reuse the same
token and body on retries; never invent tokens for existing participants or
categories.

## Sharing and access

```bash
wariwari invitation create GROUP_TOKEN --role editor --expires-in 7d
wariwari invitation revoke INVITATION_TOKEN GROUP_TOKEN
wariwari access-request status GROUP_TOKEN    # your own request: none, pending, declined, viewable
wariwari access-request create GROUP_TOKEN    # ask a private group's admins
wariwari access-request list GROUP_TOKEN      # admins: pending requests
wariwari access-request approve ID GROUP_TOKEN
wariwari access-request decline ID GROUP_TOKEN
```

A public group is shared with the snapshot's `shareUrl`; invitations are for
private groups. Invitations and access-request reviews are admin-only.

## Feedback

```bash
wariwari feedback "The CSV export is missing the category emoji"
```

## Not in the CLI

The CLI authenticates with a developer API key, and some routes accept only a
first-party app session: listing and signing out sessions, sign-in identities,
payment methods, device tokens, email change, account deletion, imports, and
group creation. Use the app for those.

## Errors

Failures are RFC 9457 Problem Details. The CLI prints `CODE: detail (HTTP nnn)`,
lists any `errors` fields, and with `--json` emits the whole problem document —
including `current` on a `409` and `retryAfter` on a `429`
(`RATE_LIMIT_EXCEEDED` or `QUOTA_EXCEEDED`) — so a caller can switch on `code`.

## Agent Integration

```bash
wariwari skill
wariwari skill install
```

`wariwari skill` prints the embedded skill. `wariwari skill install` writes it to
`~/.agents/skills/wariwari`, `~/.claude/skills/wariwari` (Claude Code) and
`$CODEX_HOME/skills/wariwari` (Codex, default `~/.codex`).
`wariwari upgrade` refreshes an installed skill along with the binary.

Example payloads:

```bash
wariwari group example create --json
wariwari participant example create --json
wariwari category example create --json
wariwari transaction example create --json
wariwari payment example create --json
```

## Development

```bash
make build
make test-unit
make check
```
