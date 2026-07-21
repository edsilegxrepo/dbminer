package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRaw_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "test.json")

	content := `{
		"database_name": "testdb",
		"version": "MySQL 8.0",
		"tables": [{"schema": "public", "table": "users", "type": "TABLE", "rows": 100}],
		"columns": [
			{"schema": "public", "table": "users", "name": "id", "type": "int", "ordinal_position": 1}
		],
		"pk_info": [{"schema": "public", "table": "users", "column": "id"}],
		"fk_info": [],
		"indexes": [],
		"views": []
	}`

	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := LoadRaw(fixture)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	if raw.DatabaseName != "testdb" {
		t.Errorf("expected database_name testdb, got %s", raw.DatabaseName)
	}
	if raw.Version != "MySQL 8.0" {
		t.Errorf("expected version MySQL 8.0, got %s", raw.Version)
	}
	if len(raw.Tables) != 1 {
		t.Errorf("expected 1 table, got %d", len(raw.Tables))
	}
	if len(raw.Columns) != 1 {
		t.Errorf("expected 1 column, got %d", len(raw.Columns))
	}
}

func TestLoadRaw_AllFields(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "full.json")

	content := `{
		"database_name": "fulldb",
		"version": "PostgreSQL 15",
		"tables": [
			{"schema": "public", "table": "users", "type": "TABLE", "rows": 1000, "engine": "InnoDB", "collation": "utf8mb4", "comment": "User table"}
		],
		"columns": [
			{"schema": "public", "table": "users", "name": "id", "type": "int", "character_maximum_length": "", "precision": 10, "ordinal_position": 1, "nullable": false, "default": "", "collation": "", "is_identity": true, "comment": "PK", "generated_expression": "", "generated_type": ""}
		],
		"pk_info": [{"schema": "public", "table": "users", "column": "id", "pk_def": "PRIMARY KEY (id)"}],
		"fk_info": [
			{"schema": "public", "table": "orders", "column": "user_id", "foreign_key_name": "fk_user", "reference_schema": "public", "reference_table": "users", "reference_column": "id", "fk_def": "FOREIGN KEY ..."}
		],
		"indexes": [
			{"schema": "public", "table": "users", "name": "idx_email", "size": 1024, "column": "email", "index_type": "BTREE", "cardinality": 500, "direction": "ASC", "column_position": 1, "unique": true}
		],
		"views": [
			{"schema": "public", "view_name": "active_users", "view_definition": "SELECT * FROM users WHERE active"}
		],
		"triggers": [
			{"schema": "public", "table": "users", "name": "trg_audit", "timing": "AFTER", "event": "INSERT", "statement": "..."}
		],
		"stored_procedures": [
			{"schema": "public", "name": "get_user", "type": "FUNCTION", "return_type": "TABLE", "parameters": [{"name": "id", "type": "int", "mode": "IN", "position": 1}], "comment": "Gets user"}
		],
		"check_constraints": [
			{"schema": "public", "table": "users", "expression": "age > 0"}
		],
		"custom_types": [
			{"schema": "public", "type": "mood", "kind": "enum", "values": ["happy", "sad"]}
		]
	}`

	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := LoadRaw(fixture)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	if raw.DatabaseName != "fulldb" {
		t.Error("wrong database name")
	}
	if len(raw.Tables) != 1 {
		t.Error("wrong tables count")
	}
	if len(raw.Columns) != 1 {
		t.Error("wrong columns count")
	}
	if len(raw.PKInfo) != 1 {
		t.Error("wrong pk_info count")
	}
	if len(raw.FKInfo) != 1 {
		t.Error("wrong fk_info count")
	}
	if len(raw.Indexes) != 1 {
		t.Error("wrong indexes count")
	}
	if len(raw.Views) != 1 {
		t.Error("wrong views count")
	}
	if len(raw.Triggers) != 1 {
		t.Error("wrong triggers count")
	}
	if len(raw.StoredProcedures) != 1 {
		t.Error("wrong stored_procedures count")
	}
	if len(raw.CheckConstraints) != 1 {
		t.Error("wrong check_constraints count")
	}
	if len(raw.CustomTypes) != 1 {
		t.Error("wrong custom_types count")
	}

	// Verify trigger fields
	trig := raw.Triggers[0]
	if trig.Name != "trg_audit" || trig.Timing != "AFTER" || trig.Event != "INSERT" {
		t.Error("trigger fields not parsed correctly")
	}

	// Verify stored procedure fields
	sp := raw.StoredProcedures[0]
	if sp.Name != "get_user" || sp.Type != "FUNCTION" || len(sp.Parameters) != 1 {
		t.Error("stored procedure fields not parsed correctly")
	}

	// Verify custom type fields
	ct := raw.CustomTypes[0]
	if ct.Type != "mood" || ct.Kind != "enum" || len(ct.Values) != 2 {
		t.Error("custom type fields not parsed correctly")
	}
}

func TestLoadRaw_FileNotFound(t *testing.T) {
	_, err := LoadRaw("/nonexistent/path/file.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadRaw_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "invalid.json")

	if err := os.WriteFile(fixture, []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadRaw(fixture)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadRaw_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "empty.json")

	if err := os.WriteFile(fixture, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadRaw(fixture)
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestLoadRaw_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "empty_obj.json")

	if err := os.WriteFile(fixture, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := LoadRaw(fixture)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	if raw.DatabaseName != "" {
		t.Errorf("expected empty database name, got %s", raw.DatabaseName)
	}
}

func TestLoadRaw_NullableFieldTypes(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "nullable.json")

	// Test various nullable field representations
	content := `{
		"database_name": "testdb",
		"tables": [],
		"columns": [
			{"table": "t1", "name": "c1", "type": "int", "ordinal_position": 1, "nullable": true},
			{"table": "t1", "name": "c2", "type": "int", "ordinal_position": 2, "nullable": "true"},
			{"table": "t1", "name": "c3", "type": "int", "ordinal_position": 3, "nullable": 1},
			{"table": "t1", "name": "c4", "type": "int", "ordinal_position": 4, "nullable": "1"},
			{"table": "t1", "name": "c5", "type": "int", "ordinal_position": 5, "nullable": false},
			{"table": "t1", "name": "c6", "type": "int", "ordinal_position": 6, "nullable": "0"}
		],
		"pk_info": [],
		"fk_info": [],
		"indexes": [],
		"views": []
	}`

	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := LoadRaw(fixture)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	if len(raw.Columns) != 6 {
		t.Fatalf("expected 6 columns, got %d", len(raw.Columns))
	}
}

func TestLoadRaw_IndexSizeVariants(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "index_sizes.json")

	content := `{
		"database_name": "testdb",
		"tables": [],
		"columns": [],
		"pk_info": [],
		"fk_info": [],
		"indexes": [
			{"table": "t1", "name": "idx1", "column": "c1", "column_position": 1, "size": 1024},
			{"table": "t1", "name": "idx2", "column": "c2", "column_position": 1, "size": "2048"},
			{"table": "t1", "name": "idx3", "column": "c3", "column_position": 1, "size": null}
		],
		"views": []
	}`

	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := LoadRaw(fixture)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	if len(raw.Indexes) != 3 {
		t.Fatalf("expected 3 indexes, got %d", len(raw.Indexes))
	}
}

func TestLoadRaw_PrecisionVariants(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "precision.json")

	content := `{
		"database_name": "testdb",
		"tables": [],
		"columns": [
			{"table": "t1", "name": "c1", "type": "decimal", "ordinal_position": 1, "precision": 10},
			{"table": "t1", "name": "c2", "type": "decimal", "ordinal_position": 2, "precision": "15"},
			{"table": "t1", "name": "c3", "type": "decimal", "ordinal_position": 3, "precision": null}
		],
		"pk_info": [],
		"fk_info": [],
		"indexes": [],
		"views": []
	}`

	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := LoadRaw(fixture)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	if len(raw.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(raw.Columns))
	}
}

func TestLoadRaw_UnicodeContent(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "unicode.json")

	content := `{
		"database_name": "日本語DB",
		"version": "MySQL 8.0",
		"tables": [{"table": "用户表", "comment": "用户信息 🎉"}],
		"columns": [{"table": "用户表", "name": "姓名", "type": "varchar", "ordinal_position": 1, "comment": "名前"}],
		"pk_info": [],
		"fk_info": [],
		"indexes": [],
		"views": []
	}`

	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	raw, err := LoadRaw(fixture)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	if raw.DatabaseName != "日本語DB" {
		t.Errorf("expected database name 日本語DB, got %s", raw.DatabaseName)
	}
	if raw.Tables[0].Comment != "用户信息 🎉" {
		t.Errorf("expected comment with emoji, got %s", raw.Tables[0].Comment)
	}
}
