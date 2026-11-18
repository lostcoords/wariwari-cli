---
name: wariwari
description: Use the WariWari CLI to manage WariWari groups, participants, categories, transactions, payments, balances, stats, exports, and access requests from an agent.
triggers:
  - wariwari
  - /wariwari
  - create wariwari event
  - create expense in wariwari
  - create payment in wariwari
  - list wariwari groups
  - show wariwari balances
  - settle up in wariwari
  - export wariwari group
invocable: true
argument-hint: "[action] [args...]"
---

# WariWari CLI

Use `wariwari` for WariWari API work. Prefer JSON output for agent workflows:

```bash
wariwari <command> --json
```

## The three rules that matter

1. **Every field on the wire is camelCase** — `lockVersion`, `amountCents`,
   `splitType`, `paidByParticipantToken`, `participantToken`, `emailAddress`,
   `categoryToken`. Never send `amount_cents` or `lock_version`; they are
   rejected.
2. **The group snapshot is the one read of a whole group.** `wariwari group
   show TOKEN` returns metadata, `visibility`, `shareUrl` (the invite link),
   `participants`, `categories`, `transactions`, `netByCurrency`,
   `suggestedPayments` and spending totals in one payload.
   There are no participants or categories list endpoints; the
   `participant list`, `category list`, and `balance` commands read this
   snapshot for you.
3. **A write returns only the record it changed, and a delete returns
   nothing.** Read the snapshot again when you need the new balances.

Maps keyed by data keep their keys verbatim: `fxRates`, `spendingByCurrency`
and `netByCurrency` are keyed by ISO-4217 currency code, and `netByCurrency`'s
inner objects by participant token. Enum *values* stay snake_case
(`read_only`), and an enum can gain a value: treat one you don't know as
unknown rather than failing.

## Setup

```bash
wariwari setup
wariwari doctor
wariwari auth status
```

Authenticate with an API key: `wariwari auth login wari_...` saves one, or set
`WARIWARI_TOKEN`. Keys are created on the web account page (Account → API keys);
there is no API route to list or mint one. `wariwari auth login` with no
argument runs a browser login instead: it prints a one-time code and waits for
a person to approve it in their browser. Use it only when the user is at the
terminal and asks for it; otherwise ask them for a key.
Free API keys can read with a monthly limit; writes need wariwari Pro and
answer `403 PRO_REQUIRED` without it. The server's error says the current price
and limit; show its `detail` and `upgradeUrl` as sent. On a Pro account a key
can do everything below, including creating groups (20 an hour). Short-term per-IP and per-credential rate limits apply
either way.

Developer keys cannot reach session-only routes (sessions, sign-in identities,
payment methods, device tokens, email change, account deletion, imports); the
CLI has no commands for them. Ask the user to do those in the app.

If a command or field in this skill is missing from the CLI, run
`wariwari upgrade --check`; `wariwari upgrade` installs the latest release and
refreshes this skill.

For Codex-style agents, `wariwari skill install` writes the skill to both
`~/.agents/skills/wariwari/` and the Codex skill directory.

Configuration precedence:

1. CLI flags: `--token`, `--profile`, `--api-url`, `--group`
2. Environment: `WARIWARI_TOKEN`, `WARIWARI_PROFILE`, `WARIWARI_API_URL`, `WARIWARI_GROUP`
3. Named profiles in `~/.config/wariwari/config.json`
4. Local `.wariwari.yaml`
5. Global `~/.config/wariwari/config.yaml`

## Reading

```bash
wariwari account show --json                    # or: wariwari me
wariwari search "ramen" --json                  # groups, people, expenses across all groups
wariwari group list --json
wariwari group list --archived --take 100 --json
wariwari group show GROUP_TOKEN --json          # the full snapshot
wariwari group activity GROUP_TOKEN --json
wariwari group stats GROUP_TOKEN --date-range this_month --json
wariwari group fx-rate GROUP_TOKEN --from USD --date 2026-05-23 --json
wariwari group notifications show GROUP_TOKEN --json
wariwari invitation list GROUP_TOKEN --json
wariwari group export GROUP_TOKEN --output trip.csv
wariwari balance GROUP_TOKEN --json
wariwari participant balance PARTICIPANT_TOKEN GROUP_TOKEN --json
wariwari participant list GROUP_TOKEN --json
wariwari category list GROUP_TOKEN --json
wariwari transaction list GROUP_TOKEN --json
wariwari transaction show TRANSACTION_TOKEN GROUP_TOKEN --json
wariwari transaction activity TRANSACTION_TOKEN GROUP_TOKEN --json
wariwari commands --json
```

### Transactions, filtered and paged

`transaction list` reads the group's transactions newest-first. Filters combine
and narrow — a well-formed token matching nothing gives an empty page, while a
malformed date is `400 INVALID_REQUEST`.

```bash
# January expenses in the Food category that Ada paid or owes a share of
wariwari transaction list GROUP_TOKEN \
  --from 2026-01-01 --to 2026-01-31 \
  --category CATEGORY_TOKEN \
  --participant PARTICIPANT_TOKEN \
  --take 50 --json
```

Get the category and participant tokens from the snapshot's `categories`
and `participants`.

### Pagination

`group list`, `transaction list`, and both activity feeds are keyset paginated.
Each response carries:

```json
{ "pagination": { "hasMore": true, "cursor": "eyJraW5kIjoia2V5c2V0In0" } }
```

Pass that value back as `--cursor` to get the next page; stop when `hasMore` is
`false`. `--take` caps at 100 and defaults to 50. The `--json` envelope also
emits a ready-to-run `next-page` breadcrumb that preserves filters, archived
mode, and page size.

Transaction reads always include `timeOfDay`, but it can be `null` for an import
whose source had no time. Do not interpret that as midnight.
`paidByParticipantToken` can also be `null` on historical transactions whose
payer was removed. Every transaction carries `hasReceiptPhoto`.

### Balances

`wariwari balance GROUP_TOKEN --json` pulls the balance view out of the
snapshot:

- `netByCurrency` — currency code → participant token → signed cents. Positive
  means that participant is owed money; negative means they owe.
- `suggestedPayments` — the minimal settle-up set, each with
  `fromParticipantToken`, `toParticipantToken`, `amountCents`, `currency`.
- `spendingByCurrency`, `totalBaseCents` — group spending totals.
- `participants` — so you can turn tokens into names.

To explain one person's balance, `wariwari participant balance PARTICIPANT_TOKEN
GROUP_TOKEN --json` returns `balanceCents` in the group currency and four
`lines` (`paid_for_others`, `paid_for_you`, `sent`, `received`), each listing
the transaction tokens behind it. `otherCurrencies: true` means some of their
transactions are kept in another currency and left out.

### Stats, exchange rates, export

- `group stats` returns totals, `categoryBreakdown`, `perPersonCents`, a
  `timeline` and `memberTotals` in minor units. Filter with `--date-range`
  (`all_time`, `this_month`, `last_month`, `last_3_months`, `last_6_months`,
  `last_12_weeks`, `this_year`), `--participant TOKEN`, `--category TOKEN`,
  `--month YYYY-MM`, `--week DATE`, or `--date DATE`.
- `group fx-rate --from USD [--date DATE]` returns `rate`, a decimal string,
  from that currency to the group currency on that day (default today), and
  `provisional` while the day's rate is not yet published.
- `group export` writes the group's `wariwari.v1` CSV to stdout, or to
  `--output FILE`. It needs settings access.

## Writing

Write commands take a JSON object via `--data` or `--data-file`. A create or
update returns the record it wrote; a delete returns nothing, and the CLI
prints its own `{"deleted": true}`. Only `group show` returns the snapshot.

```bash
wariwari group update GROUP_TOKEN --lock-version 1 --data '{"name":"Trip 2026"}' --json
wariwari group update GROUP_TOKEN --lock-version 2 --data '{"visibility":"private_registered"}' --json
wariwari participant create GROUP_TOKEN --data '{"name":"Ada","color":"#1F3A5F"}' --json
wariwari participant update PARTICIPANT_TOKEN GROUP_TOKEN --lock-version 0 --data '{"name":"Ada Lovelace"}' --json
wariwari participant update PARTICIPANT_TOKEN GROUP_TOKEN --lock-version 1 --data '{"membershipRole":"admin"}' --json
wariwari participant delete PARTICIPANT_TOKEN GROUP_TOKEN --lock-version 0 --json
wariwari participant merge DUPLICATE_TOKEN GROUP_TOKEN --into KEEP_TOKEN --json
wariwari participant move PARTICIPANT_TOKEN GROUP_TOKEN --position 1 --json
wariwari participant claim PARTICIPANT_TOKEN GROUP_TOKEN --json    # "this one is me"
wariwari category create GROUP_TOKEN --data '{"name":"Food","emoji":"🍜","position":1}' --json
wariwari category update CATEGORY_TOKEN GROUP_TOKEN --lock-version 0 --data '{"name":"Food","emoji":"🍜"}' --json
wariwari category delete CATEGORY_TOKEN GROUP_TOKEN --lock-version 0 --json
wariwari transaction create GROUP_TOKEN --data-file transaction-create.json --json
wariwari transaction update TRANSACTION_TOKEN GROUP_TOKEN --lock-version 2 --data-file transaction-update.json --json
wariwari transaction delete TRANSACTION_TOKEN GROUP_TOKEN --lock-version 2 --json
wariwari transaction bulk-edit GROUP_TOKEN --data '{"transactions":[{"token":"T1"},{"token":"T2"}],"categoryToken":"C"}' --json
wariwari transaction bulk-delete GROUP_TOKEN --data '{"transactions":[{"token":"T1","expectedRevision":2},{"token":"T2"}]}' --json
wariwari group archive GROUP_TOKEN --json       # only hides it for you
wariwari group leave GROUP_TOKEN --json         # stop being a member; your participant stays as a guest
wariwari group duplicate GROUP_TOKEN --include-members --json   # new group, same categories
wariwari group notifications mute GROUP_TOKEN --json
wariwari group delete GROUP_TOKEN --lock-version 3 --json   # deletes it for everyone
wariwari account update --data '{"locale":"ja"}' --json
```

Confirm with the user before any delete, merge or bulk edit. `bulk-edit`
applies `paidByParticipantToken`, `categoryToken`, `date`, `timeOfDay` or
`split` (`{"type": "equal|percentage|shares", "members": [{"participantToken",
"weight"}], "newMemberNames": [...]}`) to every listed transaction, all or
none, and returns the changed `transactions`. Bulk edit and bulk delete are
behind a feature flag and answer `404` while it is off.

`account update` takes a flat, partial body: `name`, `avatarColor`,
`preferredCurrency`, `locale` (`en`, `es`, `ja`), `marketingOptIn`,
`notificationPreferences`.

`group update` changes `name`, `icon`, `currency` (existing transactions are
rebased in the background) and `visibility` (`public_editable`,
`public_view_only`, `private_registered`). `participant update` renames or
recolours a guest, or sets the `membershipRole` (`read_only`, `editor`,
`admin`) of a member linked to an account; a linked member's name and colour
come from their account. A participant's `color` is an uppercase hex string
from the server's palette (`#1F3A5F`). Categories are edited one at a time with
`category update`.

### lockVersion is required on every update and delete

Pass the `lockVersion` from the record you just read as `--lock-version` for
updates, deletes and merges. It never goes in the body. Bulk edits and deletes
take it per transaction as `expectedRevision`. Without `--lock-version` (or
`expectedRevision`) the CLI reads the current value first and acts on whatever
is there, so pass it whenever you decided based on what you read. Creates,
reorders and recurring expenses take none. Retrying a delete that already
succeeded succeeds again. If someone else wrote first, the API returns
`409 CONFLICT` and, for groups, categories, transactions and participants, puts
the fresh record under `current`, keyed by its kind:

```json
{
  "code": "CONFLICT",
  "detail": "This transaction changed since you read it.",
  "status": 409,
  "current": { "transaction": { "token": "…", "lockVersion": 4, "amountCents": 13200 } }
}
```

Recover by re-reading `current` (or the record itself when `current` is
absent), re-applying your change on top of it, and retrying with the newer
`lockVersion` — do not blind-retry the same body.

### Creating a group

```bash
wariwari group create --idempotency-key "$(uuidgen)" --json \
  --data '{"name":"Trip","icon":"🧳","currency":"JPY","participants":[{"name":"Ada","color":"#1F3A5F"},{"name":"Grace"}]}'
```

`name`, `icon`, and at least one participant are required. Optional:
`creatorParticipantToken` (which participant is the user; they become its
admin), `categories` (`token`, `emoji`, `name`, `position`) to replace the
defaults, and client `token`s. A developer key may create 20 groups an hour.
The response is the group without its participants; `wariwari group show`
reads them.

### Retrying a write

Every write under a group, and `group create`, takes `--idempotency-key`. Pass
a fresh UUID, and reuse the same key and body when retrying: the server replays
the first outcome for 90 days. Reusing the key with a different body returns
`409 IDEMPOTENCY_KEY_IN_USE`; a retry older than 90 days returns
`409 IDEMPOTENCY_RECEIPT_EXPIRED`, so read the group before writing again.

## Pro workflows

Reminders, recurring expenses, receipt photo uploads and shared files need Pro
on the signed-in account (the account's `pro`). Everyone in the group sees what
a Pro member adds; reminders and recurring expenses pause (`paused`,
`ownerName`) while the member who set them up has no Pro or has left. Without it these answer `403 PRO_REQUIRED` with an `upgradeUrl`;
pass that URL to the user rather than retrying.

```bash
wariwari remind GROUP_TOKEN --json                                # everyone who owes money
wariwari remind GROUP_TOKEN --participant PARTICIPANT_TOKEN --json
wariwari reminder schedule show GROUP_TOKEN --json
wariwari reminder schedule weekly GROUP_TOKEN --weekday mon --json
wariwari reminder schedule monthly GROUP_TOKEN --day 25 --json
wariwari reminder schedule off GROUP_TOKEN --json
wariwari recurring list GROUP_TOKEN --json
wariwari recurring create GROUP_TOKEN --data-file rent.json --every monthly --starts 2026-12-01 --json
wariwari recurring update RECURRING_TOKEN GROUP_TOKEN --interval 2 --json
wariwari recurring delete RECURRING_TOKEN GROUP_TOKEN --json
wariwari transaction receipt show TRANSACTION_TOKEN GROUP_TOKEN --json
wariwari transaction receipt upload TRANSACTION_TOKEN receipt.jpg GROUP_TOKEN --json
wariwari transaction receipt delete TRANSACTION_TOKEN GROUP_TOKEN --json
wariwari file list GROUP_TOKEN --json
wariwari file upload photo.jpg GROUP_TOKEN --json
wariwari file delete FILE_ID GROUP_TOKEN --json
```

`remind` without `--participant` reminds everyone who owes and whom you may
remind; the rest come back under `skipped` with a `reason` (`unlinked`,
`opted_out`, `recently_reminded`). Each member is reminded at most once a day:
`--participant` for someone already reminded today answers
`409 ALREADY_REMINDED`. A
recurring expense's body is the expense fields of `transaction create`
(`description`, `amountCents`, `currency`, `paidByParticipantToken`,
`splitType`, `splits`, optional `categoryToken`) without `kind` or `date`;
`--every weekly|monthly|yearly`, `--interval N` and `--starts YYYY-MM-DD` set
`cadence`, `interval` and `startsOn`. A schedule has no `lockVersion`: the last
write wins. While its owner has no Pro a schedule is
`paused`: occurrences are skipped, not saved for later. Deleting a receipt
photo or a file works without Pro, and so does reading them.

## Settle-up workflow

1. `wariwari balance GROUP_TOKEN --json` and read `suggestedPayments`.
2. For each payment the user confirms, record it with a fresh
   `--idempotency-key`:

```bash
wariwari payment create GROUP_TOKEN --idempotency-key "$KEY" --json --data '{
  "fromParticipantToken": "8mPNkR4qQXtLbJR5CcxT6yWa",
  "toParticipantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
  "amountCents": 4000,
  "date": "2026-05-24"
}'
```

`currency` defaults to the group currency. A payment is a transaction of kind
`payment`; `payment create` and `payment update` set `kind` for you. Read it
with `transaction show`, delete it with `transaction delete`.
`payment update PAYMENT_TOKEN GROUP_TOKEN --lock-version N` takes only the
fields that change. The response is the payment, so run `wariwari balance`
again to see what is left to settle.

## Sharing and access workflow

- Public group: give people the snapshot's `shareUrl`.
- View-only or private group: `wariwari invitation create GROUP_TOKEN --role
  editor --expires-at 2026-12-31T23:59:00+09:00 --json` returns the
  invitation's `url` and `token`;
  `invitation list GROUP_TOKEN` shows the ones still usable; `invitation revoke
  INVITATION_TOKEN GROUP_TOKEN` disables one. `--role` is `read_only`, `editor`,
  or `admin`; `--expires-at` is an ISO 8601 date-time in the future, and
  omitting it makes a link that never expires. Admin-only.
- Accepting an invitation: `wariwari invitation accept INVITATION_TOKEN
  GROUP_TOKEN`. Joining a public group by its link: `wariwari group join
  GROUP_TOKEN`. Both return the participant you are now linked to. Leaving:
  `wariwari group leave GROUP_TOKEN`.
- Joining someone else's private group: `wariwari access-request status
  GROUP_TOKEN` (`none`, `pending`, `declined`, `viewable`), then
  `access-request create GROUP_TOKEN`. Asking again while pending, or within 7
  days of a decline, changes nothing.
- Reviewing requests (admins): `access-request list GROUP_TOKEN` returns
  `accessRequests` with numeric `id`s; `access-request approve ID GROUP_TOKEN`
  adds the requester as an editor, `access-request decline ID GROUP_TOKEN`
  turns them down without notifying them.

## Expense creation workflow

1. Find the group: `wariwari group list --json`, or use the configured default.
2. Read current state: `wariwari group show GROUP_TOKEN --json`.
3. Take existing participant tokens from `participants` and category tokens
   from `categories`. **Never invent a token for an existing resource.** A
   category is always one of the group's, named by `categoryToken`; there is no
   free-text category. Add a missing one with `wariwari category create`.
4. Generate a UUID for `--idempotency-key`. Reuse the identical key and body
   if the create must be retried.
5. Send `"kind": "expense"`; it is required on create and update and cannot
   change. A settle-up transfer is `wariwari payment create`.
6. Pick `splitType`: `equal`, `percentage`, `shares`, or `amount`.
7. Build `splits` so the `amountCents` values sum to the transaction's
   `amountCents`. For `equal`, you may omit `amountCents` from every split and
   the server divides the amount evenly. Optional: `categoryToken`,
   `timeOfDay` (`HH:MM`), and `balanceCurrency` (send the transaction currency
   to keep a foreign expense in its own balance bucket instead of converting).
8. Create it; the response is the transaction. Run `wariwari balance
   GROUP_TOKEN --json` to report the new balances.

```bash
wariwari transaction create GROUP_TOKEN --idempotency-key "$KEY" --json --data '{
  "kind": "expense",
  "description": "Team dinner",
  "categoryToken": "4QfYk8nE2pRtVwXz7aBcDeFg",
  "amountCents": 12000,
  "currency": "JPY",
  "splitType": "equal",
  "date": "2026-05-23",
  "timeOfDay": "19:30",
  "paidByParticipantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj",
  "splits": [
    {"participantToken": "7gKMjL9pPWsKaHQ4AbwS5kVj"},
    {"participantToken": "8mPNkR4qQXtLbJR5CcxT6yWa"},
    {"participantToken": "9nQPmS5rRYuMcKS6DdyU7zXb"}
  ]
}'
```

To edit: `wariwari transaction show TOKEN GROUP_TOKEN --json`, then `wariwari
transaction update TOKEN GROUP_TOKEN --lock-version N` with `"kind":
"expense"` and the changed fields. Sending `splits` replaces all existing
splits; `"categoryToken": null` clears the category.

## Errors

Every failure is one Problem body: `code`, `title`, `detail`, `status` and
`instance` (the request id to quote to support). Switch on `code`, show
`detail`, and treat a code you don't know by its HTTP status:

| Code                                              | Status | What to do                                          |
| ------------------------------------------------- | ------ | --------------------------------------------------- |
| `AUTH_CREDENTIALS_REQUIRED` / `AUTH_CREDENTIALS_INVALID` | 401 | Ask for a key and run `wariwari auth login wari_...`. |
| `ACCESS_DENIED`                                   | 403    | Session-only action the key cannot do.              |
| `INSUFFICIENT_PERMISSIONS`                        | 403    | No edit access, or an admin-only action.            |
| `PRO_REQUIRED`                                    | 403    | Needs Pro. Show `detail` and `upgradeUrl`.          |
| `NOT_FOUND`                                       | 404    | Wrong token, or the record was deleted.             |
| `INVALID_REQUEST`                                 | 400    | Malformed date, cursor, or body.                    |
| `VALIDATION_FAILED`                               | 400    | Read `errors`: field paths (`splits/0/amountCents`), header names, or `base`. |
| `CONFLICT`                                        | 409    | Stale `lockVersion` — reconcile against `current`.  |
| `IDEMPOTENCY_KEY_IN_USE`                          | 409    | Same key, different body. Use a fresh key.          |
| `IDEMPOTENCY_RECEIPT_EXPIRED`                     | 409    | Retry older than 90 days. Read the group first.     |
| `ALREADY_REMINDED`                                | 409    | That member was reminded today. Do not retry.       |
| `DUPLICATE_TOKEN`                                 | 409    | Client token already used. Read the snapshot.       |
| `RATE_LIMIT_EXCEEDED`                             | 429    | Back off for `retryAfter` seconds.                  |
| `QUOTA_EXCEEDED`                                  | 429    | Free key quota used. Show `detail` and `upgradeUrl`.|
| `STORAGE_LIMIT`                                   | 413    | The upload would pass the uploader's storage limit. |
| `INTERNAL_SERVER_EXCEPTION`                       | 500    | Retry later; report `instance` if it persists.      |

Note `VALIDATION_FAILED` is a `400`, not a `422`.

## Payload templates

```bash
wariwari group example create --json
wariwari group example update --json
wariwari participant example create --json
wariwari participant example update --json
wariwari category example create --json
wariwari category example update --json
wariwari transaction example create --json
wariwari transaction example update --json
wariwari transaction example bulk-edit --json
wariwari payment example create --json
wariwari payment example update --json
wariwari recurring example create --json
```

## Feedback

`wariwari feedback "message"` sends a note to the WariWari team; replies go to
the account's email. Only send feedback the user asked you to send.
