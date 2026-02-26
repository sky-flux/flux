# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x     | Yes       |
| < 1.0   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public GitHub issue.
2. Email your report to the maintainers via [GitHub Security Advisories](https://github.com/sky-flux/flux/security/advisories/new).
3. Include a clear description of the issue, reproduction steps, and any potential impact.

We aim to acknowledge reports within 48 hours and provide a fix or mitigation plan within 7 days.

## Scope

flux is a pure computation library with zero external dependencies and no network, filesystem, or OS access. The primary security considerations are:

- **Parameter validation**: All 21 FSRS parameters are bounds-checked before use.
- **Input validation**: Invalid ratings, mismatched card IDs, and malformed data are rejected with explicit errors.
- **No unsafe operations**: The library uses only safe Go standard library functions.
