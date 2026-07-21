# dbminer Architecture

**Version:** 0.9.0  
**Last Updated:** 2026-07-20

---

## Architecture and Design Choices

### Overview

dbminer is a multi-database schema documentation generator designed around a three-phase workflow that separates concerns between DBA access and developer tooling:

```mermaid
flowchart LR
    subgraph Phase1["Phase 1: DBA Domain"]
        A[dbminer] -->|generates| B[SQL Script]
        B -->|executed by DBA| C[(Database)]
        C -->|produces| D[schema.json]
    end
    
    subgraph Phase2["Phase 2: Developer Domain"]
        D -->|processed by| E[dbminer]
        E -->|outputs| F[Markdown Docs]
        E -->|outputs| G[JSON/NDJSON]
        E -->|outputs| H[TSV Files]
    end
    
    style Phase1 fill:#e1f5fe
    style Phase2 fill:#e8f5e9
```

### Core Design Principles

1. **Separation of Privileges**: The application never requires direct database access. SQL scripts run in DBA context; processing runs in developer context.

2. **Pure Transformation Core**: ~70% of code is pure functions with no I/O, enabling fast unit tests without mocking.

3. **Driver Abstraction**: Database-specific SQL generation is encapsulated in driver implementations using Go's `text/template`.

4. **Format Agnostic Output**: The internal `Schema` type is the canonical representation; exporters convert to target formats.

### Package Architecture

```mermaid
flowchart TB
    subgraph CLI["main.go (CLI Orchestration)"]
        flags[Flag Parsing]
        validate[Validation]
        route[Mode Routing]
    end
    
    subgraph GenSQL["gensql/ (SQL Generation)"]
        driver[Driver Interface]
        templates[Embedded Templates]
        mysql[MySQL Driver]
        postgres[Postgres Driver]
        mongodb[MongoDB Driver]
        others[Other Drivers]
    end
    
    subgraph Schema["schema/ (Data Processing)"]
        raw[LoadRaw]
        convert[ConvertRawToSchema]
        types[Schema Types]
        export[Export Functions]
    end
    
    subgraph Markdown["markdown/ (Documentation)"]
        generator[Generate]
        readme[generateReadme]
        tables[generateTableDoc]
        mermaid[Mermaid ERD]
    end
    
    flags --> validate --> route
    route -->|gensql mode| driver
    route -->|raw mode| raw
    
    driver --> templates
    templates --> mysql & postgres & mongodb & others
    
    raw --> convert --> types
    types --> export
    types --> generator
    
    generator --> readme & tables
    tables --> mermaid
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| No database drivers | Eliminates credential management, connection complexity, and privilege escalation risks |
| Embedded SQL templates | Single binary distribution; templates versioned with code |
| NDJSON for LLM | Streaming-friendly; fits within context windows better than nested JSON |
| TSV split files | Enables selective import into spreadsheets without memory issues |
| Mermaid for ERDs | Renders in GitHub, VS Code, many documentation systems natively |

---

## Assumptions

### Input Data

1. **SQL Collector Output**: Raw JSON follows the structure produced by the embedded SQL scripts
2. **MySQL Nullable Format**: Columns may use "YES"/"NO" strings instead of booleans
3. **Version Strings**: Database type detected from version string patterns (e.g., "MySQL 8.0.32", "PostgreSQL 15.2")
4. **UTF-8 Encoding**: All input files are UTF-8 encoded

### Database Capabilities

1. **Information Schema**: Target databases have INFORMATION_SCHEMA or equivalent metadata views
2. **JSON Output**: SQL Server, MySQL, PostgreSQL can output JSON natively
3. **MongoDB**: Version 4.4+ for `$sample` aggregation and schema inference

### Runtime Environment

1. **File System**: Write access to output directory
2. **No Network**: Application operates entirely offline after script generation
3. **Go 1.21+**: Uses embed.FS for templates

---

## Edge Cases

### Schema Processing

| Edge Case | Handling |
|-----------|----------|
| Missing FK target table | Relationship skipped silently (logged in debug) |
| Composite primary keys | All columns marked as PK |
| Multi-column unique indexes | Index marked unique; individual columns NOT marked unique |
| Views with no columns | Empty Fields array; still generates documentation |
| Empty schema (no tables) | Warning printed; empty output generated |
| Circular FK references | Processed without issue; Mermaid handles cycles |

### Name Handling

| Edge Case | Handling |
|-----------|----------|
| Special characters in names | Sanitized for filenames and Mermaid IDs |
| Unicode table/column names | Preserved in output; sanitized only for file paths |
| Reserved SQL keywords | Quoted in templates where needed |
| Empty names | Replaced with "_invalid_" placeholder |

### Data Types

| Edge Case | Handling |
|-----------|----------|
| Nullable as "YES"/"NO" string | `toBool()` recognizes both formats |
| Nullable as 0/1 integer | `toBool()` handles numeric types |
| CharMaxLen as string or int | Treated as string; JSON handles both |
| NULL default values | Preserved as empty string |

---

## Performance and Efficiency

### Memory Efficiency

```mermaid
flowchart LR
    subgraph Streaming["Streaming Processing"]
        A[LoadRaw] -->|Full load| B[RawSchema]
        B -->|Single pass| C[Schema]
        C -->|Streaming write| D[Output Files]
    end
    
    subgraph Memory["Memory Profile"]
        M1["Peak: 2x schema size"]
        M2["210 tables: ~10MB peak"]
    end
```

- **Single-Pass Conversion**: RawSchema to Schema in one traversal
- **Streaming NDJSON**: One record written at a time, not buffered
- **No Index Rebuilding**: Uses maps for O(1) lookups during conversion

### Time Complexity

| Operation | Complexity | Notes |
|-----------|------------|-------|
| LoadRaw | O(n) | Linear in file size |
| ConvertRawToSchema | O(t + c + i + r) | Tables + columns + indexes + relationships |
| Export JSON | O(n) | Linear in schema size |
| Export NDJSON | O(n) | Streaming, constant memory |
| Export TSV | O(n) | Streaming, constant memory |
| Generate Markdown | O(t * r) | Tables * relationships for ERD |

### Benchmarks (GoAnywhere 210 tables)

| Operation | Time | Memory |
|-----------|------|--------|
| Load + Convert | ~50ms | ~8MB |
| Export JSON | ~20ms | ~4MB |
| Export NDJSON | ~25ms | ~1MB |
| Generate Markdown | ~150ms | ~12MB |

---

## Data Flow and Control Logic

### Operational Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI as dbminer CLI
    participant Driver as SQL Driver
    participant Raw as schema/raw
    participant Conv as schema/convert
    participant Exp as schema/export
    participant MD as markdown/generator
    participant FS as File System

    Note over User,FS: Phase 1: SQL Script Generation
    User->>CLI: dbminer -gensql -driver mysql -o export.sql
    CLI->>CLI: validateFlags()
    CLI->>Driver: GetDriver("mysql")
    Driver->>Driver: GenerateSQL(opts)
    Driver->>FS: Write export.sql
    
    Note over User,FS: [DBA executes script on database]
    
    Note over User,FS: Phase 2: Schema Processing
    User->>CLI: dbminer -raw schema.json -output ./docs
    CLI->>CLI: validateFlags()
    CLI->>Raw: LoadRaw("schema.json")
    Raw->>FS: Read file
    Raw-->>CLI: *RawSchema
    CLI->>Conv: ConvertRawToSchema(raw)
    Conv->>Conv: Build lookup maps
    Conv->>Conv: Process tables, views, FKs
    Conv-->>CLI: *Schema
    CLI->>MD: Generate(schema, "./docs")
    MD->>MD: generateReadme()
    MD->>MD: generateTableDoc() [per table]
    MD->>FS: Write README.md, tables/*.md
```

### Code Relations

```mermaid
flowchart TB
    subgraph main["main.go"]
        main_fn[main]
        validateFlags
        generateSQL
        exportSchema
        printUsage
    end
    
    subgraph gensql["gensql/"]
        GetDriver
        ValidateIdentifier
        ValidateMongoDBName
        ValidateUser
        ListDrivers
        Driver_iface[Driver interface]
        GenerateSQL_method[GenerateSQL method]
    end
    
    subgraph schema_pkg["schema/"]
        LoadRaw
        ConvertRawToSchema
        Export
        detectDatabaseType
        toBool
    end
    
    subgraph markdown_pkg["markdown/"]
        Generate
        generateReadme
        generateTableDoc
        generateMermaidERD
        sanitizeFilename
    end
    
    main_fn --> validateFlags
    main_fn --> generateSQL
    main_fn --> exportSchema
    main_fn --> LoadRaw
    main_fn --> ConvertRawToSchema
    main_fn --> Generate
    
    generateSQL --> GetDriver
    generateSQL --> ValidateIdentifier
    generateSQL --> ValidateMongoDBName
    generateSQL --> ValidateUser
    GetDriver --> Driver_iface
    Driver_iface --> GenerateSQL_method
    
    exportSchema --> Export
    
    ConvertRawToSchema --> detectDatabaseType
    ConvertRawToSchema --> toBool
    
    Generate --> generateReadme
    Generate --> generateTableDoc
    generateReadme --> generateMermaidERD
    generateTableDoc --> sanitizeFilename
```

### Data Transformation Sequence

```mermaid
sequenceDiagram
    participant JSON as schema.json
    participant Raw as RawSchema
    participant Schema as Schema
    participant Out as Output

    JSON->>Raw: json.Unmarshal
    Note right of Raw: RawTable[], RawColumn[]<br/>RawFKInfo[], RawIndex[]
    
    Raw->>Schema: ConvertRawToSchema()
    Note right of Schema: Build PK lookup map
    Note right of Schema: Group columns by table
    Note right of Schema: Sort by ordinal position
    Note right of Schema: Build table ID map
    
    Schema->>Schema: Process Tables
    Note right of Schema: Create Table with Fields<br/>Detect PK from pkColumns map<br/>Detect Unique from indexes
    
    Schema->>Schema: Process Views
    Note right of Schema: Same as tables<br/>Set IsView = true
    
    Schema->>Schema: Process Triggers
    Note right of Schema: Copy timing, event, statement
    
    Schema->>Schema: Process Stored Procedures
    Note right of Schema: Copy parameters, return type
    
    Schema->>Schema: Process Relationships
    Note right of Schema: Resolve table/field IDs<br/>Skip if target missing
    
    Schema->>Out: Export or Generate
    Note right of Out: JSON: MarshalIndent<br/>NDJSON: Streaming encode<br/>TSV: Tab-separated rows<br/>Markdown: Template rendering
```

---

## Dependencies

### Go Modules

```go
module criticalsys.net/dbminer

go 1.21

// No external dependencies - standard library only
```

### Standard Library Packages

| Package | Usage |
|---------|-------|
| `flag` | CLI argument parsing |
| `fmt` | Formatted I/O |
| `os` | File operations |
| `path/filepath` | Cross-platform paths |
| `encoding/json` | JSON marshal/unmarshal |
| `text/template` | SQL template execution |
| `embed` | Embedded SQL templates |
| `regexp` | Identifier validation |
| `sort` | Slice sorting |
| `strings` | String manipulation |
| `time` | Timestamps |
| `io` | Writer interface (errWriter) |

### Embedded Resources

| Resource | Purpose |
|----------|---------|
| `gensql/templates/*.sql` | SQL export scripts per driver |
| `gensql/templates/mongodb.js` | MongoDB schema inference script |

### Build Dependencies

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.21+ | Compilation |
| gosec | latest | Security linting (optional) |
| golangci-lint | latest | Code quality (optional) |

### Runtime Dependencies

**None** - dbminer is a statically-compiled single binary with no external dependencies.

### Test Dependencies

| Dependency | Purpose |
|------------|---------|
| `testing` | Standard Go test framework |
| `t.TempDir()` | Portable temporary directories |
| GoAnywhere sample data | Realistic test fixtures |

---

## Security Considerations

See [README.md#security-assessment](README.md#security-assessment) for the detailed security assessment.

### Summary

- No database credentials in application
- Input validation prevents SQL injection in generated scripts
- Output files use safe permissions (0644 files, 0755 directories)
- No network communication
- No external dependencies reduces supply chain risk
