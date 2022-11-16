# Changelog

## [Unreleased]

### Added

- JSON result will have `ok` boolean as a helper, it will be `true` if there is any drift or drift detection failed, `false` otherwise.

## [v0.2.0] - 2022-11-15

### Added

- Include and exclude name repeatable flags, now can use commas to split a single argument.
- Include and exclude tag repeatable flags, now can use commas to split a single argument.
- Add `pretty-json` as indented JSON output format.
- Add `json` as non-indented JSON output format.

### Changed

- `json` output format now is non-indented JSON.

## [v0.1.0] - 2022-11-14

### Added

- Initial single `run` mode.
- Result output in JSON.
- Filter by workspaces name regex (include and exclude).
- Filter by workspace tag (include and exclude).
- Filter already queued and running drift detections.
- Filter by last drift detection.
- Sort by priority: Oldest drift detections, first to be executed.
- Limit the number of drift detections to execute.
- Different exit codes depending on the result of the run.
- Dry-run mode.

[unreleased]: https://github.com/slok/tfe-drift/compare/v0.2.0...HEAD
[v0.2.0]: https://github.com/slok/tfe-drift/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/slok/tfe-drift/releases/tag/v0.1.0
