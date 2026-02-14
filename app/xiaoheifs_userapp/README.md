# opensource_userapp

A new Flutter project.

## Getting Started

This project is a starting point for a Flutter application.

A few resources to get you started if this is your first Flutter project:

- [Lab: Write your first Flutter app](https://docs.flutter.dev/get-started/codelab)
- [Cookbook: Useful Flutter samples](https://docs.flutter.dev/cookbook)

For help getting started with Flutter development, view the
[online documentation](https://docs.flutter.dev/), which offers tutorials,
samples, guidance on mobile development, and a full API reference.

## Build with Git Version

Use the root script to build `userapp` with version info derived from Git:

```powershell
# from d:\proj-netmc\app
.\build_userapp_from_git.ps1
```

Rules:
- `build-name` uses latest Git tag (for example `v1.2.0` -> `1.2.0`).
- `build-number` uses `git rev-list --count HEAD`.
- If no valid tag is found, `build-name` falls back to `pubspec.yaml` `version`.

More options:

```powershell
.\build_userapp_from_git.ps1 -Target appbundle
.\build_userapp_from_git.ps1 -SplitPerAbi
.\build_userapp_from_git.ps1 -DryRun
```
