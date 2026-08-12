# Security Policy

## Reporting a Vulnerability

Please **do not** open a public issue for security vulnerabilities.

Instead, report them privately to the repository maintainers. For private
disclosure, open a GitHub Security Advisory on this repository:

<https://github.com/duncai233/xiaoheiFS/security/advisories/new>

When reporting, include:

- The affected component and version (or commit SHA).
- A description of the vulnerability and its impact.
- Reproduction steps or a proof of concept, if available.
- Any suggested fix, if you have one.

You should receive an acknowledgement within 5 business days. We ask that you
do not publicly disclose the issue until a fix has been released and this
policy has been updated.

## Scope

This policy applies to the xiaoheiFS repository: the Go backend, the Flutter
clients, the web frontend, the qz-override reference module, plugin SDKs, and
the build/release/container tooling in this repository.

Vulnerabilities in upstream dependencies should be reported to their respective
projects.

## Security Best Practices

- Secrets must never be committed. If you discover a live credential in the
  repository or its history, treat it as compromised: rotate it immediately and
  open a Security Advisory.
- CI workflows pin every third-party action to a full commit SHA and run with
  least privilege.
- Production releases must not bundle test-only plugins (e.g. `mockpay`).
- Flutter clients store credentials in the platform keystore via
  `flutter_secure_storage`; plaintext HTTP is disabled in release builds.
- Flutter app updates are only offered from Ed25519-signed manifests; the
  signing private key must live only in the release operator's secrets manager
  (never the repo or client), and must be rotated on any suspected leak. See
  `docs/update-signing-contract.md`.

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | :white_check_mark: |
| older   | :x:                |
