// Test strategy: I/O tests for export functions using t.TempDir().
// Validates JSON/NDJSON/TSV output structure, content correctness, and special character handling.
// Tests both combined and split TSV modes.
package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestSchema() *Schema {
	return &Schema{
		ID:           "1",
		Name:         "testdb",
		DatabaseType: "mysql",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Tables: []Table{
			{
				ID:     "1",
				Name:   "users",
				Schema: "public",
				Fields: []Field{
					{ID: "1", Name: "id", Type: FieldType{ID: "int", Name: "int"}, PrimaryKey: true},
					{ID: "2", Name: "email", Type: FieldType{ID: "varchar", Name: "varchar"}, Nullable: false, Unique: true},
					{ID: "3", Name: "name", Type: FieldType{ID: "varchar", Name: "varchar"}, Nullable: true},
				},
				Indexes: []Index{
					{ID: "1", Name: "PRIMARY", Unique: true, FieldIDs: []string{"1"}},
					{ID: "2", Name: "idx_email", Unique: true, FieldIDs: []string{"2"}},
				},
			},
			{
				ID:     "2",
				Name:   "orders",
				Schema: "public",
				Fields: []Field{
					{ID: "4", Name: "id", Type: FieldType{ID: "int", Name: "int"}, PrimaryKey: true},
					{ID: "5", Name: "user_id", Type: FieldType{ID: "int", Name: "int"}},
					{ID: "6", Name: "total", Type: FieldType{ID: "decimal", Name: "decimal"}},
				},
			},
		},
		Relationships: []Relationship{
			{
				ID:                "1",
				Name:              "fk_orders_users",
				SourceTableID:     "2",
				TargetTableID:     "1",
				SourceFieldID:     "5",
				TargetFieldID:     "1",
				SourceCardinality: "one",
				TargetCardinality: "many",
			},
		},
		Triggers: []Trigger{
			{ID: "1", Name: "trg_audit", Schema: "public", Table: "users", Timing: "AFTER", Event: "INSERT"},
		},
		StoredProcedures: []StoredProcedure{
			{ID: "1", Name: "get_user", Schema: "public", Type: "FUNCTION", ReturnType: "TABLE"},
		},
	}
}

func TestExportJSON(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.json")

	err := Export(s, outFile, ExportOptions{Format: FormatJSON})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	var result Schema
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if result.Name != "testdb" {
		t.Errorf("expected name testdb, got %s", result.Name)
	}
	if result.DatabaseType != "mysql" {
		t.Errorf("expected databaseType mysql, got %s", result.DatabaseType)
	}
	if len(result.Tables) != 2 {
		t.Errorf("expected 2 tables, got %d", len(result.Tables))
	}
	if len(result.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(result.Relationships))
	}
}

func TestExportJSON_Indentation(t *testing.T) {
	s := &Schema{ID: "1", Name: "test"}
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.json")

	Export(s, outFile, ExportOptions{Format: FormatJSON})

	data, _ := os.ReadFile(outFile)
	content := string(data)

	// Should be indented with 2 spaces
	if !strings.Contains(content, "  \"id\"") {
		t.Error("expected indented JSON output")
	}
}

func TestExportNDJSON(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.ndjson")

	err := Export(s, outFile, ExportOptions{Format: FormatNDJSON})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// metadata + 2 tables + 1 relationship + 1 trigger + 1 stored procedure = 6 lines
	expectedLines := 6
	if len(lines) != expectedLines {
		t.Errorf("expected %d NDJSON lines, got %d", expectedLines, len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}

		// Verify _type field exists
		if _, ok := obj["_type"]; !ok {
			t.Errorf("line %d missing _type field", i)
		}
	}
}

func TestExportNDJSON_RecordTypes(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.ndjson")

	Export(s, outFile, ExportOptions{Format: FormatNDJSON})

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	typesSeen := make(map[string]int)
	for _, line := range lines {
		var obj map[string]interface{}
		json.Unmarshal([]byte(line), &obj)
		if t, ok := obj["_type"].(string); ok {
			typesSeen[t]++
		}
	}

	expectedTypes := map[string]int{
		"metadata":         1,
		"table":            2,
		"relationship":     1,
		"trigger":          1,
		"stored_procedure": 1,
	}

	for typ, count := range expectedTypes {
		if typesSeen[typ] != count {
			t.Errorf("expected %d %s records, got %d", count, typ, typesSeen[typ])
		}
	}
}

func TestExportNDJSON_MetadataContent(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.ndjson")

	Export(s, outFile, ExportOptions{Format: FormatNDJSON})

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	var meta map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &meta)

	if meta["_type"] != "metadata" {
		t.Error("first record should be metadata")
	}
	if meta["name"] != "testdb" {
		t.Errorf("expected name testdb, got %v", meta["name"])
	}
	if int(meta["tableCount"].(float64)) != 2 {
		t.Errorf("expected tableCount 2, got %v", meta["tableCount"])
	}
	if int(meta["relCount"].(float64)) != 1 {
		t.Errorf("expected relCount 1, got %v", meta["relCount"])
	}
}

func TestExportTSV_Combined(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.tsv")

	err := Export(s, outFile, ExportOptions{Format: FormatTSV})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(string(data), "\n")

	// Should have header
	if !strings.HasPrefix(lines[0], "type\t") {
		t.Error("TSV header missing type column")
	}

	// Verify header columns
	expectedHeaders := []string{"type", "table_schema", "table_name", "column_name", "data_type"}
	headers := strings.Split(lines[0], "\t")
	for i, h := range expectedHeaders {
		if i >= len(headers) || headers[i] != h {
			t.Errorf("expected header[%d] = %s, got %s", i, h, headers[i])
		}
	}

	// Count record types
	typeCount := make(map[string]int)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) > 0 {
			typeCount[parts[0]]++
		}
	}

	if typeCount["TABLE"] != 2 {
		t.Errorf("expected 2 TABLE rows, got %d", typeCount["TABLE"])
	}
	if typeCount["COLUMN"] != 6 {
		t.Errorf("expected 6 COLUMN rows, got %d", typeCount["COLUMN"])
	}
}

func TestExportTSV_Split(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outDir := filepath.Join(dir, "split")

	err := Export(s, outDir, ExportOptions{Format: FormatTSV, TSVSplit: true})
	if err != nil {
		t.Fatal(err)
	}

	expectedFiles := []string{"tables.tsv", "columns.tsv", "indexes.tsv", "relationships.tsv"}
	for _, f := range expectedFiles {
		path := filepath.Join(outDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify tables.tsv content
	tablesData, _ := os.ReadFile(filepath.Join(outDir, "tables.tsv"))
	tablesLines := strings.Split(strings.TrimSpace(string(tablesData)), "\n")
	if len(tablesLines) != 3 { // header + 2 tables
		t.Errorf("expected 3 lines in tables.tsv, got %d", len(tablesLines))
	}

	// Verify columns.tsv content
	columnsData, _ := os.ReadFile(filepath.Join(outDir, "columns.tsv"))
	columnsLines := strings.Split(strings.TrimSpace(string(columnsData)), "\n")
	if len(columnsLines) != 7 { // header + 6 columns
		t.Errorf("expected 7 lines in columns.tsv, got %d", len(columnsLines))
	}

	// Verify relationships.tsv content
	relsData, _ := os.ReadFile(filepath.Join(outDir, "relationships.tsv"))
	relsLines := strings.Split(strings.TrimSpace(string(relsData)), "\n")
	if len(relsLines) != 2 { // header + 1 relationship
		t.Errorf("expected 2 lines in relationships.tsv, got %d", len(relsLines))
	}
}

func TestExportTSV_SplitWithTriggers(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outDir := filepath.Join(dir, "split")

	Export(s, outDir, ExportOptions{Format: FormatTSV, TSVSplit: true})

	triggersPath := filepath.Join(outDir, "triggers.tsv")
	if _, err := os.Stat(triggersPath); os.IsNotExist(err) {
		t.Error("triggers.tsv not created")
	}

	data, _ := os.ReadFile(triggersPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 { // header + 1 trigger
		t.Errorf("expected 2 lines in triggers.tsv, got %d", len(lines))
	}
}

func TestExportTSV_SplitWithProcedures(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outDir := filepath.Join(dir, "split")

	Export(s, outDir, ExportOptions{Format: FormatTSV, TSVSplit: true})

	procsPath := filepath.Join(outDir, "procedures.tsv")
	if _, err := os.Stat(procsPath); os.IsNotExist(err) {
		t.Error("procedures.tsv not created")
	}

	data, _ := os.ReadFile(procsPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 { // header + 1 procedure
		t.Errorf("expected 2 lines in procedures.tsv, got %d", len(lines))
	}
}

func TestExport_InvalidFormat(t *testing.T) {
	s := &Schema{Name: "test"}
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out")

	err := Export(s, outFile, ExportOptions{Format: "invalid"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestExport_WriteError(t *testing.T) {
	s := &Schema{Name: "test"}

	// Try to write to an invalid path (null byte makes it invalid on all platforms)
	err := Export(s, "/nonexistent\x00invalid/file.json", ExportOptions{Format: FormatJSON})
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestExportTSV_SpecialCharacters(t *testing.T) {
	s := &Schema{
		Name: "testdb",
		Tables: []Table{
			{
				ID:      "1",
				Name:    "users",
				Comment: "Contains\ttabs\tand\nnewlines",
				Fields: []Field{
					{ID: "1", Name: "data", Type: FieldType{Name: "text"}, Comment: "Tab:\there\nNewline:\nhere"},
				},
			},
		},
	}

	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.tsv")

	Export(s, outFile, ExportOptions{Format: FormatTSV})

	data, _ := os.ReadFile(outFile)
	content := string(data)

	// Tabs and newlines should be escaped/replaced
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Count(line, "\t") > 10 { // More tabs than expected columns
			t.Errorf("line %d has unescaped tabs", i)
		}
	}
}

func TestExportTSV_ForeignKeyInfo(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.tsv")

	Export(s, outFile, ExportOptions{Format: FormatTSV})

	data, _ := os.ReadFile(outFile)
	content := string(data)

	// user_id column should have FK reference info
	if !strings.Contains(content, "users") {
		t.Error("TSV should contain FK reference table name")
	}
}

func TestExportTSV_PKIndicator(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.tsv")

	Export(s, outFile, ExportOptions{Format: FormatTSV})

	data, _ := os.ReadFile(outFile)
	content := string(data)

	// Should have Y for PK columns
	if !strings.Contains(content, "\tY\t") {
		t.Error("TSV should contain Y for PK indicator")
	}
}

func TestExport_EmptySchema(t *testing.T) {
	s := &Schema{
		ID:   "1",
		Name: "emptydb",
	}

	dir := t.TempDir()

	// Test all formats with empty schema
	formats := []struct {
		format ExportFormat
		ext    string
	}{
		{FormatJSON, "json"},
		{FormatNDJSON, "ndjson"},
		{FormatTSV, "tsv"},
	}

	for _, f := range formats {
		t.Run(string(f.format), func(t *testing.T) {
			outFile := filepath.Join(dir, "empty."+f.ext)
			err := Export(s, outFile, ExportOptions{Format: f.format})
			if err != nil {
				t.Errorf("Export() error: %v", err)
			}

			if _, err := os.Stat(outFile); os.IsNotExist(err) {
				t.Error("output file not created")
			}
		})
	}
}

func TestExportTSV_IndexInfo(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.tsv")

	Export(s, outFile, ExportOptions{Format: FormatTSV})

	data, _ := os.ReadFile(outFile)
	content := string(data)

	// Should have INDEX rows
	if !strings.Contains(content, "INDEX\t") {
		t.Error("TSV should contain INDEX rows")
	}
}

func TestExportTSV_TriggerInfo(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.tsv")

	Export(s, outFile, ExportOptions{Format: FormatTSV})

	data, _ := os.ReadFile(outFile)
	content := string(data)

	// Should have TRIGGER rows
	if !strings.Contains(content, "TRIGGER\t") {
		t.Error("TSV should contain TRIGGER rows")
	}
}

func TestExportTSV_StoredProcInfo(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.tsv")

	Export(s, outFile, ExportOptions{Format: FormatTSV})

	data, _ := os.ReadFile(outFile)
	content := string(data)

	// Should have FUNCTION rows
	if !strings.Contains(content, "FUNCTION\t") {
		t.Error("TSV should contain FUNCTION rows")
	}
}

func TestEscapeTSV(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"with\ttab", "with tab"},
		{"with\nnewline", "with newline"},
		{"with\rcarriage", "withcarriage"},
		{"all\t\n\rtogether", "all  together"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeTSV(tt.input)
			if got != tt.expected {
				t.Errorf("escapeTSV(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExportNDJSON_NoHTMLEscape(t *testing.T) {
	s := &Schema{
		Name: "test<db>",
		Tables: []Table{
			{ID: "1", Name: "users", Comment: "Has <html> & 'quotes'"},
		},
	}

	dir := t.TempDir()
	outFile := filepath.Join(dir, "schema.ndjson")

	Export(s, outFile, ExportOptions{Format: FormatNDJSON})

	data, _ := os.ReadFile(outFile)
	content := string(data)

	// Should NOT escape HTML entities
	if strings.Contains(content, "\\u003c") {
		t.Error("NDJSON should not HTML-escape < character")
	}
	if strings.Contains(content, "\\u0026") {
		t.Error("NDJSON should not HTML-escape & character")
	}
}

func TestExport_LargeSchema(t *testing.T) {
	// Create a schema with many tables
	s := &Schema{
		ID:   "1",
		Name: "largedb",
	}

	for i := 0; i < 100; i++ {
		table := Table{
			ID:   string(rune('0' + i)),
			Name: "table_" + string(rune('0'+i)),
		}
		for j := 0; j < 20; j++ {
			table.Fields = append(table.Fields, Field{
				ID:   string(rune('0' + i*20 + j)),
				Name: "col_" + string(rune('0'+j)),
				Type: FieldType{Name: "varchar"},
			})
		}
		s.Tables = append(s.Tables, table)
	}

	dir := t.TempDir()

	// Test all formats handle large schema
	formats := []ExportFormat{FormatJSON, FormatNDJSON, FormatTSV}
	for _, f := range formats {
		t.Run(string(f), func(t *testing.T) {
			outFile := filepath.Join(dir, "large."+string(f))
			err := Export(s, outFile, ExportOptions{Format: f})
			if err != nil {
				t.Errorf("Export() error: %v", err)
			}
		})
	}
}
