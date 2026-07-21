// Test strategy: Export structure validation with realistic GoAnywhere data.
// Validates field-level structure of NDJSON and TSV outputs, not just file existence.
// Tests FK references in TSV, record type counts in NDJSON, and data roundtrip integrity.
package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoAnywhereExport_NDJSON_Structure validates NDJSON output structure
func TestGoAnywhereExport_NDJSON_Structure(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	s := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.ndjson")

	err := Export(s, outFile, ExportOptions{Format: FormatNDJSON})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// Parse all records
	var metadata map[string]interface{}
	var tables []map[string]interface{}
	var relationships []map[string]interface{}

	for i, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("line %d invalid JSON: %v", i, err)
		}

		switch obj["_type"] {
		case "metadata":
			metadata = obj
		case "table":
			tables = append(tables, obj)
		case "relationship":
			relationships = append(relationships, obj)
		}
	}

	// Validate metadata
	t.Run("metadata", func(t *testing.T) {
		if metadata == nil {
			t.Fatal("missing metadata record")
		}
		if metadata["name"] != "goanydb" {
			t.Errorf("expected name goanydb, got %v", metadata["name"])
		}
		if metadata["databaseType"] != "mysql" {
			t.Errorf("expected databaseType mysql, got %v", metadata["databaseType"])
		}
		if int(metadata["tableCount"].(float64)) != 10 {
			t.Errorf("expected tableCount 10, got %v", metadata["tableCount"])
		}
		if int(metadata["relCount"].(float64)) != 7 {
			t.Errorf("expected relCount 7, got %v", metadata["relCount"])
		}
	})

	// Validate tables
	t.Run("tables", func(t *testing.T) {
		if len(tables) != 10 {
			t.Errorf("expected 10 tables, got %d", len(tables))
		}

		// Find dpa_web_user
		var webUser map[string]interface{}
		for _, tbl := range tables {
			if tbl["name"] == "dpa_web_user" {
				webUser = tbl
				break
			}
		}

		if webUser == nil {
			t.Fatal("dpa_web_user table not found")
		}

		// Verify fields array exists and has content
		fields, ok := webUser["fields"].([]interface{})
		if !ok {
			t.Fatal("fields not an array")
		}
		if len(fields) != 6 {
			t.Errorf("expected 6 fields in dpa_web_user, got %d", len(fields))
		}

		// Verify first field structure
		field0 := fields[0].(map[string]interface{})
		if field0["name"] != "user_id" {
			t.Errorf("expected first field user_id, got %v", field0["name"])
		}
		if field0["primaryKey"] != true {
			t.Error("user_id should be primaryKey")
		}

		// Verify indexes
		indexes, ok := webUser["indexes"].([]interface{})
		if !ok {
			t.Fatal("indexes not an array")
		}
		if len(indexes) < 2 {
			t.Errorf("expected at least 2 indexes, got %d", len(indexes))
		}
	})

	// Validate relationships
	t.Run("relationships", func(t *testing.T) {
		if len(relationships) != 7 {
			t.Errorf("expected 7 relationships, got %d", len(relationships))
		}

		// Verify FK structure
		for _, rel := range relationships {
			if rel["sourceTableId"] == nil {
				t.Error("relationship missing sourceTableId")
			}
			if rel["targetTableId"] == nil {
				t.Error("relationship missing targetTableId")
			}
			if rel["sourceFieldId"] == nil {
				t.Error("relationship missing sourceFieldId")
			}
			if rel["targetFieldId"] == nil {
				t.Error("relationship missing targetFieldId")
			}
		}
	})
}

// TestGoAnywhereExport_TSV_Structure validates TSV output structure
func TestGoAnywhereExport_TSV_Structure(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	s := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.tsv")

	err := Export(s, outFile, ExportOptions{Format: FormatTSV})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// Parse header
	header := strings.Split(lines[0], "\t")
	expectedHeader := []string{"type", "table_schema", "table_name", "column_name", "data_type", "is_pk", "is_nullable", "default", "comment", "fk_ref_table", "fk_ref_column"}

	t.Run("header", func(t *testing.T) {
		for i, col := range expectedHeader {
			if i >= len(header) || header[i] != col {
				t.Errorf("header[%d] expected %s, got %s", i, col, header[i])
			}
		}
	})

	// Parse rows by type
	tableRows := [][]string{}
	columnRows := [][]string{}
	indexRows := [][]string{}

	for _, line := range lines[1:] {
		cols := strings.Split(line, "\t")
		if len(cols) < 3 {
			continue
		}
		switch cols[0] {
		case "TABLE":
			tableRows = append(tableRows, cols)
		case "COLUMN":
			columnRows = append(columnRows, cols)
		case "INDEX":
			indexRows = append(indexRows, cols)
		}
	}

	t.Run("tables", func(t *testing.T) {
		if len(tableRows) != 10 {
			t.Errorf("expected 10 TABLE rows, got %d", len(tableRows))
		}

		// Find dpa_web_user
		var webUserRow []string
		for _, row := range tableRows {
			if len(row) > 2 && row[2] == "dpa_web_user" {
				webUserRow = row
				break
			}
		}

		if webUserRow == nil {
			t.Fatal("dpa_web_user TABLE row not found")
		}
		if webUserRow[1] != "goanydb" {
			t.Errorf("expected schema goanydb, got %s", webUserRow[1])
		}
	})

	t.Run("columns", func(t *testing.T) {
		// Should have all columns from all tables
		if len(columnRows) < 30 {
			t.Errorf("expected 30+ COLUMN rows, got %d", len(columnRows))
		}

		// Find user_id column
		var userIdRow []string
		for _, row := range columnRows {
			if len(row) > 3 && row[2] == "dpa_web_user" && row[3] == "user_id" {
				userIdRow = row
				break
			}
		}

		if userIdRow == nil {
			t.Fatal("user_id COLUMN row not found")
		}

		// Verify structure: type, schema, table, column, datatype, is_pk, is_nullable, default, comment, fk_ref_table, fk_ref_column
		if userIdRow[4] != "bigint" {
			t.Errorf("expected type bigint, got %s", userIdRow[4])
		}
		if userIdRow[5] != "Y" {
			t.Errorf("expected is_pk Y, got %s", userIdRow[5])
		}
	})

	t.Run("indexes", func(t *testing.T) {
		if len(indexRows) < 10 {
			t.Errorf("expected 10+ INDEX rows, got %d", len(indexRows))
		}

		// Find idx_username
		var idxUsernameRow []string
		for _, row := range indexRows {
			if len(row) > 3 && row[3] == "idx_username" {
				idxUsernameRow = row
				break
			}
		}

		if idxUsernameRow == nil {
			t.Fatal("idx_username INDEX row not found")
		}
	})

	t.Run("fk_references", func(t *testing.T) {
		// Find job_id column in dpa_job_log - should have FK reference to dpa_job
		var jobIdRow []string
		for _, row := range columnRows {
			if len(row) > 10 && row[2] == "dpa_job_log" && row[3] == "job_id" {
				jobIdRow = row
				break
			}
		}

		if jobIdRow == nil {
			t.Fatal("job_id COLUMN row not found in dpa_job_log")
		}

		// fk_ref_table is column 9, fk_ref_column is column 10
		if jobIdRow[9] != "dpa_job" {
			t.Errorf("expected fk_ref_table dpa_job, got %s", jobIdRow[9])
		}
		if jobIdRow[10] != "job_id" {
			t.Errorf("expected fk_ref_column job_id, got %s", jobIdRow[10])
		}
	})
}

// TestGoAnywhereExport_TSVSplit_Structure validates split TSV files
func TestGoAnywhereExport_TSVSplit_Structure(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	s := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outDir := filepath.Join(dir, "split")

	err := Export(s, outDir, ExportOptions{Format: FormatTSV, TSVSplit: true})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	t.Run("tables.tsv", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(outDir, "tables.tsv"))
		if err != nil {
			t.Fatalf("tables.tsv not found: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		header := lines[0]

		// Verify header: id, schema, name, type, comment
		if !strings.Contains(header, "id\tschema\tname\ttype") {
			t.Errorf("tables.tsv unexpected header: %s", header)
		}

		// Verify row count (header + 10 tables)
		if len(lines) != 11 {
			t.Errorf("expected 11 lines (header + 10 tables), got %d", len(lines))
		}

		// Verify dpa_web_user row - format: id, schema, name, type, comment
		found := false
		for _, line := range lines[1:] {
			if strings.Contains(line, "dpa_web_user") {
				found = true
				cols := strings.Split(line, "\t")
				// cols[1] is schema, cols[2] is name
				if cols[1] != "goanydb" {
					t.Errorf("expected schema goanydb, got %s", cols[1])
				}
				if cols[2] != "dpa_web_user" {
					t.Errorf("expected name dpa_web_user, got %s", cols[2])
				}
				break
			}
		}
		if !found {
			t.Error("dpa_web_user not found in tables.tsv")
		}
	})

	t.Run("columns.tsv", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(outDir, "columns.tsv"))
		if err != nil {
			t.Fatalf("columns.tsv not found: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		header := lines[0]

		// Verify header: id, table_id, table_name, name, type, is_pk, is_unique, is_nullable, default, ...
		if !strings.Contains(header, "table_name") || !strings.Contains(header, "is_pk") {
			t.Errorf("columns.tsv unexpected header: %s", header)
		}

		// Count columns per table (table_name is col[2])
		tableCols := make(map[string]int)
		for _, line := range lines[1:] {
			cols := strings.Split(line, "\t")
			if len(cols) > 2 {
				tableCols[cols[2]]++
			}
		}

		if tableCols["dpa_web_user"] != 6 {
			t.Errorf("expected 6 columns for dpa_web_user, got %d", tableCols["dpa_web_user"])
		}
	})

	t.Run("indexes.tsv", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(outDir, "indexes.tsv"))
		if err != nil {
			t.Fatalf("indexes.tsv not found: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		header := lines[0]

		// Verify header: id, table_id, table_name, name, is_unique, columns
		if !strings.Contains(header, "table_name") || !strings.Contains(header, "is_unique") {
			t.Errorf("indexes.tsv unexpected header: %s", header)
		}

		// Find idx_username - format: id, table_id, table_name, name, is_unique, columns
		found := false
		for _, line := range lines[1:] {
			if strings.Contains(line, "idx_username") {
				found = true
				cols := strings.Split(line, "\t")
				// cols[4] is is_unique, should be "Y"
				if len(cols) > 4 && cols[4] != "Y" {
					t.Errorf("idx_username is_unique expected Y, got %s", cols[4])
				}
				// cols[5] is columns, should contain "username"
				if len(cols) > 5 && !strings.Contains(cols[5], "username") {
					t.Errorf("idx_username columns expected username, got %s", cols[5])
				}
				break
			}
		}
		if !found {
			t.Error("idx_username not found in indexes.tsv")
		}
	})

	t.Run("relationships.tsv", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(outDir, "relationships.tsv"))
		if err != nil {
			t.Fatalf("relationships.tsv not found: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")

		// Should have header + 7 relationships
		if len(lines) != 8 {
			t.Errorf("expected 8 lines (header + 7 relationships), got %d", len(lines))
		}

		// Verify FK from dpa_job_log to dpa_job
		found := false
		for _, line := range lines[1:] {
			if strings.Contains(line, "dpa_job_log") && strings.Contains(line, "dpa_job") {
				found = true
				break
			}
		}
		if !found {
			t.Error("FK dpa_job_log -> dpa_job not found in relationships.tsv")
		}
	})
}

// TestGoAnywhereExport_NDJSON_Roundtrip verifies data survives export/import
func TestGoAnywhereExport_NDJSON_Roundtrip(t *testing.T) {
	raw, _ := LoadRaw("../testdata/goanywhere_sample.json")
	original := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outFile := filepath.Join(dir, "export.ndjson")

	Export(original, outFile, ExportOptions{Format: FormatNDJSON})

	// Parse NDJSON and rebuild counts
	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	tableCount := 0
	relCount := 0
	fieldCount := 0

	for _, line := range lines {
		var obj map[string]interface{}
		json.Unmarshal([]byte(line), &obj)

		switch obj["_type"] {
		case "table":
			tableCount++
			if fields, ok := obj["fields"].([]interface{}); ok {
				fieldCount += len(fields)
			}
		case "relationship":
			relCount++
		}
	}

	// Verify counts match original
	if tableCount != len(original.Tables) {
		t.Errorf("table count mismatch: original=%d, exported=%d", len(original.Tables), tableCount)
	}
	if relCount != len(original.Relationships) {
		t.Errorf("relationship count mismatch: original=%d, exported=%d", len(original.Relationships), relCount)
	}

	originalFieldCount := 0
	for _, tbl := range original.Tables {
		originalFieldCount += len(tbl.Fields)
	}
	if fieldCount != originalFieldCount {
		t.Errorf("field count mismatch: original=%d, exported=%d", originalFieldCount, fieldCount)
	}
}

// TestGoAnywhereExport_FullSchema_NDJSON tests NDJSON with full 210-table schema
func TestGoAnywhereExport_FullSchema_NDJSON(t *testing.T) {
	fullPath := os.Getenv("GOANYWHERE_SCHEMA_PATH")
	if fullPath == "" {
		t.Skip("Set GOANYWHERE_SCHEMA_PATH to run full schema export test")
	}

	raw, _ := LoadRaw(fullPath)
	s := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outFile := filepath.Join(dir, "full.ndjson")

	err := Export(s, outFile, ExportOptions{Format: FormatNDJSON})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// Count record types
	typeCounts := make(map[string]int)
	for _, line := range lines {
		var obj map[string]interface{}
		json.Unmarshal([]byte(line), &obj)
		if t, ok := obj["_type"].(string); ok {
			typeCounts[t]++
		}
	}

	if typeCounts["metadata"] != 1 {
		t.Errorf("expected 1 metadata record, got %d", typeCounts["metadata"])
	}
	if typeCounts["table"] < 200 {
		t.Errorf("expected 200+ table records, got %d", typeCounts["table"])
	}
	if typeCounts["relationship"] < 100 {
		t.Errorf("expected 100+ relationship records, got %d", typeCounts["relationship"])
	}
}

// TestGoAnywhereExport_FullSchema_TSVSplit tests split TSV with full schema
func TestGoAnywhereExport_FullSchema_TSVSplit(t *testing.T) {
	fullPath := os.Getenv("GOANYWHERE_SCHEMA_PATH")
	if fullPath == "" {
		t.Skip("Set GOANYWHERE_SCHEMA_PATH to run full schema export test")
	}

	raw, _ := LoadRaw(fullPath)
	s := ConvertRawToSchema(raw)

	dir := t.TempDir()
	outDir := filepath.Join(dir, "split")

	err := Export(s, outDir, ExportOptions{Format: FormatTSV, TSVSplit: true})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	// Verify tables.tsv has 200+ rows
	tablesData, _ := os.ReadFile(filepath.Join(outDir, "tables.tsv"))
	tablesLines := strings.Split(strings.TrimSpace(string(tablesData)), "\n")
	if len(tablesLines) < 201 { // header + 200 tables
		t.Errorf("expected 201+ lines in tables.tsv, got %d", len(tablesLines))
	}

	// Verify columns.tsv has 1500+ rows
	columnsData, _ := os.ReadFile(filepath.Join(outDir, "columns.tsv"))
	columnsLines := strings.Split(strings.TrimSpace(string(columnsData)), "\n")
	if len(columnsLines) < 1500 {
		t.Errorf("expected 1500+ lines in columns.tsv, got %d", len(columnsLines))
	}

	// Verify relationships.tsv has 100+ rows
	relsData, _ := os.ReadFile(filepath.Join(outDir, "relationships.tsv"))
	relsLines := strings.Split(strings.TrimSpace(string(relsData)), "\n")
	if len(relsLines) < 100 {
		t.Errorf("expected 100+ lines in relationships.tsv, got %d", len(relsLines))
	}
}
