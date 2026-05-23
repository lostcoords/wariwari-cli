# Security Policy

Report WariWari CLI security issues privately to the project maintainers. Do not open public issues for token leaks, authentication bypasses, or API access-control defects.

## Credential Storage

The CLI stores API tokens in the operating system keyring when available:

| Platform | Storage |
| --- | --- |
| macOS | Keychain |
| Windows | Credential Manager |
| Linux | Secret Service |

Set `WARIWARI_NO_KEYRING=1` to use file fallback under `~/.config/wariwari/credentials/`. Fallback files are written with `0600` permissions.

