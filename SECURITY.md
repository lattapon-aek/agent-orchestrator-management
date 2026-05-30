# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release (`main`) | Yes |
| Older tags | No — please upgrade |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report security issues by emailing **lattapon.kea@dohome.co.th** with the subject line `[AOM Security]`.

Include:
- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof-of-concept (if available)
- Any suggested remediation

You can expect an acknowledgement within **3 business days** and a resolution timeline within **14 days** for confirmed issues.

## Scope

AOM is a local CLI tool. Its attack surface is limited to:
- The local filesystem (`.aom/` directory, SQLite database, git worktrees)
- Subprocess execution (tmux, git, AI agent CLIs)
- Environment variables read at runtime (`AOM_ACTOR`, `GOPATH`, etc.)

Shell hooks (`.aom/hooks/*.sh`) run with the operator's full user privileges. Only grant hook scripts to trusted code.
