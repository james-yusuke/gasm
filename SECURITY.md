# Security Policy

## Supported Versions

gasm is a learning project and does not currently publish stable release lines. Security fixes are handled on the default branch.

## Reporting a Vulnerability

Please do not open a public issue for a suspected security problem.

Instead, contact the maintainer privately with:

- A short description of the issue.
- Steps to reproduce it.
- The affected file, command, or input if known.
- Any suggested fix or mitigation.

The project is educational, so the expected response is best-effort. Reports that help improve parser safety, malformed-input handling, generated binary correctness, or dependency hygiene are especially welcome.

## Scope

This project should not be used to assemble or run untrusted code in production environments. Treat generated binaries and assembly input as unsafe unless you have reviewed them yourself.

