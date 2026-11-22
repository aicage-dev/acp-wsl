# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.3] - 2026-11-22

### Added

- ACP v1 path translation for Markdown plan files, subagent workspaces, and session message content.

### Changed

- Updated supported ACP schemas to v1.34.1 and v2.0.0-beta.3.
- Migrated Markdown lint configuration to JSON.

## [1.0.2] - 2026-11-22

### Added

- ACP v2 path translation for session options, skill commands, subagent workspaces, and Markdown plan files.

### Changed

- Updated supported ACP schemas to v1.33.0 and v2.0.0-alpha.17.

## [1.0.1] - 2026-11-22

### Changed

- Improved automated update detection for ACP schema dependencies.
- Expanded version-specific test coverage for ACP path routing.
- Updated CI linting configuration and GitHub Actions dependencies.

## [1.0.0] - 2026-11-01

### Added

- ACP stdio proxy that translates filesystem paths between Windows and WSL.
- Support for ACP v1 and v2 schema-aware path mapping, with optional mapping of absolute paths in unknown fields.
- `acp-wsl-debug` command for logging complete ACP traffic and path-mapping decisions.
- Automated release artifacts for Windows and validated WinGet manifests.
