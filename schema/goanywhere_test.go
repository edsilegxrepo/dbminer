// Test strategy: Realistic data tests using actual GoAnywhere MFT schema structure.
// Sample fixture (10 tables) tests core functionality; full schema (210 tables via env var)
// validates scale handling. Tests FK chains, composite PKs, nullable formats, and all export formats.
package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoAnywhereSchema tests with realistic GoAnywhere MFT schema data
func TestGoAnywhereSchema_Load(t *testing.T) {
	raw, err := LoadRaw("../testdata/goanywhere_sample.json")
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	// Verify basic structure
	if raw.DatabaseName != "goanydb" {
		t.Errorf("expected database name goanydb, got %s", raw.DatabaseName)
	}
	if !strings.Contains(raw.Version, "8.4") {
		t.Errorf("expected MySQL 8.4.x, got %s", raw.Version)
	}
	if len(raw.Tables) != 10 {
		t.Errorf("expected 10 tables, got %d", len(raw.Tables))
	}
	if len(raw.FKInfo) != 7 {
		t.Errorf("expected 7 foreign keys, got %d", len(raw.FKInfo))
	}
}

func TestGoAnywhereSchema_Convert(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	// Verify conversion
	if schema.Name != "goanydb" {
		t.Errorf("expected schema name goanydb, got %s", schema.Name)
	}
	if schema.DatabaseType != "mysql" {
		t.Errorf("expected mysql database type, got %s", schema.DatabaseType)
	}
	if len(schema.Tables) != 10 {
		t.Errorf("expected 10 tables, got %d", len(schema.Tables))
	}
	if len(schema.Relationships) != 7 {
		t.Errorf("expected 7 relationships, got %d", len(schema.Relationships))
	}
}

func TestGoAnywhereSchema_PKDetection(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	// Find dpa_web_user table
	var webUser *Table
	for i := range schema.Tables {
		if schema.Tables[i].Name == "dpa_web_user" {
			webUser = &schema.Tables[i]
			break
		}
	}

	if webUser == nil {
		t.Fatal("dpa_web_user table not found")
	}

	// Verify user_id is PK
	var userIdField *Field
	for i := range webUser.Fields {
		if webUser.Fields[i].Name == "user_id" {
			userIdField = &webUser.Fields[i]
			break
		}
	}

	if userIdField == nil {
		t.Fatal("user_id field not found")
	}
	if !userIdField.PrimaryKey {
		t.Error("user_id should be marked as primary key")
	}
	if !userIdField.Unique {
		t.Error("user_id should be marked as unique (PK implies unique)")
	}
}

func TestGoAnywhereSchema_CompositePK(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	// Find dpa_addr_book_con_group_map table (has composite PK)
	var mapTable *Table
	for i := range schema.Tables {
		if schema.Tables[i].Name == "dpa_addr_book_con_group_map" {
			mapTable = &schema.Tables[i]
			break
		}
	}

	if mapTable == nil {
		t.Fatal("dpa_addr_book_con_group_map table not found")
	}

	// Both contact_id and group_id should be PK
	pkCount := 0
	for _, f := range mapTable.Fields {
		if f.PrimaryKey {
			pkCount++
		}
	}

	if pkCount != 2 {
		t.Errorf("expected 2 PK columns in composite PK, got %d", pkCount)
	}
}

func TestGoAnywhereSchema_FKChain(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	// Verify FK chain: job_log -> job -> project -> web_user
	// Build relationship map
	relsBySource := make(map[string][]Relationship)
	for _, rel := range schema.Relationships {
		relsBySource[rel.SourceTableID] = append(relsBySource[rel.SourceTableID], rel)
	}

	// Build table name to ID map
	tableNameToID := make(map[string]string)
	for _, t := range schema.Tables {
		tableNameToID[t.Name] = t.ID
	}

	// Verify job_log references job
	jobLogID := tableNameToID["dpa_job_log"]
	jobID := tableNameToID["dpa_job"]
	projectID := tableNameToID["dpa_project"]
	webUserID := tableNameToID["dpa_web_user"]

	foundJobLogToJob := false
	for _, rel := range relsBySource[jobLogID] {
		if rel.TargetTableID == jobID {
			foundJobLogToJob = true
			break
		}
	}
	if !foundJobLogToJob {
		t.Error("missing FK: job_log -> job")
	}

	foundJobToProject := false
	for _, rel := range relsBySource[jobID] {
		if rel.TargetTableID == projectID {
			foundJobToProject = true
			break
		}
	}
	if !foundJobToProject {
		t.Error("missing FK: job -> project")
	}

	foundProjectToUser := false
	for _, rel := range relsBySource[projectID] {
		if rel.TargetTableID == webUserID {
			foundProjectToUser = true
			break
		}
	}
	if !foundProjectToUser {
		t.Error("missing FK: project -> web_user")
	}
}

func TestGoAnywhereSchema_IndexDetection(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	// Find dpa_web_user table
	var webUser *Table
	for i := range schema.Tables {
		if schema.Tables[i].Name == "dpa_web_user" {
			webUser = &schema.Tables[i]
			break
		}
	}

	if webUser == nil {
		t.Fatal("dpa_web_user table not found")
	}

	// Should have PRIMARY and idx_username indexes
	if len(webUser.Indexes) < 2 {
		t.Errorf("expected at least 2 indexes, got %d", len(webUser.Indexes))
	}

	// Find idx_username
	var usernameIdx *Index
	for i := range webUser.Indexes {
		if webUser.Indexes[i].Name == "idx_username" {
			usernameIdx = &webUser.Indexes[i]
			break
		}
	}

	if usernameIdx == nil {
		t.Error("idx_username index not found")
	} else if !usernameIdx.Unique {
		t.Error("idx_username should be unique")
	}
}

func TestGoAnywhereSchema_UniqueFromIndex(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	// Find dpa_web_user.username - should be unique from idx_username
	var webUser *Table
	for i := range schema.Tables {
		if schema.Tables[i].Name == "dpa_web_user" {
			webUser = &schema.Tables[i]
			break
		}
	}

	var usernameField *Field
	for i := range webUser.Fields {
		if webUser.Fields[i].Name == "username" {
			usernameField = &webUser.Fields[i]
			break
		}
	}

	if usernameField == nil {
		t.Fatal("username field not found")
	}
	if !usernameField.Unique {
		t.Error("username should be marked unique (from idx_username)")
	}
}

func TestGoAnywhereSchema_NullableHandling(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	// Find dpa_web_user table
	var webUser *Table
	for i := range schema.Tables {
		if schema.Tables[i].Name == "dpa_web_user" {
			webUser = &schema.Tables[i]
			break
		}
	}

	fieldNullable := make(map[string]bool)
	for _, f := range webUser.Fields {
		fieldNullable[f.Name] = f.Nullable
	}

	// user_id should NOT be nullable (PK)
	if fieldNullable["user_id"] {
		t.Error("user_id should not be nullable")
	}
	// username should NOT be nullable
	if fieldNullable["username"] {
		t.Error("username should not be nullable")
	}
	// email should be nullable
	if !fieldNullable["email"] {
		t.Error("email should be nullable")
	}
	// last_login should be nullable
	if !fieldNullable["last_login"] {
		t.Error("last_login should be nullable")
	}
}

func TestGoAnywhereSchema_DefaultValues(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	// Find dpa_web_user.is_active - should have default "1"
	var webUser *Table
	for i := range schema.Tables {
		if schema.Tables[i].Name == "dpa_web_user" {
			webUser = &schema.Tables[i]
			break
		}
	}

	var isActiveField *Field
	for i := range webUser.Fields {
		if webUser.Fields[i].Name == "is_active" {
			isActiveField = &webUser.Fields[i]
			break
		}
	}

	if isActiveField == nil {
		t.Fatal("is_active field not found")
	}
	if isActiveField.Default != "1" {
		t.Errorf("is_active default should be '1', got %s", isActiveField.Default)
	}

	// Find created_date with CURRENT_TIMESTAMP default
	var createdField *Field
	for i := range webUser.Fields {
		if webUser.Fields[i].Name == "created_date" {
			createdField = &webUser.Fields[i]
			break
		}
	}

	if createdField == nil {
		t.Fatal("created_date field not found")
	}
	if createdField.Default != "CURRENT_TIMESTAMP" {
		t.Errorf("created_date default should be 'CURRENT_TIMESTAMP', got %s", createdField.Default)
	}
}

func TestGoAnywhereSchema_ExportJSON(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outFile := filepath.Join(dir, "goanywhere_export.json")

	err := Export(schema, outFile, ExportOptions{Format: FormatJSON})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	// Load exported JSON and verify structure
	data, _ := os.ReadFile(outFile)
	var exported Schema
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if exported.Name != "goanydb" {
		t.Error("exported name mismatch")
	}
	if len(exported.Tables) != 10 {
		t.Errorf("exported %d tables, expected 10", len(exported.Tables))
	}
	if len(exported.Relationships) != 7 {
		t.Errorf("exported %d relationships, expected 7", len(exported.Relationships))
	}
}

func TestGoAnywhereSchema_ExportNDJSON(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outFile := filepath.Join(dir, "goanywhere_export.ndjson")

	err := Export(schema, outFile, ExportOptions{Format: FormatNDJSON})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// Should have: 1 metadata + 10 tables + 7 relationships = 18 lines
	expectedLines := 18
	if len(lines) != expectedLines {
		t.Errorf("expected %d NDJSON lines, got %d", expectedLines, len(lines))
	}

	// Verify each line is valid JSON with _type
	for i, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d is invalid JSON: %v", i, err)
		}
		if _, ok := obj["_type"]; !ok {
			t.Errorf("line %d missing _type field", i)
		}
	}
}

func TestGoAnywhereSchema_ExportTSV(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outFile := filepath.Join(dir, "goanywhere_export.tsv")

	err := Export(schema, outFile, ExportOptions{Format: FormatTSV})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(string(data), "\n")

	// Count record types
	typeCount := make(map[string]int)
	for _, line := range lines[1:] { // Skip header
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) > 0 {
			typeCount[parts[0]]++
		}
	}

	if typeCount["TABLE"] != 10 {
		t.Errorf("expected 10 TABLE rows, got %d", typeCount["TABLE"])
	}
	// Columns: we have ~40 columns in the sample
	if typeCount["COLUMN"] < 30 {
		t.Errorf("expected 30+ COLUMN rows, got %d", typeCount["COLUMN"])
	}
}

func TestGoAnywhereSchema_ExportTSVSplit(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	schema := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outDir := filepath.Join(dir, "split")

	err := Export(schema, outDir, ExportOptions{Format: FormatTSV, TSVSplit: true})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	// Verify all expected files exist
	expectedFiles := []string{"tables.tsv", "columns.tsv", "indexes.tsv", "relationships.tsv"}
	for _, f := range expectedFiles {
		path := filepath.Join(outDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify tables.tsv has correct count
	tablesData, _ := os.ReadFile(filepath.Join(outDir, "tables.tsv"))
	tablesLines := strings.Split(strings.TrimSpace(string(tablesData)), "\n")
	if len(tablesLines) != 11 { // header + 10 tables
		t.Errorf("expected 11 lines in tables.tsv, got %d", len(tablesLines))
	}

	// Verify relationships.tsv has correct count
	relsData, _ := os.ReadFile(filepath.Join(outDir, "relationships.tsv"))
	relsLines := strings.Split(strings.TrimSpace(string(relsData)), "\n")
	if len(relsLines) != 8 { // header + 7 relationships
		t.Errorf("expected 8 lines in relationships.tsv, got %d", len(relsLines))
	}
}

// TestGoAnywhereSchema_FullFile tests against the complete GoAnywhere schema
// Set GOANYWHERE_SCHEMA_PATH env var to run this test
func TestGoAnywhereSchema_FullFile(t *testing.T) {
	fullPath := os.Getenv("GOANYWHERE_SCHEMA_PATH")
	if fullPath == "" {
		fullPath = filepath.Join("..", "goanywhere_schema_raw.json")
	}
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Skipf("Full GoAnywhere schema file not available at %s (set GOANYWHERE_SCHEMA_PATH)", fullPath)
	}

	raw, err := LoadRaw(fullPath)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	// Verify scale
	if len(raw.Tables) < 200 {
		t.Errorf("expected 200+ tables, got %d", len(raw.Tables))
	}
	if len(raw.Columns) < 1500 {
		t.Errorf("expected 1500+ columns, got %d", len(raw.Columns))
	}

	// Convert and verify
	schema := ConvertRawToSchema(raw)
	if len(schema.Tables) != len(raw.Tables) {
		t.Errorf("table count mismatch: raw=%d, schema=%d", len(raw.Tables), len(schema.Tables))
	}

	// Export to all formats
	dir := t.TempDir()

	formats := []struct {
		format ExportFormat
		file   string
	}{
		{FormatJSON, "full.json"},
		{FormatNDJSON, "full.ndjson"},
		{FormatTSV, "full.tsv"},
	}

	for _, f := range formats {
		t.Run(string(f.format), func(t *testing.T) {
			outFile := filepath.Join(dir, f.file)
			err := Export(schema, outFile, ExportOptions{Format: f.format})
			if err != nil {
				t.Errorf("Export(%s) error: %v", f.format, err)
			}

			info, _ := os.Stat(outFile)
			if info.Size() < 1000 {
				t.Errorf("Export(%s) output too small: %d bytes", f.format, info.Size())
			}
		})
	}
}
