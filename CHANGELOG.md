# Changelog

All notable changes to dbminer will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.1] - 2026-07-20

### Added

- `-group-by` flag for flexible table grouping in markdown output:
  - `auto` (default): Auto-detect delimiter from table names
  - `prefix`: Group by underscore-separated prefix
  - `schema`: Group by database schema name
  - `none`: No grouping, single alphabetical list
- `-version` flag to print version and exit
- Auto-detection of table naming convention (`.`, `_`, `-` delimiters)

### Changed

- Version now configurable at build time via `-ldflags "-X main.version=x.y.z"`

---

## [0.9.0] - 2026-07-20

### Added

- Initial release of dbminer multi-database schema documentation generator
- SQL script generation for 9 database platforms:
  - MySQL 8.0+ and MySQL 5.7
  - MariaDB 10.2+
  - PostgreSQL
  - SQLite
  - Microsoft SQL Server 2017+ and SQL Server 2016
  - Oracle Database
  - MongoDB 4.4+ (JavaScript-based schema inference)
- Schema processing from SQL collector JSON output
- Export formats:
  - JSON (ChartDB-compatible)
  - NDJSON (newline-delimited JSON for LLM context)
  - TSV (combined and split modes for spreadsheet analysis)
- Markdown documentation generation:
  - Schema README with table of contents and domain grouping
  - Per-table documentation with columns, indexes, relationships
  - Mermaid ERD diagrams (domain-level and per-table)
- Support for all major database entities:
  - Tables and views
  - Columns with types, nullability, defaults, comments
  - Primary keys and unique constraints
  - Foreign key relationships
  - Indexes (simple and composite)
  - Triggers
  - Stored procedures and functions
  - Generated/computed columns
- Input validation for SQL identifiers to prevent injection
- Test suite with 85%+ code coverage using realistic GoAnywhere MFT data

### Notes

- This is a pre-release version (0.9.0) - API may change before 1.0.0
- Tested primarily with GoAnywhere MFT MySQL database (210 tables)
- Windows and Linux compatible
