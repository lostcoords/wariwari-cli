---
name: wariwari
description: Use the WariWari CLI to manage WariWari groups, participants, categories, transactions, balances, and account API access from an agent.
---

# WariWari CLI

Use `wariwari` for WariWari API work. Prefer JSON output for agent workflows:

```bash
wariwari <command> --json
```

## Setup

```bash
wariwari setup
wariwari setup claude
wariwari doctor
wariwari auth status
```

Configuration precedence:

1. CLI flags: `--token`, `--profile`, `--api-url`, `--group`
2. Environment: `WARIWARI_TOKEN`, `WARIWARI_PROFILE`, `WARIWARI_API_URL`, `WARIWARI_GROUP`
3. Named profiles in `~/.config/wariwari/config.json`
4. Local `.wariwari.yaml`
5. Global `~/.config/wariwari/config.yaml`

## Common Commands

```bash
wariwari me --json
wariwari group list --json
wariwari group ls --json
wariwari group show GROUP_TOKEN --json
wariwari group view GROUP_TOKEN --json
wariwari group snapshot GROUP_TOKEN --json
wariwari group activity GROUP_TOKEN --json
wariwari balance GROUP_TOKEN --json
wariwari participant list GROUP_TOKEN --json
wariwari category list GROUP_TOKEN --json
wariwari transaction list GROUP_TOKEN --json
wariwari transaction show TRANSACTION_TOKEN GROUP_TOKEN --json
wariwari transaction view TRANSACTION_TOKEN GROUP_TOKEN --json
wariwari transaction activity TRANSACTION_TOKEN GROUP_TOKEN --json
wariwari token list --json
wariwari token create --name "Automation" --json
wariwari token delete ID --json
wariwari commands --json
```

Create/update commands accept a JSON object:

```bash
wariwari group create --data '{"name":"Trip","currency":"JPY"}' --idempotency-key "$(uuidgen)" --json
wariwari participant upsert PARTICIPANT_TOKEN GROUP_TOKEN --data '{"name":"Ada"}' --json
wariwari category upsert CATEGORY_TOKEN GROUP_TOKEN --data '{"name":"Food","emoji":"F"}' --json
wariwari transaction upsert TRANSACTION_TOKEN GROUP_TOKEN --data-file transaction.json --json
```

For existing upserts, include `lock_version` from the current server record. A stale version returns `409 CONFLICT` with the current record in the error body.
