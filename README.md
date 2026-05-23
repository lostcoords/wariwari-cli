# WariWari CLI

`wariwari` is a command-line interface for WariWari. It follows the same shape as Fizzy CLI: token authentication, named profiles, JSON envelopes, breadcrumbs for next commands, an embedded agent skill, and Claude plugin metadata.

## Quick Start

```bash
make build
./bin/wariwari setup
./bin/wariwari doctor
./bin/wariwari group list --json
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
6. Defaults, including `https://wariwari.jp`

Credentials are stored in the OS keyring by default. Set `WARIWARI_NO_KEYRING=1` to force file fallback.

## Commands

```bash
wariwari me
wariwari group list
wariwari group show GROUP_TOKEN
wariwari group view GROUP_TOKEN
wariwari group snapshot GROUP_TOKEN
wariwari group activity GROUP_TOKEN
wariwari balance GROUP_TOKEN
wariwari participant list GROUP_TOKEN
wariwari category list GROUP_TOKEN
wariwari transaction list GROUP_TOKEN
wariwari transaction show TRANSACTION_TOKEN GROUP_TOKEN
wariwari transaction view TRANSACTION_TOKEN GROUP_TOKEN
wariwari transaction activity TRANSACTION_TOKEN GROUP_TOKEN
wariwari token list
wariwari token create --name "Automation"
wariwari token delete ID
```

Write commands accept JSON:

```bash
wariwari group create --data '{"name":"Trip","currency":"JPY"}' --idempotency-key "$(uuidgen)"
wariwari group update GROUP_TOKEN --data '{"name":"Trip 2026","lock_version":1}'
wariwari participant upsert PARTICIPANT_TOKEN GROUP_TOKEN --data '{"name":"Ada","lock_version":0}'
wariwari category upsert CATEGORY_TOKEN GROUP_TOKEN --data '{"name":"Food","emoji":"F","lock_version":0}'
wariwari transaction upsert TRANSACTION_TOKEN GROUP_TOKEN --data-file transaction.json
```

## Agent Integration

```bash
wariwari skill
wariwari skill install
wariwari setup claude
```

The embedded skill installs to `~/.agents/skills/wariwari/SKILL.md`.
`wariwari setup claude` also links it into `~/.claude/skills/wariwari`.

## Development

```bash
make build
make test-unit
make check
```
