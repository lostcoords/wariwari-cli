# WariWari CLI Development Context

This repository mirrors the Fizzy CLI pattern for WariWari.

## Structure

```
wariwari-cli/
├── cmd/wariwari/       # Main entrypoint
├── internal/config/    # Config files, profiles, credential storage
├── internal/client/    # Direct WariWari HTTP API client
├── internal/commands/  # Cobra command implementations
├── skills/             # Embedded agent skill
└── .claude-plugin/     # Claude Code plugin metadata
```

## API

The CLI talks to WariWari's `/v1` JSON API with `Authorization: Bearer <token>`.

Key routes:

- `GET /v1/me`
- `GET /v1/groups`
- `POST /v1/groups`
- `GET /v1/groups/:group_token`
- `PUT /v1/groups/:group_token`
- `GET /v1/groups/:group_token/snapshot`
- `GET /v1/groups/:group_token/balances`
- `GET /v1/groups/:group_token/participants`
- `PUT /v1/groups/:group_token/participants/:participant_token`
- `DELETE /v1/groups/:group_token/participants/:participant_token`
- `GET /v1/groups/:group_token/categories`
- `PUT /v1/groups/:group_token/categories/:category_token`
- `DELETE /v1/groups/:group_token/categories/:category_token`
- `GET /v1/groups/:group_token/transactions`
- `GET /v1/groups/:group_token/transactions/:transaction_token`
- `PUT /v1/groups/:group_token/transactions/:transaction_token`
- `DELETE /v1/groups/:group_token/transactions/:transaction_token`
- `GET /v1/groups/:group_token/transactions/:transaction_token/activity`
- `GET /v1/api_keys`
- `POST /v1/api_keys`
- `DELETE /v1/api_keys/:id`

## Conventions

- Keep command behavior thin and predictable.
- Put config and credential behavior in `internal/config`.
- Keep HTTP mechanics in `internal/client`.
- Return JSON envelopes by default, and raw data with `--quiet`.
- Preserve Fizzy's setup pattern: `setup`, `setup claude`, `doctor`, `auth`, `config`, `skill`, `commands`.
