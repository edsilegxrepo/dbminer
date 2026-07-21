# dbminer Test Suite Documentation

**Version:** 0.9.1  
**Last Updated:** 2026-07-20  
**Coverage:** 85.8%

---

## Architecture, Design and Principles

### Design Philosophy

The test suite follows these core principles:

1. **Pure Logic First** - ~70% of dbminer code is pure transformation with no I/O. These functions are tested with fast, table-driven unit tests requiring no mocking.

2. **Realistic Data Over Synthetic** - Tests use real GoAnywhere MFT schema data (210 tables, 2000 columns, 154 FKs) rather than minimal synthetic fixtures.

3. **Portable by Default** - All tests run identically on Windows and Linux using Go's `t.TempDir()` for file operations.

4. **Structure Validation** - Export tests don't just verify files exist; they parse output and validate field-level structure.

5. **No Mocking Required** - The architecture separates I/O from logic, making most code directly testable without mocks.

### Package Responsibilities

```
dbminer/
├── main.go              # CLI orchestration (flag parsing, mode routing)
├── main_test.go         # Flag validation, generateSQL, exportSchema
├── testdata/            # Centralized test fixtures
│   └── goanywhere_sample.json
├── schema/
│   ├── raw.go           # JSON loading (LoadRaw)
│   ├── convert.go       # RawSchema → Schema transformation
│   ├── export.go        # JSON/NDJSON/TSV export
│   └── *_test.go        # Unit + realistic data tests
├── gensql/
│   ├── driver.go        # Driver registry, validation
│   ├── template.go      # Template execution
│   └── *_test.go        # All 9 drivers, validation tests
└── markdown/
    ├── generator.go     # Markdown documentation generation
    └── *_test.go        # Structure and content validation
```

---

## Logic Flow of Tests

### Main Categories

| Category | Purpose | Test Files |
|----------|---------|------------|
| **Unit Tests** | Pure function testing | `convert_test.go`, `driver_test.go`, `template_test.go` |
| **I/O Tests** | File read/write with temp dirs | `raw_test.go`, `export_test.go`, `generator_test.go` |
| **Realistic Data** | GoAnywhere schema validation | `goanywhere_test.go`, `goanywhere_export_test.go`, `markdown/goanywhere_test.go` |
| **Integration** | End-to-end CLI workflows | `main_test.go` |

### Positive Testing

- Valid schema conversion with all entity types (tables, views, FKs, indexes, triggers, procedures)
- All 9 database drivers generate valid output
- Export to all formats (JSON, NDJSON, TSV combined, TSV split)
- Markdown generation with relationships, indexes, ERD diagrams
- FK chain navigation (job_log → job → project → web_user)

### Negative Testing

- Invalid JSON input files
- Empty/missing files
- Unknown database drivers
- Invalid identifiers (SQL injection patterns, special characters)
- Invalid flag combinations
- Missing required flags

---

## Technical Requirements and Setup

### Dependencies

- **Go 1.21+** (uses generics, embed.FS)
- No external test dependencies (standard `testing` package only)

### Environment Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `GOANYWHERE_SCHEMA_PATH` | Path to full 210-table GoAnywhere schema for scale testing | Optional (tests skip if unset) |

### Constraints

- Tests use `t.TempDir()` for all file operations (auto-cleanup)
- No network calls in unit tests
- No database connections required for offline tests
- Tests must pass on both Windows and Linux

### Test Data Location

All test fixtures are centralized in `dbminer/testdata/`:

```
testdata/
└── goanywhere_sample.json   # 10 tables, 40 columns, 7 FKs, 20 indexes
```

Full schema file (external, referenced by env var):
```
arch-design/DB/goanywhere_schema_raw.json   # 210 tables, 1974 columns, 154 FKs
```

---

## List of Tests

### schema/convert_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| Basic Conversion | `TestConvertRawToSchema_MinimalTable` | Single table with PK | Schema has 1 table, 2 fields, PK detected |
| Multi-Entity | `TestConvertRawToSchema_MultipleTablesAndViews` | Tables + views | 3 entities, view marked IsView=true |
| Foreign Keys | `TestConvertRawToSchema_ForeignKeys` | FK relationship creation | 1 relationship with correct cardinality |
| Foreign Keys | `TestConvertRawToSchema_ForeignKeysWithSchema` | Cross-schema FKs | FK resolves with schema prefix |
| Foreign Keys | `TestConvertRawToSchema_ForeignKeysMissingTable` | Missing target table | FK skipped, no error |
| Indexes | `TestConvertRawToSchema_Indexes` | Index detection | 3 indexes, unique flag from index |
| Triggers | `TestConvertRawToSchema_Triggers` | Trigger parsing | Timing, event, statement captured |
| Stored Procs | `TestConvertRawToSchema_StoredProcedures` | SP/function parsing | Parameters, return type captured |
| Generated Cols | `TestConvertRawToSchema_GeneratedColumns` | Computed columns | Expression and type (STORED/VIRTUAL) |
| Ordering | `TestConvertRawToSchema_ColumnOrdering` | Ordinal position sort | Fields in correct order |
| Ordering | `TestConvertRawToSchema_IndexColumnOrdering` | Multi-column index order | Index columns in position order |
| Type Detection | `TestDetectDatabaseType` | 14 database type scenarios | Correct type for each version string |
| Type Coercion | `TestToBool` | 18 input type scenarios | Correct boolean for each input |
| Composite PK | `TestConvertRawToSchema_CompositePK` | Multi-column primary key | Both columns marked PK |
| Unique Index | `TestConvertRawToSchema_SingleColumnUniqueIndex` | Unique from index | Field marked Unique |
| Unique Index | `TestConvertRawToSchema_UniqueConstraintFromMultiColumnIndex` | Multi-col unique | Individual fields NOT marked unique |
| PK Unique | `TestConvertRawToSchema_PKImpliesUnique` | PK implies unique | PK field also marked Unique |
| Defaults | `TestConvertRawToSchema_FieldDefault` | Default values | CURRENT_TIMESTAMP and literals |
| Nullable | `TestConvertRawToSchema_FieldCollation` | Collation capture | Collation string preserved |
| Comments | `TestConvertRawToSchema_FieldComment` | Column comments | Comment string preserved |
| Views | `TestConvertRawToSchema_ViewColumns` | View field handling | No PK on view columns |
| Empty | `TestConvertRawToSchema_EmptyInput` | Empty schema | 0 tables, no error |
| Positions | `TestConvertRawToSchema_DefaultPositions` | X/Y defaults | DefaultX, DefaultY values |

### schema/raw_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| Load Valid | `TestLoadRaw_ValidJSON` | Basic JSON parsing | All fields populated |
| Load Valid | `TestLoadRaw_AllFields` | Full schema with all entities | 10+ entity types parsed |
| Error Handling | `TestLoadRaw_FileNotFound` | Missing file | Error returned |
| Error Handling | `TestLoadRaw_InvalidJSON` | Malformed JSON | Error returned |
| Error Handling | `TestLoadRaw_EmptyFile` | Zero-byte file | Error returned |
| Edge Cases | `TestLoadRaw_EmptyObject` | `{}` input | Empty schema, no error |
| Type Variants | `TestLoadRaw_NullableFieldTypes` | bool/string/int nullable | All parsed correctly |
| Type Variants | `TestLoadRaw_IndexSizeVariants` | int/string/null size | All parsed correctly |
| Type Variants | `TestLoadRaw_PrecisionVariants` | int/string/null precision | All parsed correctly |
| Unicode | `TestLoadRaw_UnicodeContent` | CJK characters, emoji | Content preserved |

### schema/export_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| JSON Export | `TestExportJSON` | Valid JSON output | Unmarshals to Schema |
| JSON Export | `TestExportJSON_Indentation` | Pretty print | 2-space indentation |
| NDJSON Export | `TestExportNDJSON` | Line-per-record | Correct line count |
| NDJSON Export | `TestExportNDJSON_RecordTypes` | _type field | All types present |
| NDJSON Export | `TestExportNDJSON_MetadataContent` | Metadata record | Counts match schema |
| NDJSON Export | `TestExportNDJSON_NoHTMLEscape` | No HTML escaping | < and & not escaped |
| TSV Export | `TestExportTSV_Combined` | Combined format | Header + all entities |
| TSV Export | `TestExportTSV_Split` | Separate files | 4 files created |
| TSV Export | `TestExportTSV_SplitWithTriggers` | triggers.tsv | File created when triggers exist |
| TSV Export | `TestExportTSV_SplitWithProcedures` | procedures.tsv | File created when SPs exist |
| TSV Content | `TestExportTSV_SpecialCharacters` | Tab/newline escape | No raw tabs in output |
| TSV Content | `TestExportTSV_ForeignKeyInfo` | FK reference columns | Target table in output |
| TSV Content | `TestExportTSV_PKIndicator` | PK column flag | Y in is_pk column |
| TSV Content | `TestExportTSV_IndexInfo` | INDEX rows | Index names present |
| TSV Content | `TestExportTSV_TriggerInfo` | TRIGGER rows | Trigger names present |
| TSV Content | `TestExportTSV_StoredProcInfo` | FUNCTION rows | SP names present |
| Error Handling | `TestExport_InvalidFormat` | Unknown format | Error returned |
| Error Handling | `TestExport_WriteError` | Invalid path | Error returned |
| Edge Cases | `TestExport_EmptySchema` | No tables | Files created, no error |
| Scale | `TestExport_LargeSchema` | 100 tables x 20 cols | All formats succeed |
| Utility | `TestEscapeTSV` | Escape function | Tab/newline replaced |

### schema/goanywhere_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| Load | `TestGoAnywhereSchema_Load` | Realistic JSON load | 10 tables, 7 FKs parsed |
| Convert | `TestGoAnywhereSchema_Convert` | Full conversion | mysql type, 7 relationships |
| PK Detection | `TestGoAnywhereSchema_PKDetection` | user_id is PK | PrimaryKey=true, Unique=true |
| Composite PK | `TestGoAnywhereSchema_CompositePK` | contact_id + group_id | 2 PK columns |
| FK Chain | `TestGoAnywhereSchema_FKChain` | 4-level chain | All 3 FKs found |
| Indexes | `TestGoAnywhereSchema_IndexDetection` | idx_username | Unique index found |
| Unique | `TestGoAnywhereSchema_UniqueFromIndex` | username unique | Field marked unique |
| Nullable | `TestGoAnywhereSchema_NullableHandling` | YES/NO parsing | Correct nullable flags |
| Defaults | `TestGoAnywhereSchema_DefaultValues` | is_active, created_date | "1" and "CURRENT_TIMESTAMP" |
| Export JSON | `TestGoAnywhereSchema_ExportJSON` | Roundtrip | 10 tables, 7 rels in output |
| Export NDJSON | `TestGoAnywhereSchema_ExportNDJSON` | Line count | 18 lines (1+10+7) |
| Export TSV | `TestGoAnywhereSchema_ExportTSV` | Row counts | 10 TABLE, 30+ COLUMN |
| Export Split | `TestGoAnywhereSchema_ExportTSVSplit` | All files | 4 files with correct counts |
| Full Schema | `TestGoAnywhereSchema_FullFile` | 210 tables | All formats succeed |

### schema/goanywhere_export_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| NDJSON Structure | `TestGoAnywhereExport_NDJSON_Structure` | Field-level validation | metadata, tables, relationships correct |
| TSV Structure | `TestGoAnywhereExport_TSV_Structure` | Column-level validation | Header, types, FK refs correct |
| TSV Split | `TestGoAnywhereExport_TSVSplit_Structure` | Per-file validation | All 4 files correct format |
| Roundtrip | `TestGoAnywhereExport_NDJSON_Roundtrip` | Data preservation | Counts match original |
| Full NDJSON | `TestGoAnywhereExport_FullSchema_NDJSON` | 210 tables NDJSON | 200+ table records |
| Full TSV | `TestGoAnywhereExport_FullSchema_TSVSplit` | 210 tables split | 200+ rows per file |

### gensql/driver_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| Validation | `TestValidateIdentifier` | 13 identifier scenarios | Valid/invalid correctly |
| Validation | `TestValidateMongoDBName` | 11 MongoDB name scenarios | Hyphen allowed, invalid rejected |
| Validation | `TestValidateUser` | 12 user name scenarios | @.% allowed, ;\ rejected |
| Registry | `TestGetDriver` | All 9 drivers | Driver returned, correct name |
| Registry | `TestGetDriver_Unknown` | Invalid driver names | Error returned |
| Registry | `TestListDrivers` | Driver enumeration | 9 drivers, sorted |
| Generation | `TestAllDriversGenerateSQL_Basic` | All drivers basic | Non-empty output >100 chars |
| Generation | `TestAllDriversGenerateSQL_DirectMode` | Direct query mode | No procedure wrapper |
| Generation | `TestAllDriversGenerateSQL_NoAdminMode` | NoAdmin flag | Output generated |
| Generation | `TestAllDriversGenerateSQL_WithExecUser` | ExecUser parameter | Output generated |
| MySQL | `TestMySQLDriver_StoredProc` | Procedure mode | CREATE PROCEDURE in output |
| MySQL | `TestMySQLDriver_DirectQuery` | Direct mode | No CREATE PROCEDURE |
| Postgres | `TestPostgresDriver_StoredProc` | Function mode | Output generated |
| MSSQL | `TestMSSQLDriver_StoredProc` | Procedure mode | CREATE PROCEDURE in output |
| Oracle | `TestOracleDriver_StoredProc` | Procedure mode | Output generated |
| SQLite | `TestSQLiteDriver_AlwaysDirect` | No procedures | SELECT without procedure |
| MongoDB | `TestMongoDBDriver_JavaScriptOutput` | JS not SQL | db.getCollectionNames() |
| MongoDB | `TestMongoDBDriver_SampleSizeDefaults` | 4 size scenarios | Defaults and caps applied |
| MongoDB | `TestMongoDBDriver_MaxDepthDefaults` | 4 depth scenarios | Defaults and caps applied |
| JSON Output | `TestDriversContainJSONOutput` | JSON generation | JSON keyword in most drivers |

### gensql/template_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| All Drivers | `TestExecuteTemplate_AllDrivers` | Template execution | Non-empty for all 9 |
| Error | `TestExecuteTemplate_UnknownTemplate` | Invalid name | Error returned |
| Substitution | `TestExecuteTemplate_DBNameSubstitution` | DBName variable | No error, non-empty |
| Substitution | `TestExecuteTemplate_SPNameSubstitution` | SPName variable | No error, non-empty |
| Modes | `TestExecuteTemplate_StoredProcMode` | StoredProc=true | CREATE in output |
| Modes | `TestExecuteTemplate_DirectMode` | StoredProc=false | Output generated |
| Modes | `TestExecuteTemplate_IncludeAdmin` | Admin queries | Output varies |
| Modes | `TestExecuteTemplate_StoredProcWithAllVars` | All vars together | No error |
| MongoDB | `TestExecuteTemplate_MongoDBVars` | SampleSize, MaxDepth | Values in output |
| Defaults | `TestTemplateVars_Defaults` | Zero values | No error |
| Syntax | `TestExecuteTemplate_ValidSyntax` | No template artifacts | No <no value> |
| Syntax | `TestExecuteTemplate_SQLInjectionPrevention` | Special chars | Quoted/escaped |
| Variants | `TestExecuteTemplate_MySQLvsMySQL57` | Different syntax | Outputs differ |
| Variants | `TestExecuteTemplate_MSSQLvsMSSQL2016` | Different syntax | Outputs differ |
| Scale | `TestExecuteTemplate_OutputNotTruncated` | Minimum size | >100 bytes each |
| Extensions | `TestTemplateExtensions` | .sql vs .js | Correct syntax type |

### markdown/generator_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| Structure | `TestGenerate_BasicStructure` | Dirs and files | README + tables/ created |
| README | `TestGenerate_README_Header` | Header content | Name, type, counts |
| README | `TestGenerate_README_TableOfContents` | TOC links | Links to all tables |
| README | `TestGenerate_README_MermaidERD` | Domain ERD | mermaid block present |
| README | `TestGenerate_README_RelationshipsTable` | FK table | Relationship details |
| README | `TestGenerate_README_TriggersSection` | Triggers table | Trigger names |
| README | `TestGenerate_README_StoredProceduresSection` | SP table | SP names |
| README | `TestGenerate_README_ViewIcon` | View indicator | (eye icon) for views |
| Table Doc | `TestGenerate_TableDoc_Header` | Doc header | Type, schema, columns |
| Table Doc | `TestGenerate_TableDoc_ColumnsTable` | Column table | All fields, PK/UQ indicators |
| Table Doc | `TestGenerate_TableDoc_Indexes` | Index section | Index names |
| Table Doc | `TestGenerate_TableDoc_Triggers` | Triggers section | Timing, event |
| Table Doc | `TestGenerate_TableDoc_OutboundRelationships` | FK references | Target table links |
| Table Doc | `TestGenerate_TableDoc_InboundRelationships` | Referenced by | Source table links |
| Table Doc | `TestGenerate_TableDoc_MermaidERD` | Table ERD | mermaid block |
| Table Doc | `TestGenerate_TableDoc_BackLink` | Navigation | Link to README |
| Views | `TestGenerate_ViewDoc` | View type | Type: View |
| Generated | `TestGenerate_GeneratedColumns` | Computed cols | Expression shown |
| Defaults | `TestGenerate_DefaultValues` | Default values | CURRENT_TIMESTAMP |
| Nullable | `TestGenerate_NullableIndicator` | YES/NO | Both indicators |
| Indexes | `TestGenerate_MultipleIndexColumns` | Composite index | Column list |
| SPs | `TestGenerate_StoredProcParameters` | Param details | IN/OUT, types |
| Sanitize | `TestSanitizeFilename` | 10 scenarios | Safe filenames |
| Sanitize | `TestSanitizeMermaidID` | 6 scenarios | Valid IDs |
| Escape | `TestEscapeMarkdownCell` | Pipe, newline | Escaped output |
| Escape | `TestEscapeMermaidLabel` | Pipe, bracket | Replaced output |
| Edge Cases | `TestGenerate_EmptySchema` | No tables | README created |
| Edge Cases | `TestGenerate_InvalidOutputDir` | Bad path | Error returned |
| Unicode | `TestGenerate_UnicodeContent` | CJK, emoji | Content preserved |
| Scale | `TestGenerate_LargeSchema` | 50 tables | All files created |
| Delimiter | `TestDetectDelimiter` | 6 delimiter scenarios | Correct delimiter or empty |
| Grouping | `TestGetTableGroup` | 5 groupBy scenarios | Correct group key |
| GroupBy | `TestGenerate_GroupByNone` | none option | Single "Tables" group |
| GroupBy | `TestGenerate_GroupBySchema` | schema option | Schema-based groups |

### markdown/goanywhere_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| README | `TestGoAnywhereMarkdown_Generate` | Realistic README | Header, counts, ERD, TOC |
| Table Docs | `TestGoAnywhereMarkdown_TableDocs` | All tables | 10 files, correct content |
| Table Docs | `TestGoAnywhereMarkdown_TableDocs/dpa_web_user` | User table | PK, indexes, inbound FK |
| Table Docs | `TestGoAnywhereMarkdown_TableDocs/dpa_job_log` | Log table | Outbound FK, ERD |
| Table Docs | `TestGoAnywhereMarkdown_TableDocs/dpa_trading_partner` | Isolated table | No ERD section |
| FK Chain | `TestGoAnywhereMarkdown_FKChainVisualization` | Navigation | All links work |
| Columns | `TestGoAnywhereMarkdown_ColumnDetails` | Types, nullable | varchar(100), YES/NO |
| Indexes | `TestGoAnywhereMarkdown_IndexDetails` | Composite index | Multiple columns |
| Grouping | `TestGoAnywhereMarkdown_DomainGrouping` | TOC groups | dpa_ prefix groups |
| Full Schema | `TestGoAnywhereMarkdown_FullSchema` | 210 tables | 200+ files, README >10KB |

### main_test.go

| Logical Group | Test Name | Purpose | Success Criteria |
|---------------|-----------|---------|------------------|
| Flags | `TestValidateFlags_GenSQLMode` | 5 gensql scenarios | Mutual exclusion enforced |
| Flags | `TestValidateFlags_RawMode` | 8 raw scenarios | Required flags enforced |
| Flags | `TestValidateFlags_NoModeOrphanFlags` | 4 orphan scenarios | Orphan flags rejected |
| Generate | `TestGenerateSQL_AllDrivers` | All 9 drivers | Output files created |
| Generate | `TestGenerateSQL_DirectMode` | Direct flag | Output created |
| Generate | `TestGenerateSQL_NoAdminMode` | NoAdmin flag | Output created |
| Generate | `TestGenerateSQL_Stdout` | No -o flag | Writes to stdout |
| Generate | `TestGenerateSQL_MongoDB` | JS output | .js file created |
| Validation | `TestGenerateSQL_InvalidDriver` | Bad driver | Error returned |
| Validation | `TestGenerateSQL_InvalidDBName` | Hyphen in SQL name | Error returned |
| Validation | `TestGenerateSQL_MongoDBValidation` | MongoDB names | Hyphen allowed, leading hyphen rejected |
| Validation | `TestGenerateSQL_InvalidSPName` | Bad SP name | Error returned |
| Validation | `TestGenerateSQL_InvalidUser` | Bad user name | Error returned |
| Export | `TestExportSchema_AllFormats` | JSON, NDJSON, TSV | All files created |
| Export | `TestExportSchema_TSVSplit` | Split TSV | Multiple files created |
| Export | `TestExportSchema_InvalidFormat` | Bad format | Error returned |

---

## Code Coverage Report

### Current Statistics (2026-07-20)

| Package | Coverage | Notes |
|---------|----------|-------|
| `criticalsys.net/dbminer` (main) | 46.5% | CLI orchestration, `main()` untested |
| `criticalsys.net/dbminer/gensql` | 90.4% | All drivers, validation |
| `criticalsys.net/dbminer/markdown` | 95.7% | Full generation coverage |
| `criticalsys.net/dbminer/schema` | 92.1% | Convert, export, load |
| **Total** | **85.8%** | Exceeds 80% requirement |

### Key Function Coverage

| Function | Coverage |
|----------|----------|
| `ConvertRawToSchema` | 99.1% |
| `detectDatabaseType` | 100.0% |
| `toBool` | 100.0% |
| `LoadRaw` | 100.0% |
| `Export` | 100.0% |
| `generateReadme` | 97.4% |
| `generateTableDoc` | 96.1% |
| `generateMermaidERD` | 97.2% |
| `validateFlags` | 100.0% |
| `generateSQL` | 92.3% |
| `exportSchema` | 92.3% |

### How to Refresh Coverage Statistics

**PowerShell:**
```powershell
cd x:\projects\filetransfer\goanywhere\arch-design\DB\dbminer
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | Select-String "total"
```

**Bash:**
```bash
cd "x:/projects/filetransfer/goanywhere/arch-design/DB/dbminer"
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

**Generate HTML Report:**
```bash
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Requirements

- **Minimum Total Coverage:** 80%
- **Package Minimum:** 85% for core packages (schema, gensql, markdown)
- **Critical Functions:** 95% for conversion and export logic

---

## Realistic Data Simulation

### Test Fixtures

| Fixture | Location | Description |
|---------|----------|-------------|
| `goanywhere_sample.json` | `testdata/` | 10 tables, 40 columns, 7 FKs, 20 indexes |
| `goanywhere_schema_raw.json` | `arch-design/DB/` | Full 210 tables, 1974 columns, 154 FKs |

### Integration Tests with Live Data

Full schema tests require the `GOANYWHERE_SCHEMA_PATH` environment variable:

**PowerShell:**
```powershell
$env:GOANYWHERE_SCHEMA_PATH = "x:\projects\filetransfer\goanywhere\arch-design\DB\goanywhere_schema_raw.json"
go test ./... -v -run "FullSchema"
```

**Bash:**
```bash
GOANYWHERE_SCHEMA_PATH="x:/projects/filetransfer/goanywhere/arch-design/DB/goanywhere_schema_raw.json" \
  go test ./... -v -run "FullSchema"
```

### What Full Schema Tests Validate

| Test | Validates |
|------|-----------|
| `TestGoAnywhereSchema_FullFile` | Load 210 tables, convert, export all formats |
| `TestGoAnywhereExport_FullSchema_NDJSON` | 200+ table records, 100+ relationships |
| `TestGoAnywhereExport_FullSchema_TSVSplit` | 1500+ column rows, correct file structure |
| `TestGoAnywhereMarkdown_FullSchema` | 200+ table docs, README >10KB |

### Live Database Integration (Future)

For live database testing, the workflow is:

1. Generate SQL script: `dbminer -gensql -driver mysql -db goanydb -o export.sql`
2. Execute on database: `mysql -u user -p goanydb < export.sql > schema.json`
3. Process output: `dbminer -raw schema.json -output ./docs`
4. Validate output structure and content

---

## How to Run the Tests

### Run All Tests

**PowerShell:**
```powershell
cd x:\projects\filetransfer\goanywhere\arch-design\DB\dbminer
go test ./... -v
```

**Bash:**
```bash
cd "x:/projects/filetransfer/goanywhere/arch-design/DB/dbminer"
go test ./... -v
```

### Run with Coverage

**PowerShell:**
```powershell
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

**Bash:**
```bash
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

### Run Specific Test Categories

**Unit tests only (fast):**
```bash
go test ./... -short
```

**GoAnywhere realistic data tests:**
```bash
go test ./... -v -run "GoAnywhere"
```

**Full schema tests (requires env var):**
```powershell
$env:GOANYWHERE_SCHEMA_PATH = "x:\projects\filetransfer\goanywhere\arch-design\DB\goanywhere_schema_raw.json"
go test ./... -v -run "FullSchema"
```

**Single package:**
```bash
go test ./schema -v
go test ./gensql -v
go test ./markdown -v
```

**Single test:**
```bash
go test ./schema -v -run "TestConvertRawToSchema_ForeignKeys"
```

### Run with Race Detection

```bash
go test ./... -race
```

---

## Maintenance and Troubleshooting

### When to Update Tests

1. **New feature added** - Add tests for new functionality
2. **Bug fixed** - Add regression test for the bug
3. **Schema format changed** - Update fixtures and validation tests
4. **Driver added** - Add to driver test loops

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `testdata/goanywhere_sample.json: no such file` | Wrong working directory | Run from `dbminer/` directory |
| `GOANYWHERE_SCHEMA_PATH` tests skipped | Env var not set | Set env var or skip full schema tests |
| Coverage below 80% | New code without tests | Add tests for new functions |
| Test fails on Windows but passes on Linux | Path separators | Use `filepath.Join()` not string concat |
| `toBool` returns wrong value | Unexpected nullable format | Add case to `toBool()` switch |

### Adding New Test Fixtures

1. Place JSON in `dbminer/testdata/`
2. Reference with relative path `../testdata/filename.json` from package tests
3. Document in this file under "Test Fixtures"

### Updating Coverage After Code Changes

```bash
# 1. Run tests and generate coverage
go test ./... -coverprofile=coverage.out

# 2. Get summary
go tool cover -func=coverage.out | grep total

# 3. Update TESTING.md with new percentage

# 4. If coverage dropped, identify uncovered lines:
go tool cover -html=coverage.out
```

### Test File Naming Convention

| Pattern | Purpose |
|---------|---------|
| `*_test.go` | Standard Go test file |
| `goanywhere_test.go` | Realistic data tests using GoAnywhere fixtures |
| `goanywhere_export_test.go` | Export-specific realistic data tests |

### Pre-Commit Checklist

- [ ] `go test ./...` passes
- [ ] `go test ./... -race` passes
- [ ] Coverage >= 80%
- [ ] New code has corresponding tests
- [ ] TESTING.md updated if tests added
