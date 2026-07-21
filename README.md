# dbminer

**Multi-Database Schema Documentation Generator**

Version 0.9.0 | [Architecture](ARCHITECTURE.md) | [Testing](TESTING.md) | [Changelog](CHANGELOG.md)

---

## Application Overview

dbminer generates comprehensive database schema documentation without requiring direct database access. It uses a two-phase workflow:

1. **Generate SQL Script** - Create a database-specific export script
2. **Process Output** - Convert the SQL output to documentation or data formats

### Objectives

- Document database schemas across 9 database platforms
- Enable schema analysis without DBA privileges
- Support multiple output formats for different use cases:
  - **Markdown** - Human-readable documentation with ERD diagrams
  - **JSON** - ChartDB-compatible for visualization tools
  - **NDJSON** - LLM-friendly streaming format
  - **TSV** - Spreadsheet analysis and data integration

### Supported Databases

| Database | Driver | Version | Notes |
|----------|--------|---------|-------|
| MySQL | `mysql` | 8.0+ | CTEs with user variables |
| MySQL | `mysql57` | 5.7 | GROUP_CONCAT fallback |
| MariaDB | `mariadb` | 10.2+ | MySQL-compatible |
| PostgreSQL | `postgres` | 10+ | Full metadata support |
| SQLite | `sqlite` | 3.x | Direct query only (no stored procs) |
| SQL Server | `mssql` | 2017+ | STRING_AGG aggregation |
| SQL Server | `mssql2016` | 2016- | FOR XML PATH fallback |
| Oracle | `oracle` | 12c+ | ALL_* dictionary views |
| MongoDB | `mongodb` | 4.4+ | JavaScript schema inference |

---

## Security Assessment

### Summary

| Category | Risk | Status |
|----------|------|--------|
| Network Exposure | None | Offline CLI tool |
| Credential Handling | None | No secrets processed |
| Input Validation | Low | Regex-validated identifiers |
| File System Access | Low | User-specified paths only |
| Supply Chain | None | Zero external dependencies |
| Privilege Level | None | Runs as invoking user |

### Threat Model

**Attack Surface:** Minimal - standalone CLI with no network, no database connections, no user authentication.

| Threat | Mitigation |
|--------|------------|
| SQL Injection via generated scripts | All identifiers validated against strict regex before template substitution |
| Path Traversal | Output paths used as-is; OS validates; no path manipulation |
| Malicious JSON input | JSON parsed via stdlib; no code execution; data-only processing |
| Template Injection | Templates embedded at compile-time; user input never interpreted as template code |
| Denial of Service | No network; local resource limits apply |

### Encryption

| Layer | Status | Notes |
|-------|--------|-------|
| In Transit | N/A | No network communication |
| At Rest | N/A | No data persistence; output files inherit OS permissions |
| Secrets | N/A | No credentials handled by application |

Database credentials are provided by the DBA when executing generated scripts - dbminer never sees them.

### Secret Management

- **No secrets stored**: Application processes schema metadata only
- **No credentials in scripts**: Generated SQL contains database/procedure names, not passwords
- **No environment variables read**: No secret injection mechanism exists
- **No configuration files**: All options via CLI flags

### Authentication & Authorization

| Aspect | Implementation |
|--------|----------------|
| User Authentication | None required - local CLI tool |
| Service Authentication | None - no external services |
| Authorization Model | OS file permissions only |
| Session Management | None - stateless execution |

### RBAC (Role-Based Access Control)

- **Not Applicable**: Single-user CLI tool
- File permissions: 0644 for files, 0755 for directories (standard user ownership)
- No multi-user or role-based features

### Dependency Analysis

**External Dependencies: Zero**

dbminer uses only Go standard library packages, eliminating supply chain risk:

| Package | Purpose | Security Notes |
|---------|---------|----------------|
| `encoding/json` | JSON parsing | No arbitrary code execution |
| `text/template` | SQL generation | Templates embedded, not user-provided |
| `os` | File I/O | Paths validated by OS |
| `path/filepath` | Path handling | Cross-platform safe joins |
| `regexp` | Input validation | Compiled patterns, no ReDoS risk |
| `flag` | CLI parsing | Standard library |
| `fmt`, `strings`, `sort`, `time` | Utilities | No security surface |
| `embed` | Template embedding | Compile-time only |

**Verification:**
```bash
go mod tidy          # No external modules
govulncheck ./...    # No known vulnerabilities
```

### Privilege Requirements

| Requirement | Status |
|-------------|--------|
| Root/Admin | Not required |
| Network access | Not required |
| Database access | Not required (scripts executed separately) |
| File system | Write to user-specified output directory |

Runs entirely in unprivileged user context.

### Input Validation

All user-provided identifiers are validated before use in templates:

| Input | Pattern | Max Length | Rejection |
|-------|---------|------------|-----------|
| Database name | `^[a-zA-Z_][a-zA-Z0-9_]*$` | 128 | Hyphen, space, special chars |
| MongoDB name | `^[a-zA-Z_][a-zA-Z0-9_\-]*$` | 64 | Leading hyphen, special chars |
| Procedure name | `^[a-zA-Z_][a-zA-Z0-9_]*$` | 128 | Same as database |
| User name | `^[a-zA-Z_][a-zA-Z0-9_@.%'-]*$` | 128 | Semicolon, backslash, quotes |
| File paths | OS-validated | OS limit | Invalid path characters |

**Validation prevents:**
- SQL injection in generated scripts
- Command injection via identifiers
- Path traversal attacks

### Output Security

| Output Type | Security Consideration |
|-------------|----------------------|
| SQL scripts | Read-only after generation; DBA reviews before execution |
| JSON/NDJSON | Data-only; no executable content |
| TSV | Plain text; no macros or formulas |
| Markdown | Static documentation; Mermaid diagrams are text-based |

Generated files contain schema metadata only - no credentials, no executable code.

### Known Limitations

1. **No output encryption**: Files written in plaintext (schema metadata is not sensitive)
2. **No integrity signing**: Output files not cryptographically signed
3. **No audit logging**: CLI tool does not maintain logs

These are appropriate for a local development/documentation tool.

---

## Code Quality Assessment

### Static Analysis

- **gosec**: All findings addressed with `#nosec` directives for intentional patterns
- **golangci-lint**: Clean pass with default rules
- **go vet**: No issues

### Best Practices

| Practice | Implementation |
|----------|---------------|
| Error Handling | All errors propagated with context via `fmt.Errorf("context: %w", err)` |
| Resource Cleanup | `defer` for file handles with error-capturing close pattern |
| Input Validation | Regex validation before any string interpolation |
| Separation of Concerns | I/O separated from transformation logic |
| Testability | 85.8% code coverage; pure functions for core logic |
| Documentation | Inline comments for non-obvious patterns only |

### Code Structure

```
dbminer/
├── main.go           # CLI entry point, flag handling, mode routing
├── go.mod            # Module definition (no external deps)
├── schema/           # Data types and transformations
│   ├── types.go      # Schema, Table, Field, Relationship types
│   ├── raw.go        # RawSchema types, LoadRaw()
│   ├── convert.go    # ConvertRawToSchema(), type detection
│   └── export.go     # JSON, NDJSON, TSV exporters
├── gensql/           # SQL script generation
│   ├── driver.go     # Driver interface, validation functions
│   ├── template.go   # Template execution
│   ├── mysql.go      # MySQL driver implementation
│   └── templates/    # Embedded SQL/JS templates
└── markdown/         # Documentation generation
    └── generator.go  # Markdown + Mermaid ERD generation
```

---

## Command Line Arguments

### Input/Output Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-raw` | string | - | Path to JSON file from SQL collector |
| `-output` | string | `./docs` | Output directory for Markdown files |
| `-export-format` | string | - | Export format: `json`, `ndjson`, `tsv` |
| `-o` | string | - | Output file for `-gensql` or `-export-format` |
| `-tsv-split` | bool | `false` | Split TSV into separate files per entity |

### SQL Generation Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-gensql` | bool | `false` | Generate SQL export script (mode selector) |
| `-driver` | string | `mysql` | Database driver (see supported list) |
| `-db` | string | `mydb` | Database name in generated script |
| `-sp` | string | `sp_exportschema` | Stored procedure name |
| `-user` | string | - | User to impersonate (optional) |
| `-direct` | bool | `false` | Generate direct query instead of stored procedure |
| `-noadmin` | bool | `false` | Skip admin-privileged queries |

### MongoDB-Specific Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-sample` | int | `100` | Documents to sample per collection |
| `-depth` | int | `2` | Nested object expansion depth |

---

## Usage Examples

### Workflow 1: Generate Markdown Documentation

**Step 1: Generate SQL script**
```bash
dbminer -gensql -driver mysql -db goanydb -o export_schema.sql
```

**Step 2: DBA executes script**
```bash
mysql -u dbadmin -p goanydb < export_schema.sql > schema.json
```

**Step 3: Generate documentation**
```bash
dbminer -raw schema.json -output ./docs
```

**Output:**
```
docs/
├── README.md           # Schema overview with ERD
└── tables/
    ├── dpa_web_user.md
    ├── dpa_project.md
    └── ... (one file per table)
```

### Workflow 2: Export for LLM Analysis

```bash
dbminer -raw schema.json -export-format ndjson -o schema.ndjson
```

**Sample NDJSON output:**
```json
{"_type":"metadata","name":"goanydb","databaseType":"mysql","tableCount":210,"relCount":154}
{"_type":"table","id":"1","name":"dpa_web_user","fields":[...]}
{"_type":"relationship","id":"1","sourceTableId":"5","targetTableId":"1",...}
```

### Workflow 3: Export for Spreadsheet Analysis

**Combined TSV:**
```bash
dbminer -raw schema.json -export-format tsv -o schema.tsv
```

**Split TSV (separate files per entity type):**
```bash
dbminer -raw schema.json -export-format tsv -o ./export -tsv-split
```

**Output:**
```
export/
├── tables.tsv        # id, schema, name, type, comment
├── columns.tsv       # id, table_id, name, type, is_pk, ...
├── indexes.tsv       # id, table_id, name, is_unique, columns
├── relationships.tsv # id, name, src_table, src_col, tgt_table, tgt_col
├── triggers.tsv      # (if triggers exist)
└── procedures.tsv    # (if stored procedures exist)
```

### Workflow 4: MongoDB Schema Inference

```bash
# Generate JavaScript schema inference script
dbminer -gensql -driver mongodb -db analytics -sample 500 -depth 3 -o infer_schema.js

# Run in MongoDB shell
mongosh analytics < infer_schema.js > schema.json

# Generate documentation
dbminer -raw schema.json -output ./docs
```

### Workflow 5: Direct SQL Query (No Stored Procedure)

```bash
# For environments where stored procedures cannot be created
dbminer -gensql -driver mysql -db mydb -direct -o query.sql
```

### Workflow 6: Limited Privileges Mode

```bash
# Skip queries requiring admin privileges (index sizes, row counts)
dbminer -gensql -driver postgres -db mydb -noadmin -o export.sql
```

---

## Sample Outputs

### Markdown README (excerpt)

```markdown
# goanydb Database Schema

**Database Type:** mysql
**Generated:** 2026-07-20 14:30:00
**Tables:** 210
**Relationships:** 154

---

## Schema Overview (by Domain)

\`\`\`mermaid
flowchart LR
    dpa_web[dpa_web<br/>45 tables]
    dpa_job[dpa_job<br/>32 tables]
    dpa_web -->|23| dpa_job
\`\`\`

---

## Table of Contents

### dpa_web

- [dpa_web_user](tables/dpa_web_user.md) - 6 columns
- [dpa_web_role](tables/dpa_web_role.md) - 4 columns
```

### NDJSON (streaming format)

```json
{"_type":"metadata","name":"goanydb","databaseType":"mysql","tableCount":210}
{"_type":"table","id":"1","name":"dpa_web_user","schema":"goanydb","fields":[{"id":"1","name":"user_id","type":{"id":"bigint","name":"bigint"},"primaryKey":true}]}
{"_type":"relationship","id":"1","name":"fk_project_user","sourceTableId":"5","targetTableId":"1","sourceFieldId":"12","targetFieldId":"1"}
```

### TSV (columns.tsv excerpt)

```
id	table_id	table_name	name	type	is_pk	is_unique	is_nullable	default
1	1	dpa_web_user	user_id	bigint	Y	Y	N	
2	1	dpa_web_user	username	varchar	N	Y	N	
3	1	dpa_web_user	email	varchar	N	N	Y	
```

---

## Deployment

### Build from Source

```bash
cd dbminer
go build -o dbminer .
```

### Cross-Platform Build

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o dbminer.exe .

# Linux
GOOS=linux GOARCH=amd64 go build -o dbminer .

# macOS
GOOS=darwin GOARCH=amd64 go build -o dbminer .
```

### Verify Installation

```bash
./dbminer -h
```

---

## Testing

See [TESTING.md](TESTING.md) for comprehensive test documentation.

### Quick Test

```bash
go test ./... -v
```

### Coverage Report

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

**Current Coverage:** 85.8%
