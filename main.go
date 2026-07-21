// Package main provides the CLI entry point for dbminer, a multi-database schema
// documentation generator. It orchestrates three operational modes:
//   - gensql: Generate database-specific SQL/JS export scripts
//   - export: Convert raw schema JSON to JSON/NDJSON/TSV formats
//   - markdown: Generate human-readable documentation with Mermaid ERDs
package main

import (
	"flag"
	"fmt"
	"os"

	"criticalsys.net/dbminer/gensql"
	"criticalsys.net/dbminer/markdown"
	"criticalsys.net/dbminer/schema"
)

// version is set at build time via -ldflags "-X main.version=x.y.z"
var version = "dev"

func main() {
	// Input flags
	rawFile := flag.String("raw", "", "Path to raw JSON file from SQL collector")

	// Output flags
	outputDir := flag.String("output", "./docs", "Output directory for markdown files")
	groupBy := flag.String("group-by", "auto", "Table grouping in markdown: auto, prefix, schema, none")
	exportFormat := flag.String("export-format", "", "Export format: json, ndjson, tsv (outputs to -o file instead of markdown)")
	exportFile := flag.String("o", "", "Output file for -export-format (required with -export-format)")
	tsvSplit := flag.Bool("tsv-split", false, "Split TSV into separate files (tables.tsv, columns.tsv, etc)")

	// SQL generation flags
	genSQLFlag := flag.Bool("gensql", false, "Generate database-specific export SQL script")
	driver := flag.String("driver", "mysql", "Database driver: mysql, mysql57, mariadb, postgres, sqlite, mssql, mssql2016, oracle, mongodb")
	dbName := flag.String("db", "mydb", "Database name for generated SP")
	spName := flag.String("sp", "sp_exportschema", "Stored procedure name")
	execUser := flag.String("user", "", "User to impersonate when executing (optional)")
	directSQL := flag.Bool("direct", false, "Generate direct SQL query instead of stored procedure")
	noAdmin := flag.Bool("noadmin", false, "Skip queries requiring admin privileges (index sizes, row counts)")

	// MongoDB-specific flags
	sampleSize := flag.Int("sample", 100, "MongoDB: documents to sample per collection (default 100)")
	maxDepth := flag.Int("depth", 2, "MongoDB: nested object expansion depth (default 2)")

	// Info flags
	showVersion := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("dbminer %s\n", version)
		return
	}

	// Validate flag combinations
	if err := validateFlags(*genSQLFlag, *rawFile, *exportFormat, *exportFile, *tsvSplit, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Mode 1: Generate SQL script for DBA
	if *genSQLFlag {
		if err := generateSQL(*driver, *dbName, *spName, *execUser, *directSQL, *noAdmin, *sampleSize, *maxDepth, *exportFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Require input file for processing modes
	if *rawFile == "" {
		printUsage()
		os.Exit(1)
	}

	// Load raw schema from SQL collector output
	raw, err := schema.LoadRaw(*rawFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading raw schema: %v\n", err)
		os.Exit(1)
	}

	// Convert to unified schema format
	s := schema.ConvertRawToSchema(raw)

	if len(s.Tables) == 0 {
		fmt.Fprintf(os.Stderr, "Warning: Schema contains no tables\n")
	}

	// Mode 2: Export to JSON/NDJSON/TSV
	if *exportFormat != "" {
		if err := exportSchema(s, *exportFormat, *exportFile, *tsvSplit); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Mode 3: Generate markdown documentation (default)
	opts := markdown.Options{GroupBy: *groupBy}
	if err := markdown.Generate(s, *outputDir, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating documentation: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Generated documentation for %d tables in %s\n", len(s.Tables), *outputDir)
}

func printUsage() {
	fmt.Printf("dbminer v%s - Database Schema Documentation Generator\n", version)
	fmt.Println()
	fmt.Println("WORKFLOW:")
	fmt.Println("  Step 1: Generate SQL script and send to DBA")
	fmt.Println("          dbminer -gensql -driver mysql -o export.sql")
	fmt.Println()
	fmt.Println("  Step 2: DBA runs script on database, returns schema.json")
	fmt.Println()
	fmt.Println("  Step 3: Generate outputs from schema.json")
	fmt.Println("          dbminer -raw schema.json                           # Markdown docs")
	fmt.Println("          dbminer -raw schema.json -export-format ndjson -o out.ndjson  # LLM")
	fmt.Println("          dbminer -raw schema.json -export-format tsv -o out.tsv        # Excel")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -gensql              Generate SQL export script for DBA")
	fmt.Println("  -direct              Generate direct SQL query instead of stored procedure")
	fmt.Println("  -noadmin             Skip queries requiring admin privileges (index sizes, row counts)")
	fmt.Println("  -driver <name>       Database: mysql, mysql57, mariadb, postgres, sqlite,")
	fmt.Println("                       mssql, mssql2016, oracle, mongodb")
	fmt.Println("  -db <name>           Database name in generated script (default: mydb)")
	fmt.Println("  -sp <name>           Stored procedure name (default: sp_exportschema)")
	fmt.Println("  -user <name>         User to impersonate when executing (optional)")
	fmt.Println("  -raw <file>          Input JSON from SQL collector")
	fmt.Println("  -output <dir>        Markdown output directory (default: ./docs)")
	fmt.Println("  -export-format <fmt> Export format: json, ndjson, tsv")
	fmt.Println("  -o <file>            Output file for -gensql or -export-format")
	fmt.Println("  -tsv-split           Split TSV into tables.tsv, columns.tsv, etc.")
	fmt.Println()
	fmt.Println("MONGODB OPTIONS:")
	fmt.Println("  -sample <n>          Documents to sample per collection (default: 100)")
	fmt.Println("  -depth <n>           Nested object expansion depth (default: 2)")
}

// validateFlags enforces mutual exclusivity between operational modes and
// validates flag combinations. Returns an error for invalid flag combinations.
func validateFlags(genSQL bool, rawFile, exportFormat, exportFile string, tsvSplit bool, outputDir string) error {
	// -gensql mode: cannot combine with -raw or -export-format
	if genSQL {
		if rawFile != "" {
			return fmt.Errorf("-gensql and -raw are mutually exclusive")
		}
		if exportFormat != "" {
			return fmt.Errorf("-gensql and -export-format are mutually exclusive")
		}
		if tsvSplit {
			return fmt.Errorf("-tsv-split is only valid with -export-format tsv")
		}
		if outputDir != "./docs" {
			return fmt.Errorf("-output is only valid with -raw (for markdown generation)")
		}
		return nil
	}

	// -raw mode validations
	if rawFile != "" {
		if exportFormat != "" {
			// Export mode: require -o
			if exportFile == "" {
				return fmt.Errorf("-o <output file> is required with -export-format")
			}
			// -tsv-split only valid with tsv format
			if tsvSplit && exportFormat != "tsv" {
				return fmt.Errorf("-tsv-split is only valid with -export-format tsv")
			}
			// -output not valid with -export-format
			if outputDir != "./docs" {
				return fmt.Errorf("-output and -export-format are mutually exclusive (use -o for export)")
			}
		} else {
			// Markdown mode: -o not valid
			if exportFile != "" {
				return fmt.Errorf("-o is only valid with -gensql or -export-format")
			}
			if tsvSplit {
				return fmt.Errorf("-tsv-split is only valid with -export-format tsv")
			}
		}
		return nil
	}

	// No mode specified - will show usage
	// But check for orphan flags
	if exportFormat != "" {
		return fmt.Errorf("-export-format requires -raw <input file>")
	}
	if exportFile != "" {
		return fmt.Errorf("-o requires -gensql or -export-format")
	}
	if tsvSplit {
		return fmt.Errorf("-tsv-split requires -raw and -export-format tsv")
	}

	return nil
}

// generateSQL creates database-specific export scripts using the specified driver.
// Validates all identifiers before template execution to prevent SQL injection.
func generateSQL(driverName, dbName, spName, execUser string, direct, noAdmin bool, sampleSize, maxDepth int, outFile string) error {
	d, err := gensql.GetDriver(driverName)
	if err != nil {
		return err
	}

	// Use MongoDB-specific validation for database names
	if driverName == "mongodb" {
		if err := gensql.ValidateMongoDBName(dbName); err != nil {
			return err
		}
	} else {
		if err := gensql.ValidateIdentifier(dbName, "database name"); err != nil {
			return err
		}
		if err := gensql.ValidateIdentifier(spName, "procedure name"); err != nil {
			return err
		}
	}
	if err := gensql.ValidateUser(execUser); err != nil {
		return err
	}

	opts := gensql.GenerateOptions{
		DBName:     dbName,
		SPName:     spName,
		ExecUser:   execUser,
		Direct:     direct,
		NoAdmin:    noAdmin,
		SampleSize: sampleSize,
		MaxDepth:   maxDepth,
	}
	script, err := d.GenerateSQL(opts)
	if err != nil {
		return fmt.Errorf("generating script: %w", err)
	}

	scriptType := "SQL"
	if driverName == "mongodb" {
		scriptType = "JavaScript"
	}

	if outFile == "" {
		fmt.Print(script)
		fmt.Fprintf(os.Stderr, "%s script written to stdout\n", scriptType)
	} else {
		if err := os.WriteFile(outFile, []byte(script), 0o644); err != nil { // #nosec G306 -- 0644 is standard for user-created files
			return fmt.Errorf("writing script file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Generated %s script: %s\n", scriptType, outFile)
	}
	return nil
}

// exportSchema writes the schema to the specified file in the given format.
// Supports json, ndjson, and tsv (with optional split into separate files).
func exportSchema(s *schema.Schema, format, outFile string, tsvSplit bool) error {
	var f schema.ExportFormat
	switch format {
	case "json":
		f = schema.FormatJSON
	case "ndjson":
		f = schema.FormatNDJSON
	case "tsv":
		f = schema.FormatTSV
	default:
		return fmt.Errorf("unknown export format: %s (use: json, ndjson, tsv)", format)
	}

	opts := schema.ExportOptions{
		Format:   f,
		TSVSplit: tsvSplit,
	}

	if err := schema.Export(s, outFile, opts); err != nil {
		return err
	}

	if tsvSplit && f == schema.FormatTSV {
		fmt.Fprintf(os.Stderr, "Exported schema to %s/ (tables.tsv, columns.tsv, indexes.tsv, relationships.tsv)\n", outFile)
	} else {
		fmt.Fprintf(os.Stderr, "Exported schema to %s\n", outFile)
	}
	return nil
}
