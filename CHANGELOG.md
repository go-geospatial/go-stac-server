# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Fixed parsing of `sortBy` field in search POST body when `sortBy` is a string

## [v0.3.0] - 2023-08-01

### Added

- Add option to set map CRS EPSG:4326 with `crs` option in stac browser configuration

## [v0.2.0] - 2023-07-30

### Added

- Added ability to configure basemaps in stac-browser

## [v0.1.0] - 2023-07-28

### Added

- Implement STAC core API functions
- Browseable extension
- Filter extension
- Query extension
- Sort extension
- Transaction extension

[unreleased]: https://github.com/olivierlacan/keep-a-changelog/compare/v0.3.0...HEAD
[v0.3.0]: https://github.com/olivierlacan/keep-a-changelog/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/olivierlacan/keep-a-changelog/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/go-geospatial/go-stac-server/releases/tag/v0.1.0
