// Test strategy: Integration tests for CLI orchestration layer.
// Tests validateFlags for mutual exclusivity, generateSQL for all drivers,
// and exportSchema for all formats. Uses t.TempDir() for file operations.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"criticalsys.net/dbminer/schema"
)

func TestValidateFlags_GenSQLMode(t *testing.T) {
	tests := []struct {
		name      string
		genSQL    bool
		rawFile   string
		exportFmt string
		exportOut string
		tsvSplit  bool
		outputDir string
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "gensql alone valid",
			genSQL:  true,
			wantErr: false,
		},
		{
			name:    "gensql with raw",
			genSQL:  true,
			rawFile: "input.json",
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		{
			name:      "gensql with export-format",
			genSQL:    true,
			exportFmt: "json",
			wantErr:   true,
			errMsg:    "mutually exclusive",
		},
		{
			name:     "gensql with tsv-split",
			genSQL:   true,
			tsvSplit: true,
			wantErr:  true,
			errMsg:   "only valid with",
		},
		{
			name:      "gensql with output dir changed",
			genSQL:    true,
			outputDir: "./custom",
			wantErr:   true,
			errMsg:    "only valid with -raw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := tt.outputDir
			if outputDir == "" {
				outputDir = "./docs"
			}
			err := validateFlags(tt.genSQL, tt.rawFile, tt.exportFmt, tt.exportOut, tt.tsvSplit, outputDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateFlags() error = %v, want containing %s", err, tt.errMsg)
			}
		})
	}
}

func TestValidateFlags_RawMode(t *testing.T) {
	tests := []struct {
		name      string
		rawFile   string
		exportFmt string
		exportOut string
		tsvSplit  bool
		outputDir string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "raw alone valid (markdown mode)",
			rawFile:   "input.json",
			outputDir: "./docs",
			wantErr:   false,
		},
		{
			name:      "raw with export-format without -o",
			rawFile:   "input.json",
			exportFmt: "json",
			outputDir: "./docs",
			wantErr:   true,
			errMsg:    "-o <output file> is required",
		},
		{
			name:      "raw with export-format and -o valid",
			rawFile:   "input.json",
			exportFmt: "json",
			exportOut: "out.json",
			outputDir: "./docs",
			wantErr:   false,
		},
		{
			name:      "raw with tsv-split but not tsv format",
			rawFile:   "input.json",
			exportFmt: "json",
			exportOut: "out.json",
			tsvSplit:  true,
			outputDir: "./docs",
			wantErr:   true,
			errMsg:    "only valid with -export-format tsv",
		},
		{
			name:      "raw with tsv-split and tsv format valid",
			rawFile:   "input.json",
			exportFmt: "tsv",
			exportOut: "out",
			tsvSplit:  true,
			outputDir: "./docs",
			wantErr:   false,
		},
		{
			name:      "raw with export-format and -output both set",
			rawFile:   "input.json",
			exportFmt: "json",
			exportOut: "out.json",
			outputDir: "./custom",
			wantErr:   true,
			errMsg:    "mutually exclusive",
		},
		{
			name:      "raw markdown mode with -o",
			rawFile:   "input.json",
			exportOut: "out.json",
			outputDir: "./docs",
			wantErr:   true,
			errMsg:    "-o is only valid",
		},
		{
			name:      "raw markdown mode with tsv-split",
			rawFile:   "input.json",
			tsvSplit:  true,
			outputDir: "./docs",
			wantErr:   true,
			errMsg:    "only valid with -export-format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlags(false, tt.rawFile, tt.exportFmt, tt.exportOut, tt.tsvSplit, tt.outputDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateFlags() error = %v, want containing %s", err, tt.errMsg)
			}
		})
	}
}

func TestValidateFlags_NoModeOrphanFlags(t *testing.T) {
	tests := []struct {
		name      string
		exportFmt string
		exportOut string
		tsvSplit  bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "no mode no flags valid",
			wantErr: false,
		},
		{
			name:      "export-format without raw",
			exportFmt: "json",
			wantErr:   true,
			errMsg:    "requires -raw",
		},
		{
			name:      "-o without mode",
			exportOut: "out.json",
			wantErr:   true,
			errMsg:    "requires -gensql or -export-format",
		},
		{
			name:     "tsv-split without raw",
			tsvSplit: true,
			wantErr:  true,
			errMsg:   "requires -raw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlags(false, "", tt.exportFmt, tt.exportOut, tt.tsvSplit, "./docs")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateFlags() error = %v, want containing %s", err, tt.errMsg)
			}
		})
	}
}

func TestGenerateSQL_AllDrivers(t *testing.T) {
	drivers := []string{"mysql", "mysql57", "mariadb", "postgres", "sqlite", "mssql", "mssql2016", "oracle", "mongodb"}

	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			dir := t.TempDir()
			outFile := filepath.Join(dir, "export.sql")

			err := generateSQL(driver, "testdb", "sp_export", "", false, false, 100, 2, outFile)
			if err != nil {
				t.Fatalf("generateSQL() error: %v", err)
			}

			data, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("ReadFile() error: %v", err)
			}
			if len(data) == 0 {
				t.Error("output file is empty")
			}
		})
	}
}

func TestGenerateSQL_DirectMode(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.sql")

	err := generateSQL("mysql", "testdb", "sp_export", "", true, false, 100, 2, outFile)
	if err != nil {
		t.Fatalf("generateSQL() error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	if len(data) == 0 {
		t.Error("output file is empty")
	}
}

func TestGenerateSQL_NoAdminMode(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.sql")

	err := generateSQL("mysql", "testdb", "sp_export", "", false, true, 100, 2, outFile)
	if err != nil {
		t.Fatalf("generateSQL() error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	if len(data) == 0 {
		t.Error("output file is empty")
	}
}

func TestGenerateSQL_Stdout(t *testing.T) {
	// When outFile is empty, should write to stdout (no error)
	err := generateSQL("mysql", "testdb", "sp_export", "", false, false, 100, 2, "")
	if err != nil {
		t.Fatalf("generateSQL() error: %v", err)
	}
}

func TestGenerateSQL_MongoDB(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.js")

	err := generateSQL("mongodb", "testdb", "", "", false, false, 500, 5, outFile)
	if err != nil {
		t.Fatalf("generateSQL() error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	if len(data) == 0 {
		t.Error("output file is empty")
	}
}

func TestGenerateSQL_InvalidDriver(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.sql")

	err := generateSQL("invalid_driver", "testdb", "sp_export", "", false, false, 100, 2, outFile)
	if err == nil {
		t.Error("expected error for invalid driver")
	}
}

func TestGenerateSQL_InvalidDBName(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.sql")

	// SQL drivers use ValidateIdentifier (no hyphens)
	err := generateSQL("mysql", "invalid-db-name", "sp_export", "", false, false, 100, 2, outFile)
	if err == nil {
		t.Error("expected error for invalid database name with hyphen")
	}
}

func TestGenerateSQL_MongoDBValidation(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.js")

	// MongoDB allows hyphens in db names
	err := generateSQL("mongodb", "my-database", "", "", false, false, 100, 2, outFile)
	if err != nil {
		t.Errorf("MongoDB should allow hyphens in db name: %v", err)
	}

	// But not starting with hyphen
	err = generateSQL("mongodb", "-invalid", "", "", false, false, 100, 2, outFile)
	if err == nil {
		t.Error("expected error for db name starting with hyphen")
	}
}

func TestGenerateSQL_InvalidSPName(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.sql")

	err := generateSQL("mysql", "testdb", "invalid-proc-name", "", false, false, 100, 2, outFile)
	if err == nil {
		t.Error("expected error for invalid procedure name")
	}
}

func TestGenerateSQL_InvalidUser(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.sql")

	err := generateSQL("mysql", "testdb", "sp_export", "invalid;user", false, false, 100, 2, outFile)
	if err == nil {
		t.Error("expected error for invalid user")
	}
}

func TestExportSchema_AllFormats(t *testing.T) {
	dir := t.TempDir()

	// Create a minimal test schema JSON
	inputFile := filepath.Join(dir, "input.json")
	content := `{
		"database_name": "testdb",
		"version": "MySQL 8.0",
		"tables": [{"table": "users", "type": "TABLE"}],
		"columns": [{"table": "users", "name": "id", "type": "int", "ordinal_position": 1}],
		"pk_info": [],
		"fk_info": [],
		"indexes": [],
		"views": []
	}`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Load and convert
	raw, err := schema.LoadRaw(inputFile)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}
	s := schema.ConvertRawToSchema(raw)

	formats := []struct {
		format   string
		ext      string
		tsvSplit bool
	}{
		{"json", ".json", false},
		{"ndjson", ".ndjson", false},
		{"tsv", ".tsv", false},
	}

	for _, f := range formats {
		t.Run(f.format, func(t *testing.T) {
			outFile := filepath.Join(dir, "output_"+f.format+f.ext)
			err := exportSchema(s, f.format, outFile, f.tsvSplit)
			if err != nil {
				t.Fatalf("exportSchema() error: %v", err)
			}

			if _, err := os.Stat(outFile); os.IsNotExist(err) {
				t.Error("output file not created")
			}
		})
	}
}

func TestExportSchema_TSVSplit(t *testing.T) {
	dir := t.TempDir()

	inputFile := filepath.Join(dir, "input.json")
	content := `{
		"database_name": "testdb",
		"tables": [{"table": "users"}],
		"columns": [{"table": "users", "name": "id", "type": "int", "ordinal_position": 1}],
		"pk_info": [],
		"fk_info": [],
		"indexes": [],
		"views": []
	}`
	os.WriteFile(inputFile, []byte(content), 0644)

	raw, _ := schema.LoadRaw(inputFile)
	s := schema.ConvertRawToSchema(raw)
	outDir := filepath.Join(dir, "split")

	err := exportSchema(s, "tsv", outDir, true)
	if err != nil {
		t.Fatalf("exportSchema() error: %v", err)
	}

	// Check split files exist
	expectedFiles := []string{"tables.tsv", "columns.tsv"}
	for _, f := range expectedFiles {
		if _, err := os.Stat(filepath.Join(outDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

func TestExportSchema_InvalidFormat(t *testing.T) {
	dir := t.TempDir()

	inputFile := filepath.Join(dir, "input.json")
	content := `{"database_name": "testdb", "tables": [], "columns": [], "pk_info": [], "fk_info": [], "indexes": [], "views": []}`
	os.WriteFile(inputFile, []byte(content), 0644)

	raw, _ := schema.LoadRaw(inputFile)
	s := schema.ConvertRawToSchema(raw)
	outFile := filepath.Join(dir, "output.txt")

	err := exportSchema(s, "invalid", outFile, false)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}
