# Changelog

## [Unreleased]

### Added

- Allow specifying the number of workers that will be used concurrently to fetch the workspaces data. 

### Changed

- On controller mode, the metrics exporter workspace workspace retrieval has changed to async mode being updated at regular intervals.

## [v0.5.0] - 2022-12-11

BREAKING: Prometheus metrics labels `workspaces_name` and `workspaces_id` have been renamed.

### Added

- `organization_name` on workspace Prometheus metric workspace info.

### Changed

- The Prometheus metric label `workspaces_name` now is `workspace_name`.
- The Prometheus metric label `workspaces_id` now is `workspace_id`.

## [v0.4.0] - 2022-11-27

### Added

- JSON result will have the plan run duration on `run_duration` field.
- `controller` mode, this will run the drift detector as a controller running the drift detections.
- Prometheus drift-detection metrics exporter on the `controller` mode to be able to export drift-detection metrics.
- Able to disabled controller in `controller` mode to only run the metrics exporter if required.

## [v0.3.0] - 2022-11-18

### Added

- JSON result will have `ok` boolean as a helper, it will be `true` if there is any drift or drift detection failed, `false` otherwise.
- JSON result will have the workspace tags as a helper to grab information.

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

[unreleased]: https://github.com/slok/tfe-drift/compare/v0.5.0...HEAD
[v0.5.0]: https://github.com/slok/tfe-drift/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/slok/tfe-drift/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/slok/tfe-drift/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/slok/tfe-drift/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/slok/tfe-drift/releases/tag/v0.1.0
