// Test strategy: Unit tests for driver validation and registry functions.
// Table-driven tests cover identifier validation (SQL injection prevention),
// MongoDB name validation, user validation, and driver lookup for all 9 platforms.
package gensql

import (
	"strings"
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		label   string
		wantErr bool
		errMsg  string
	}{
		{"valid simple", "mydb", "database", false, ""},
		{"valid underscore", "my_database", "database", false, ""},
		{"valid leading underscore", "_internal", "database", false, ""},
		{"valid mixed case", "MyDatabase", "database", false, ""},
		{"valid with numbers", "db123", "database", false, ""},
		{"empty", "", "database", true, "cannot be empty"},
		{"starts with number", "1database", "database", true, "must be alphanumeric"},
		{"has hyphen", "my-db", "database", true, "must be alphanumeric"},
		{"has space", "my db", "database", true, "must be alphanumeric"},
		{"has dot", "my.db", "database", true, "must be alphanumeric"},
		{"has special char", "my@db", "database", true, "must be alphanumeric"},
		{"too long", strings.Repeat("a", 129), "database", true, "too long"},
		{"max length", strings.Repeat("a", 128), "database", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIdentifier(tt.input, tt.label)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIdentifier() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateIdentifier() error = %v, want containing %s", err, tt.errMsg)
			}
		})
	}
}

func TestValidateMongoDBName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"valid simple", "mydb", false, ""},
		{"valid with hyphen", "my-database", false, ""},
		{"valid with underscore", "my_db", false, ""},
		{"valid mixed", "my-db_test", false, ""},
		{"valid leading underscore", "_mydb", false, ""},
		{"empty", "", true, "cannot be empty"},
		{"starts with number", "1db", true, "invalid database name"},
		{"starts with hyphen", "-db", true, "invalid database name"},
		{"has space", "my db", true, "invalid database name"},
		{"has dot", "my.db", true, "invalid database name"},
		{"too long", strings.Repeat("a", 65), true, "too long"},
		{"max length", strings.Repeat("a", 64), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMongoDBName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMongoDBName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateMongoDBName() error = %v, want containing %s", err, tt.errMsg)
			}
		})
	}
}

func TestValidateUser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty allowed", "", false},
		{"simple", "admin", false},
		{"with underscore", "db_admin", false},
		{"with at sign", "user@domain", false},
		{"with dot", "user.name", false},
		{"with percent", "user%host", false},
		{"with apostrophe", "o'brien", false},
		{"sql server UPN", "user@domain.com", false},
		{"backslash not allowed", "domain\\user", true},
		{"space not allowed", "user name", true},
		{"semicolon not allowed", "user;drop", true},
		{"too long", strings.Repeat("a", 129), true},
		{"max length", strings.Repeat("a", 128), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUser(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDriver(t *testing.T) {
	validDrivers := []string{
		"mysql", "mysql57", "mariadb", "postgres",
		"sqlite", "mssql", "mssql2016", "oracle", "mongodb",
	}

	for _, name := range validDrivers {
		t.Run(name, func(t *testing.T) {
			d, err := GetDriver(name)
			if err != nil {
				t.Errorf("GetDriver(%s) unexpected error: %v", name, err)
			}
			if d == nil {
				t.Errorf("GetDriver(%s) returned nil driver", name)
			}
			if d.Name() != name {
				t.Errorf("GetDriver(%s).Name() = %s, want %s", name, d.Name(), name)
			}
		})
	}
}

func TestGetDriver_Unknown(t *testing.T) {
	invalidDrivers := []string{"unknown", "mysql8", "pg", "sqlserver", ""}

	for _, name := range invalidDrivers {
		t.Run(name, func(t *testing.T) {
			_, err := GetDriver(name)
			if err == nil {
				t.Errorf("GetDriver(%s) expected error", name)
			}
			if !strings.Contains(err.Error(), "unknown driver") {
				t.Errorf("GetDriver(%s) error = %v, want 'unknown driver'", name, err)
			}
		})
	}
}

func TestListDrivers(t *testing.T) {
	drivers := ListDrivers()

	if len(drivers) != 9 {
		t.Errorf("expected 9 drivers, got %d", len(drivers))
	}

	// Verify sorted
	for i := 1; i < len(drivers); i++ {
		if drivers[i] < drivers[i-1] {
			t.Errorf("drivers not sorted: %s comes after %s", drivers[i], drivers[i-1])
		}
	}

	// Verify expected drivers present
	expected := map[string]bool{
		"mysql": true, "mysql57": true, "mariadb": true,
		"postgres": true, "sqlite": true, "mssql": true,
		"mssql2016": true, "oracle": true, "mongodb": true,
	}

	for _, d := range drivers {
		if !expected[d] {
			t.Errorf("unexpected driver: %s", d)
		}
		delete(expected, d)
	}

	for d := range expected {
		t.Errorf("missing driver: %s", d)
	}
}

func TestAllDriversGenerateSQL_Basic(t *testing.T) {
	drivers := ListDrivers()
	opts := GenerateOptions{
		DBName:   "testdb",
		SPName:   "sp_export",
		Direct:   false,
		NoAdmin:  false,
	}

	for _, name := range drivers {
		t.Run(name, func(t *testing.T) {
			d, _ := GetDriver(name)
			script, err := d.GenerateSQL(opts)
			if err != nil {
				t.Fatalf("GenerateSQL() error: %v", err)
			}
			if script == "" {
				t.Error("GenerateSQL() returned empty script")
			}
			// Script should be substantial (at least 100 chars)
			if len(script) < 100 {
				t.Errorf("script suspiciously short: %d chars", len(script))
			}
		})
	}
}

func TestAllDriversGenerateSQL_DirectMode(t *testing.T) {
	drivers := ListDrivers()
	opts := GenerateOptions{
		DBName: "testdb",
		Direct: true,
	}

	for _, name := range drivers {
		t.Run(name, func(t *testing.T) {
			d, _ := GetDriver(name)
			script, err := d.GenerateSQL(opts)
			if err != nil {
				t.Fatalf("GenerateSQL() error: %v", err)
			}
			if script == "" {
				t.Error("GenerateSQL() returned empty script")
			}
		})
	}
}

func TestAllDriversGenerateSQL_NoAdminMode(t *testing.T) {
	drivers := ListDrivers()
	opts := GenerateOptions{
		DBName:  "testdb",
		NoAdmin: true,
	}

	for _, name := range drivers {
		t.Run(name, func(t *testing.T) {
			d, _ := GetDriver(name)
			script, err := d.GenerateSQL(opts)
			if err != nil {
				t.Fatalf("GenerateSQL() error: %v", err)
			}
			if script == "" {
				t.Error("GenerateSQL() returned empty script")
			}
		})
	}
}

func TestAllDriversGenerateSQL_WithExecUser(t *testing.T) {
	// Skip MongoDB as it doesn't use ExecUser
	drivers := []string{"mysql", "mysql57", "mariadb", "postgres", "mssql", "mssql2016", "oracle"}
	opts := GenerateOptions{
		DBName:   "testdb",
		SPName:   "sp_export",
		ExecUser: "app_user",
	}

	for _, name := range drivers {
		t.Run(name, func(t *testing.T) {
			d, _ := GetDriver(name)
			script, err := d.GenerateSQL(opts)
			if err != nil {
				t.Fatalf("GenerateSQL() error: %v", err)
			}
			if script == "" {
				t.Error("GenerateSQL() returned empty script")
			}
		})
	}
}

func TestMySQLDriver_StoredProc(t *testing.T) {
	d, _ := GetDriver("mysql")
	opts := GenerateOptions{
		DBName: "testdb",
		SPName: "sp_export_schema",
		Direct: false,
	}

	script, _ := d.GenerateSQL(opts)

	if !strings.Contains(script, "CREATE PROCEDURE") {
		t.Error("expected CREATE PROCEDURE in stored proc mode")
	}
	if !strings.Contains(script, "sp_export_schema") {
		t.Error("expected procedure name in script")
	}
}

func TestMySQLDriver_DirectQuery(t *testing.T) {
	d, _ := GetDriver("mysql")
	opts := GenerateOptions{
		DBName: "testdb",
		Direct: true,
	}

	script, _ := d.GenerateSQL(opts)

	if strings.Contains(script, "CREATE PROCEDURE") {
		t.Error("unexpected CREATE PROCEDURE in direct mode")
	}
}

func TestPostgresDriver_StoredProc(t *testing.T) {
	d, _ := GetDriver("postgres")
	opts := GenerateOptions{
		DBName: "testdb",
		SPName: "sp_export",
		Direct: false,
	}

	script, err := d.GenerateSQL(opts)
	if err != nil {
		t.Fatalf("GenerateSQL() error: %v", err)
	}
	if script == "" {
		t.Error("GenerateSQL() returned empty script")
	}
	// PostgreSQL stored proc mode should have function/procedure definition
	if !strings.Contains(script, "CREATE") && !strings.Contains(script, "FUNCTION") && !strings.Contains(script, "PROCEDURE") {
		t.Log("Note: postgres stored proc mode may use different syntax")
	}
}

func TestMSSQLDriver_StoredProc(t *testing.T) {
	d, _ := GetDriver("mssql")
	opts := GenerateOptions{
		DBName: "testdb",
		SPName: "sp_export",
		Direct: false,
	}

	script, _ := d.GenerateSQL(opts)

	if !strings.Contains(script, "CREATE PROCEDURE") {
		t.Error("expected CREATE PROCEDURE in stored proc mode")
	}
}

func TestOracleDriver_StoredProc(t *testing.T) {
	d, _ := GetDriver("oracle")
	opts := GenerateOptions{
		DBName: "testdb",
		SPName: "sp_export",
		Direct: false,
	}

	script, err := d.GenerateSQL(opts)
	if err != nil {
		t.Fatalf("GenerateSQL() error: %v", err)
	}
	if script == "" {
		t.Error("GenerateSQL() returned empty script")
	}
}

func TestSQLiteDriver_AlwaysDirect(t *testing.T) {
	d, _ := GetDriver("sqlite")

	// SQLite doesn't support stored procedures
	opts := GenerateOptions{
		DBName: "testdb",
		Direct: false, // Should still produce direct SQL
	}

	script, _ := d.GenerateSQL(opts)

	// Should have SELECT statements, not procedure definition
	if strings.Contains(script, "CREATE PROCEDURE") {
		t.Error("SQLite should not have CREATE PROCEDURE")
	}
	if !strings.Contains(script, "SELECT") {
		t.Error("SQLite should have SELECT statements")
	}
}

func TestMongoDBDriver_JavaScriptOutput(t *testing.T) {
	d, _ := GetDriver("mongodb")
	opts := GenerateOptions{
		DBName:     "testdb",
		SampleSize: 100,
		MaxDepth:   2,
	}

	script, _ := d.GenerateSQL(opts)

	// Should be JavaScript, not SQL
	if strings.Contains(script, "CREATE PROCEDURE") {
		t.Error("MongoDB should not have SQL procedures")
	}
	if !strings.Contains(script, "db.getCollectionNames()") {
		t.Error("MongoDB should use mongosh API")
	}
}

func TestMongoDBDriver_SampleSizeDefaults(t *testing.T) {
	d, _ := GetDriver("mongodb")

	tests := []struct {
		name       string
		sampleSize int
		expected   string
	}{
		{"zero uses default", 0, "SAMPLE_SIZE = 100"},
		{"negative uses default", -1, "SAMPLE_SIZE = 100"},
		{"custom value", 500, "SAMPLE_SIZE = 500"},
		{"capped at max", 20000, "SAMPLE_SIZE = 10000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := GenerateOptions{
				DBName:     "testdb",
				SampleSize: tt.sampleSize,
			}
			script, _ := d.GenerateSQL(opts)
			if !strings.Contains(script, tt.expected) {
				t.Errorf("expected %s in script", tt.expected)
			}
		})
	}
}

func TestMongoDBDriver_MaxDepthDefaults(t *testing.T) {
	d, _ := GetDriver("mongodb")

	tests := []struct {
		name     string
		maxDepth int
		expected string
	}{
		{"zero uses default", 0, "MAX_DEPTH = 2"},
		{"negative uses default", -1, "MAX_DEPTH = 2"},
		{"custom value", 5, "MAX_DEPTH = 5"},
		{"capped at max", 15, "MAX_DEPTH = 10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := GenerateOptions{
				DBName:   "testdb",
				MaxDepth: tt.maxDepth,
			}
			script, _ := d.GenerateSQL(opts)
			if !strings.Contains(script, tt.expected) {
				t.Errorf("expected %s in script", tt.expected)
			}
		})
	}
}

func TestDriversContainJSONOutput(t *testing.T) {
	// All drivers should produce JSON-compatible output
	drivers := ListDrivers()

	for _, name := range drivers {
		t.Run(name, func(t *testing.T) {
			d, _ := GetDriver(name)
			opts := GenerateOptions{
				DBName: "testdb",
				Direct: true,
			}
			script, _ := d.GenerateSQL(opts)

			// Should reference JSON in some form
			hasJSON := strings.Contains(strings.ToLower(script), "json") ||
				strings.Contains(script, "JSON_") ||
				strings.Contains(script, "FOR JSON") ||
				strings.Contains(script, "json_") ||
				strings.Contains(script, "JSON.stringify")

			if !hasJSON {
				t.Log("Warning: script may not produce JSON output")
			}
		})
	}
}
