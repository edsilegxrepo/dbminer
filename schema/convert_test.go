// Test strategy: Unit tests for pure transformation functions.
// Uses in-memory RawSchema fixtures (no file I/O) for fast execution.
// Covers PK/FK detection, type coercion, database type inference, and edge cases.
package schema

import (
	"testing"
)

func TestConvertRawToSchema_MinimalTable(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Version:      "MySQL 8.0",
		Tables: []RawTable{
			{Schema: "public", Table: "users", Type: "TABLE", Comment: "User accounts"},
		},
		Columns: []RawColumn{
			{Schema: "public", Table: "users", Name: "id", Type: "int", OrdinalPos: 1, Nullable: false},
			{Schema: "public", Table: "users", Name: "name", Type: "varchar", OrdinalPos: 2, Nullable: true, CharMaxLen: "255"},
		},
		PKInfo: []RawPKInfo{
			{Schema: "public", Table: "users", Column: "id"},
		},
	}

	schema := ConvertRawToSchema(raw)

	if schema.Name != "testdb" {
		t.Errorf("expected name testdb, got %s", schema.Name)
	}
	if schema.ID != "0" {
		t.Errorf("expected ID 0, got %s", schema.ID)
	}
	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	table := schema.Tables[0]
	if table.Name != "users" {
		t.Errorf("expected table name users, got %s", table.Name)
	}
	if table.Schema != "public" {
		t.Errorf("expected schema public, got %s", table.Schema)
	}
	if table.Comment != "User accounts" {
		t.Errorf("expected comment 'User accounts', got %s", table.Comment)
	}
	if table.IsView {
		t.Error("expected IsView false")
	}
	if len(table.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(table.Fields))
	}

	// Verify PK detection
	if !table.Fields[0].PrimaryKey {
		t.Error("expected id to be primary key")
	}
	if table.Fields[1].PrimaryKey {
		t.Error("expected name to not be primary key")
	}

	// Verify nullable
	if table.Fields[0].Nullable {
		t.Error("expected id to not be nullable")
	}
	if !table.Fields[1].Nullable {
		t.Error("expected name to be nullable")
	}

	// Verify char max length
	if table.Fields[1].CharMaxLen != "255" {
		t.Errorf("expected CharMaxLen 255, got %s", table.Fields[1].CharMaxLen)
	}
}

func TestConvertRawToSchema_MultipleTablesAndViews(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users", Type: "TABLE"},
			{Table: "orders", Type: "TABLE"},
		},
		Views: []RawView{
			{ViewName: "active_users", Schema: "public", ViewDefinition: "SELECT * FROM users WHERE active = 1"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "id", Type: "int", OrdinalPos: 1},
			{Table: "orders", Name: "id", Type: "int", OrdinalPos: 1},
			{Table: "active_users", Name: "id", Type: "int", OrdinalPos: 1},
		},
	}

	schema := ConvertRawToSchema(raw)

	if len(schema.Tables) != 3 {
		t.Fatalf("expected 3 tables (2 tables + 1 view), got %d", len(schema.Tables))
	}

	// Find the view
	var view *Table
	for i := range schema.Tables {
		if schema.Tables[i].Name == "active_users" {
			view = &schema.Tables[i]
			break
		}
	}

	if view == nil {
		t.Fatal("view active_users not found")
	}
	if !view.IsView {
		t.Error("expected active_users to be a view")
	}
}

func TestConvertRawToSchema_ForeignKeys(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "orders"},
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "orders", Name: "id", Type: "int", OrdinalPos: 1},
			{Table: "orders", Name: "user_id", Type: "int", OrdinalPos: 2},
			{Table: "users", Name: "id", Type: "int", OrdinalPos: 1},
		},
		PKInfo: []RawPKInfo{
			{Table: "orders", Column: "id"},
			{Table: "users", Column: "id"},
		},
		FKInfo: []RawFKInfo{
			{
				Table:           "orders",
				Column:          "user_id",
				ForeignKeyName:  "fk_orders_users",
				ReferenceTable:  "users",
				ReferenceColumn: "id",
			},
		},
	}

	schema := ConvertRawToSchema(raw)

	if len(schema.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(schema.Relationships))
	}

	rel := schema.Relationships[0]
	if rel.Name != "fk_orders_users" {
		t.Errorf("expected FK name fk_orders_users, got %s", rel.Name)
	}
	if rel.SourceCardinality != "one" {
		t.Errorf("expected source cardinality 'one', got %s", rel.SourceCardinality)
	}
	if rel.TargetCardinality != "many" {
		t.Errorf("expected target cardinality 'many', got %s", rel.TargetCardinality)
	}
}

func TestConvertRawToSchema_ForeignKeysWithSchema(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Schema: "sales", Table: "orders"},
			{Schema: "hr", Table: "users"},
		},
		Columns: []RawColumn{
			{Schema: "sales", Table: "orders", Name: "id", Type: "int", OrdinalPos: 1},
			{Schema: "sales", Table: "orders", Name: "user_id", Type: "int", OrdinalPos: 2},
			{Schema: "hr", Table: "users", Name: "id", Type: "int", OrdinalPos: 1},
		},
		PKInfo: []RawPKInfo{
			{Schema: "sales", Table: "orders", Column: "id"},
			{Schema: "hr", Table: "users", Column: "id"},
		},
		FKInfo: []RawFKInfo{
			{
				Schema:          "sales",
				Table:           "orders",
				Column:          "user_id",
				ForeignKeyName:  "fk_orders_users",
				ReferenceSchema: "hr",
				ReferenceTable:  "users",
				ReferenceColumn: "id",
			},
		},
	}

	schema := ConvertRawToSchema(raw)

	if len(schema.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(schema.Relationships))
	}
}

func TestConvertRawToSchema_ForeignKeysMissingTable(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "orders"},
		},
		Columns: []RawColumn{
			{Table: "orders", Name: "id", Type: "int", OrdinalPos: 1},
			{Table: "orders", Name: "user_id", Type: "int", OrdinalPos: 2},
		},
		FKInfo: []RawFKInfo{
			{
				Table:           "orders",
				Column:          "user_id",
				ForeignKeyName:  "fk_orders_users",
				ReferenceTable:  "nonexistent",
				ReferenceColumn: "id",
			},
		},
	}

	schema := ConvertRawToSchema(raw)

	// FK should be skipped when target table doesn't exist
	if len(schema.Relationships) != 0 {
		t.Errorf("expected 0 relationships (target missing), got %d", len(schema.Relationships))
	}
}

func TestConvertRawToSchema_Indexes(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "id", Type: "int", OrdinalPos: 1},
			{Table: "users", Name: "email", Type: "varchar", OrdinalPos: 2},
			{Table: "users", Name: "name", Type: "varchar", OrdinalPos: 3},
		},
		Indexes: []RawIndex{
			{Table: "users", Name: "PRIMARY", Column: "id", ColPosition: 1, Unique: true},
			{Table: "users", Name: "idx_email", Column: "email", ColPosition: 1, Unique: true},
			{Table: "users", Name: "idx_name_email", Column: "name", ColPosition: 1, Unique: false},
			{Table: "users", Name: "idx_name_email", Column: "email", ColPosition: 2, Unique: false},
		},
	}

	schema := ConvertRawToSchema(raw)

	if len(schema.Tables) != 1 {
		t.Fatal("expected 1 table")
	}

	table := schema.Tables[0]
	if len(table.Indexes) != 3 {
		t.Fatalf("expected 3 indexes, got %d", len(table.Indexes))
	}

	// Check unique index on email marks field as unique
	var emailField *Field
	for i := range table.Fields {
		if table.Fields[i].Name == "email" {
			emailField = &table.Fields[i]
			break
		}
	}
	if emailField == nil {
		t.Fatal("email field not found")
	}
	if !emailField.Unique {
		t.Error("expected email to be unique")
	}
}

func TestConvertRawToSchema_Triggers(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "id", Type: "int", OrdinalPos: 1},
		},
		Triggers: []RawTrigger{
			{
				Schema:    "public",
				Table:     "users",
				Name:      "trg_users_audit",
				Timing:    "AFTER",
				Event:     "INSERT",
				Statement: "INSERT INTO audit_log VALUES (NEW.id)",
			},
		},
	}

	schema := ConvertRawToSchema(raw)

	if len(schema.Triggers) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(schema.Triggers))
	}

	trig := schema.Triggers[0]
	if trig.Name != "trg_users_audit" {
		t.Errorf("expected trigger name trg_users_audit, got %s", trig.Name)
	}
	if trig.Timing != "AFTER" {
		t.Errorf("expected timing AFTER, got %s", trig.Timing)
	}
	if trig.Event != "INSERT" {
		t.Errorf("expected event INSERT, got %s", trig.Event)
	}
}

func TestConvertRawToSchema_StoredProcedures(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		StoredProcedures: []RawStoredProcedure{
			{
				Schema:     "public",
				Name:       "get_user",
				Type:       "FUNCTION",
				ReturnType: "TABLE",
				Parameters: []RawProcedureParam{
					{Name: "user_id", Type: "int", Mode: "IN", Position: 1},
				},
				Comment: "Gets user by ID",
			},
			{
				Schema: "public",
				Name:   "update_stats",
				Type:   "PROCEDURE",
			},
		},
	}

	schema := ConvertRawToSchema(raw)

	if len(schema.StoredProcedures) != 2 {
		t.Fatalf("expected 2 stored procedures, got %d", len(schema.StoredProcedures))
	}

	fn := schema.StoredProcedures[0]
	if fn.Name != "get_user" {
		t.Errorf("expected name get_user, got %s", fn.Name)
	}
	if fn.Type != "FUNCTION" {
		t.Errorf("expected type FUNCTION, got %s", fn.Type)
	}
	if fn.ReturnType != "TABLE" {
		t.Errorf("expected return type TABLE, got %s", fn.ReturnType)
	}
	if len(fn.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].Name != "user_id" {
		t.Errorf("expected param name user_id, got %s", fn.Parameters[0].Name)
	}
}

func TestConvertRawToSchema_GeneratedColumns(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "products"},
		},
		Columns: []RawColumn{
			{Table: "products", Name: "price", Type: "decimal", OrdinalPos: 1},
			{Table: "products", Name: "qty", Type: "int", OrdinalPos: 2},
			{Table: "products", Name: "total", Type: "decimal", OrdinalPos: 3, GeneratedExpr: "price * qty", GeneratedType: "STORED"},
		},
	}

	schema := ConvertRawToSchema(raw)

	table := schema.Tables[0]
	totalField := table.Fields[2]

	if totalField.GeneratedExpr != "price * qty" {
		t.Errorf("expected generated expr 'price * qty', got %s", totalField.GeneratedExpr)
	}
	if totalField.GeneratedType != "STORED" {
		t.Errorf("expected generated type STORED, got %s", totalField.GeneratedType)
	}
}

func TestConvertRawToSchema_ColumnOrdering(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "third", Type: "int", OrdinalPos: 3},
			{Table: "users", Name: "first", Type: "int", OrdinalPos: 1},
			{Table: "users", Name: "second", Type: "int", OrdinalPos: 2},
		},
	}

	schema := ConvertRawToSchema(raw)

	fields := schema.Tables[0].Fields
	if fields[0].Name != "first" {
		t.Errorf("expected first field to be 'first', got %s", fields[0].Name)
	}
	if fields[1].Name != "second" {
		t.Errorf("expected second field to be 'second', got %s", fields[1].Name)
	}
	if fields[2].Name != "third" {
		t.Errorf("expected third field to be 'third', got %s", fields[2].Name)
	}
}

func TestConvertRawToSchema_IndexColumnOrdering(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "a", Type: "int", OrdinalPos: 1},
			{Table: "users", Name: "b", Type: "int", OrdinalPos: 2},
			{Table: "users", Name: "c", Type: "int", OrdinalPos: 3},
		},
		Indexes: []RawIndex{
			{Table: "users", Name: "idx_composite", Column: "c", ColPosition: 3},
			{Table: "users", Name: "idx_composite", Column: "a", ColPosition: 1},
			{Table: "users", Name: "idx_composite", Column: "b", ColPosition: 2},
		},
	}

	schema := ConvertRawToSchema(raw)

	idx := schema.Tables[0].Indexes[0]
	if len(idx.FieldIDs) != 3 {
		t.Fatalf("expected 3 fields in index, got %d", len(idx.FieldIDs))
	}

	// Verify ordering by checking field names via ID lookup
	fieldMap := make(map[string]string)
	for _, f := range schema.Tables[0].Fields {
		fieldMap[f.ID] = f.Name
	}

	if fieldMap[idx.FieldIDs[0]] != "a" {
		t.Errorf("expected first index column to be 'a', got %s", fieldMap[idx.FieldIDs[0]])
	}
	if fieldMap[idx.FieldIDs[1]] != "b" {
		t.Errorf("expected second index column to be 'b', got %s", fieldMap[idx.FieldIDs[1]])
	}
	if fieldMap[idx.FieldIDs[2]] != "c" {
		t.Errorf("expected third index column to be 'c', got %s", fieldMap[idx.FieldIDs[2]])
	}
}

func TestDetectDatabaseType(t *testing.T) {
	tests := []struct {
		name     string
		raw      *RawSchema
		expected string
	}{
		{
			name:     "PostgreSQL from version lowercase",
			raw:      &RawSchema{Version: "postgresql 15.2"},
			expected: "postgresql",
		},
		{
			name:     "PostgreSQL from version mixed case",
			raw:      &RawSchema{Version: "PostgreSQL 15.2"},
			expected: "postgresql",
		},
		{
			name:     "Postgres short form",
			raw:      &RawSchema{Version: "Postgres 14"},
			expected: "postgresql",
		},
		{
			name:     "MariaDB from version",
			raw:      &RawSchema{Version: "10.6.12-MariaDB"},
			expected: "mariadb",
		},
		{
			name:     "SQL Server from version",
			raw:      &RawSchema{Version: "Microsoft SQL Server 2019"},
			expected: "sqlserver",
		},
		{
			name:     "SQL Server short",
			raw:      &RawSchema{Version: "SQL Server 2022"},
			expected: "sqlserver",
		},
		{
			name:     "Oracle from version",
			raw:      &RawSchema{Version: "Oracle Database 19c"},
			expected: "oracle",
		},
		{
			name:     "Oracle from dbname",
			raw:      &RawSchema{DatabaseName: "ORACLE_DB"},
			expected: "oracle",
		},
		{
			name:     "SQLite from dbname",
			raw:      &RawSchema{DatabaseName: "sqlite"},
			expected: "sqlite",
		},
		{
			name:     "SQLite from version",
			raw:      &RawSchema{Version: "SQLite 3.39"},
			expected: "sqlite",
		},
		{
			name: "MongoDB from table type COLLECTION",
			raw: &RawSchema{
				Tables: []RawTable{{Type: "COLLECTION"}},
			},
			expected: "mongodb",
		},
		{
			name: "MongoDB from table type lowercase",
			raw: &RawSchema{
				Tables: []RawTable{{Type: "collection"}},
			},
			expected: "mongodb",
		},
		{
			name: "MongoDB from engine",
			raw: &RawSchema{
				Tables: []RawTable{{Engine: "MongoDB"}},
			},
			expected: "mongodb",
		},
		{
			name: "PostgreSQL from custom types",
			raw: &RawSchema{
				CustomTypes: []RawCustomType{{Type: "mood", Kind: "enum"}},
			},
			expected: "postgresql",
		},
		{
			name:     "MySQL default",
			raw:      &RawSchema{Version: "8.0.32"},
			expected: "mysql",
		},
		{
			name:     "MySQL explicit",
			raw:      &RawSchema{Version: "MySQL 8.0.32"},
			expected: "mysql",
		},
		{
			name:     "Empty defaults to MySQL",
			raw:      &RawSchema{},
			expected: "mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectDatabaseType(tt.raw)
			if got != tt.expected {
				t.Errorf("detectDatabaseType() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"string 1", "1", true},
		{"string false", "false", false},
		{"string 0", "0", false},
		{"string empty", "", false},
		{"string yes", "yes", true},   // MySQL nullable format
		{"string YES", "YES", true},   // MySQL nullable format uppercase
		{"string y", "y", true},       // Short form
		{"float64 1", float64(1), true},
		{"float64 0", float64(0), false},
		{"float64 negative", float64(-1), true},
		{"float64 0.5", float64(0.5), true},
		{"int 1", 1, true},
		{"int 0", 0, false},
		{"int negative", -1, true},
		{"nil", nil, false},
		{"struct", struct{}{}, false},
		{"slice", []int{1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toBool(tt.input)
			if got != tt.expected {
				t.Errorf("toBool(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConvertRawToSchema_EmptyInput(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "emptydb",
	}

	schema := ConvertRawToSchema(raw)

	if schema.Name != "emptydb" {
		t.Errorf("expected name emptydb, got %s", schema.Name)
	}
	if len(schema.Tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(schema.Tables))
	}
	if len(schema.Relationships) != 0 {
		t.Errorf("expected 0 relationships, got %d", len(schema.Relationships))
	}
}

func TestConvertRawToSchema_DefaultPositions(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "id", Type: "int", OrdinalPos: 1},
		},
	}

	schema := ConvertRawToSchema(raw)

	table := schema.Tables[0]
	if table.X != DefaultX {
		t.Errorf("expected X = %d, got %d", DefaultX, table.X)
	}
	if table.Y != DefaultY {
		t.Errorf("expected Y = %d, got %d", DefaultY, table.Y)
	}
}

func TestConvertRawToSchema_FieldCollation(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "name", Type: "varchar", OrdinalPos: 1, Collation: "utf8mb4_unicode_ci"},
		},
	}

	schema := ConvertRawToSchema(raw)

	field := schema.Tables[0].Fields[0]
	if field.Collation != "utf8mb4_unicode_ci" {
		t.Errorf("expected collation utf8mb4_unicode_ci, got %s", field.Collation)
	}
}

func TestConvertRawToSchema_FieldDefault(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "created_at", Type: "timestamp", OrdinalPos: 1, Default: "CURRENT_TIMESTAMP"},
			{Table: "users", Name: "status", Type: "varchar", OrdinalPos: 2, Default: "'active'"},
		},
	}

	schema := ConvertRawToSchema(raw)

	if schema.Tables[0].Fields[0].Default != "CURRENT_TIMESTAMP" {
		t.Errorf("expected default CURRENT_TIMESTAMP, got %s", schema.Tables[0].Fields[0].Default)
	}
	if schema.Tables[0].Fields[1].Default != "'active'" {
		t.Errorf("expected default 'active', got %s", schema.Tables[0].Fields[1].Default)
	}
}

func TestConvertRawToSchema_FieldComment(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "id", Type: "int", OrdinalPos: 1, Comment: "Primary key"},
		},
	}

	schema := ConvertRawToSchema(raw)

	if schema.Tables[0].Fields[0].Comment != "Primary key" {
		t.Errorf("expected comment 'Primary key', got %s", schema.Tables[0].Fields[0].Comment)
	}
}

func TestConvertRawToSchema_ViewColumns(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Views: []RawView{
			{ViewName: "user_summary", Schema: "public"},
		},
		Columns: []RawColumn{
			{Table: "user_summary", Name: "user_id", Type: "int", OrdinalPos: 1, Nullable: true},
			{Table: "user_summary", Name: "total_orders", Type: "int", OrdinalPos: 2, Nullable: true},
		},
	}

	schema := ConvertRawToSchema(raw)

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table (view), got %d", len(schema.Tables))
	}

	view := schema.Tables[0]
	if !view.IsView {
		t.Error("expected IsView to be true")
	}
	if len(view.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(view.Fields))
	}

	// Views shouldn't have PKs
	for _, f := range view.Fields {
		if f.PrimaryKey {
			t.Errorf("view field %s should not be PK", f.Name)
		}
	}
}

func TestConvertRawToSchema_UniqueConstraintFromMultiColumnIndex(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "email", Type: "varchar", OrdinalPos: 1},
			{Table: "users", Name: "tenant_id", Type: "int", OrdinalPos: 2},
		},
		Indexes: []RawIndex{
			{Table: "users", Name: "idx_email_tenant", Column: "email", ColPosition: 1, Unique: true},
			{Table: "users", Name: "idx_email_tenant", Column: "tenant_id", ColPosition: 2, Unique: true},
		},
	}

	schema := ConvertRawToSchema(raw)

	// Multi-column unique index should NOT mark individual columns as unique
	for _, f := range schema.Tables[0].Fields {
		if f.Unique {
			t.Errorf("field %s should not be marked unique (multi-column index)", f.Name)
		}
	}
}

func TestConvertRawToSchema_SingleColumnUniqueIndex(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "email", Type: "varchar", OrdinalPos: 1},
		},
		Indexes: []RawIndex{
			{Table: "users", Name: "idx_email", Column: "email", ColPosition: 1, Unique: true},
		},
	}

	schema := ConvertRawToSchema(raw)

	if !schema.Tables[0].Fields[0].Unique {
		t.Error("expected email to be unique (single-column unique index)")
	}
}

func TestConvertRawToSchema_PKImpliesUnique(t *testing.T) {
	raw := &RawSchema{
		DatabaseName: "testdb",
		Tables: []RawTable{
			{Table: "users"},
		},
		Columns: []RawColumn{
			{Table: "users", Name: "id", Type: "int", OrdinalPos: 1},
		},
		PKInfo: []RawPKInfo{
			{Table: "users", Column: "id"},
		},
	}

	schema := ConvertRawToSchema(raw)

	field := schema.Tables[0].Fields[0]
	if !field.PrimaryKey {
		t.Error("expected id to be PK")
	}
	if !field.Unique {
		t.Error("expected id to be unique (PK implies unique)")
	}
}
