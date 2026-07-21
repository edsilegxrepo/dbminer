// Test strategy: Output structure tests for markdown generation using t.TempDir().
// Validates README content (header, TOC, ERD, relationships), table doc structure
// (columns, indexes, triggers, FKs), and utility functions (sanitize, escape).
package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"criticalsys.net/dbminer/schema"
)

func newTestSchema() *schema.Schema {
	return &schema.Schema{
		ID:           "1",
		Name:         "testdb",
		DatabaseType: "mysql",
		Tables: []schema.Table{
			{
				ID:      "1",
				Name:    "users",
				Schema:  "public",
				Comment: "User accounts table",
				Fields: []schema.Field{
					{ID: "1", Name: "id", Type: schema.FieldType{Name: "int"}, PrimaryKey: true, Nullable: false},
					{ID: "2", Name: "email", Type: schema.FieldType{Name: "varchar"}, CharMaxLen: "255", Nullable: false, Unique: true},
					{ID: "3", Name: "name", Type: schema.FieldType{Name: "varchar"}, CharMaxLen: "100", Nullable: true},
					{ID: "4", Name: "created_at", Type: schema.FieldType{Name: "timestamp"}, Default: "CURRENT_TIMESTAMP"},
				},
				Indexes: []schema.Index{
					{ID: "1", Name: "PRIMARY", Unique: true, FieldIDs: []string{"1"}},
					{ID: "2", Name: "idx_email", Unique: true, FieldIDs: []string{"2"}},
				},
			},
			{
				ID:     "2",
				Name:   "orders",
				Schema: "public",
				Fields: []schema.Field{
					{ID: "5", Name: "id", Type: schema.FieldType{Name: "int"}, PrimaryKey: true},
					{ID: "6", Name: "user_id", Type: schema.FieldType{Name: "int"}},
					{ID: "7", Name: "total", Type: schema.FieldType{Name: "decimal"}},
				},
			},
		},
		Relationships: []schema.Relationship{
			{
				ID:                "1",
				Name:              "fk_orders_users",
				SourceTableID:     "2",
				TargetTableID:     "1",
				SourceFieldID:     "6",
				TargetFieldID:     "1",
				SourceCardinality: "one",
				TargetCardinality: "many",
			},
		},
		Triggers: []schema.Trigger{
			{ID: "1", Name: "trg_users_audit", Schema: "public", Table: "users", Timing: "AFTER", Event: "INSERT"},
		},
		StoredProcedures: []schema.StoredProcedure{
			{
				ID:         "1",
				Name:       "get_user",
				Schema:     "public",
				Type:       "FUNCTION",
				ReturnType: "TABLE",
				Parameters: []schema.ProcedureParam{
					{Name: "user_id", Type: "int", Mode: "IN"},
				},
			},
		},
	}
}

func TestGenerate_BasicStructure(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	err := Generate(s, dir, Options{})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Check README exists
	readme := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		t.Error("README.md not created")
	}

	// Check tables directory
	tablesDir := filepath.Join(dir, "tables")
	if _, err := os.Stat(tablesDir); os.IsNotExist(err) {
		t.Error("tables/ directory not created")
	}

	// Check table files
	expectedTables := []string{"users.md", "orders.md"}
	for _, f := range expectedTables {
		path := filepath.Join(tablesDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("%s not created", f)
		}
	}
}

func TestGenerate_README_Header(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	checks := []string{
		"# testdb Database Schema",
		"**Database Type:** mysql",
		"**Tables:** 2",
		"**Relationships:** 1",
		"**Triggers:** 1",
		"**Stored Procedures/Functions:** 1",
	}

	for _, check := range checks {
		if !strings.Contains(readme, check) {
			t.Errorf("README missing: %s", check)
		}
	}
}

func TestGenerate_README_TableOfContents(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	if !strings.Contains(readme, "## Table of Contents") {
		t.Error("README missing Table of Contents")
	}

	// Should have links to table docs
	if !strings.Contains(readme, "[users](tables/users.md)") {
		t.Error("README missing link to users table")
	}
	if !strings.Contains(readme, "[orders](tables/orders.md)") {
		t.Error("README missing link to orders table")
	}
}

func TestGenerate_README_MermaidERD(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	if !strings.Contains(readme, "```mermaid") {
		t.Error("README missing Mermaid diagram")
	}
	if !strings.Contains(readme, "flowchart") {
		t.Error("README missing flowchart declaration")
	}
}

func TestGenerate_README_RelationshipsTable(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	if !strings.Contains(readme, "## Relationships Overview") {
		t.Error("README missing Relationships Overview")
	}
	// Check for relationship table structure (source -> target)
	if !strings.Contains(readme, "orders") || !strings.Contains(readme, "users") {
		t.Error("README missing relationship table names")
	}
}

func TestGenerate_README_TriggersSection(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	if !strings.Contains(readme, "## Triggers") {
		t.Error("README missing Triggers section")
	}
	if !strings.Contains(readme, "trg_users_audit") {
		t.Error("README missing trigger name")
	}
	if !strings.Contains(readme, "AFTER") || !strings.Contains(readme, "INSERT") {
		t.Error("README missing trigger timing/event")
	}
}

func TestGenerate_README_StoredProceduresSection(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	if !strings.Contains(readme, "## Stored Procedures & Functions") {
		t.Error("README missing Stored Procedures section")
	}
	if !strings.Contains(readme, "get_user") {
		t.Error("README missing procedure name")
	}
	if !strings.Contains(readme, "FUNCTION") {
		t.Error("README missing procedure type")
	}
}

func TestGenerate_TableDoc_Header(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	checks := []string{
		"# users",
		"**Type:** Table",
		"**Schema:** public",
		"**Description:** User accounts table",
		"**Columns:** 4",
	}

	for _, check := range checks {
		if !strings.Contains(doc, check) {
			t.Errorf("users.md missing: %s", check)
		}
	}
}

func TestGenerate_TableDoc_ColumnsTable(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	if !strings.Contains(doc, "## Columns") {
		t.Error("missing Columns section")
	}

	// Check header
	if !strings.Contains(doc, "| # | Name | Type | Nullable | Default | Key |") {
		t.Error("missing columns table header")
	}

	// Check PK indicator
	if !strings.Contains(doc, "PK") {
		t.Error("missing PK indicator")
	}

	// Check unique indicator
	if !strings.Contains(doc, "UQ") {
		t.Error("missing UQ indicator")
	}

	// Check type with length
	if !strings.Contains(doc, "varchar(255)") {
		t.Error("missing type with length")
	}
}

func TestGenerate_TableDoc_Indexes(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	if !strings.Contains(doc, "## Indexes") {
		t.Error("missing Indexes section")
	}
	if !strings.Contains(doc, "idx_email") {
		t.Error("missing index name")
	}
}

func TestGenerate_TableDoc_Triggers(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	if !strings.Contains(doc, "## Triggers") {
		t.Error("missing Triggers section in table doc")
	}
	if !strings.Contains(doc, "trg_users_audit") {
		t.Error("missing trigger in table doc")
	}
}

func TestGenerate_TableDoc_OutboundRelationships(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "orders.md"))
	doc := string(content)

	if !strings.Contains(doc, "## Relationships") {
		t.Error("missing Relationships section")
	}
	if !strings.Contains(doc, "References (this table") {
		t.Error("missing outbound references section")
	}
	if !strings.Contains(doc, "user_id") {
		t.Error("missing FK column")
	}
	if !strings.Contains(doc, "[users]") {
		t.Error("missing link to referenced table")
	}
}

func TestGenerate_TableDoc_InboundRelationships(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	if !strings.Contains(doc, "Referenced By") {
		t.Error("missing inbound references section")
	}
	if !strings.Contains(doc, "[orders]") {
		t.Error("missing link to referencing table")
	}
}

func TestGenerate_TableDoc_MermaidERD(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	// orders has relationships, should have ERD
	content, _ := os.ReadFile(filepath.Join(dir, "tables", "orders.md"))
	doc := string(content)

	if !strings.Contains(doc, "```mermaid") {
		t.Error("missing Mermaid ERD in orders.md")
	}
	if !strings.Contains(doc, "## Entity Relationship Diagram") {
		t.Error("missing ERD section header")
	}
}

func TestGenerate_TableDoc_BackLink(t *testing.T) {
	s := newTestSchema()
	dir := t.TempDir()

	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	if !strings.Contains(doc, "[← Back to Schema](../README.md)") {
		t.Error("missing back link to README")
	}
}

func TestGenerate_ViewDoc(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		Tables: []schema.Table{
			{
				ID:     "1",
				Name:   "active_users",
				Schema: "public",
				IsView: true,
				Fields: []schema.Field{
					{ID: "1", Name: "id", Type: schema.FieldType{Name: "int"}},
				},
			},
		},
	}

	dir := t.TempDir()
	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "active_users.md"))
	doc := string(content)

	if !strings.Contains(doc, "**Type:** View") {
		t.Error("view not marked as View type")
	}
}

func TestGenerate_README_ViewIcon(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		Tables: []schema.Table{
			{ID: "1", Name: "users", IsView: false},
			{ID: "2", Name: "active_users", IsView: true},
		},
	}

	dir := t.TempDir()
	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	// Tables get clipboard icon, views get eye icon
	if !strings.Contains(readme, "📋") {
		t.Error("missing table icon")
	}
	if !strings.Contains(readme, "👁️") {
		t.Error("missing view icon")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users", "users"},
		{"user-data", "user-data"},
		{"user_data", "user_data"},
		{"user.data", "user.data"},
		{"user|pipe", "user_pipe"},
		{"user<angle>", "user_angle_"},
		{"user:colon", "user_colon"},
		{"user*star", "user_star"},
		{"user?question", "user_question"},
		{"user\"quote", "user_quote"},
		{"", "_invalid_"},
		{".", "_invalid_"},
		{"..", "_invalid_"},
		{strings.Repeat("a", 250), strings.Repeat("a", 200)}, // Truncated
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeMermaidID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users", "users"},
		{"user_data", "user_data"},
		{"user-data", "user_data"},
		{"user.data", "user_data"},
		{"User123", "User123"},
		{"user table", "user_table"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeMermaidID(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeMermaidID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEscapeMarkdownCell(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"with|pipe", "with\\|pipe"},
		{"with\nnewline", "with newline"},
		{"multiple|pipes|here", "multiple\\|pipes\\|here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeMarkdownCell(tt.input)
			if got != tt.expected {
				t.Errorf("escapeMarkdownCell(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEscapeMermaidLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"with|pipe", "with_pipe"},
		{"with[bracket]", "with_bracket_"},
		{"with\"quote", "with_quote"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeMermaidLabel(tt.input)
			if got != tt.expected {
				t.Errorf("escapeMermaidLabel(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGenerate_EmptySchema(t *testing.T) {
	s := &schema.Schema{
		Name:         "emptydb",
		DatabaseType: "mysql",
	}

	dir := t.TempDir()
	err := Generate(s, dir, Options{})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// README should still be created
	if _, err := os.Stat(filepath.Join(dir, "README.md")); os.IsNotExist(err) {
		t.Error("README.md not created for empty schema")
	}
}

func TestGenerate_GeneratedColumns(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		Tables: []schema.Table{
			{
				ID:   "1",
				Name: "products",
				Fields: []schema.Field{
					{ID: "1", Name: "price", Type: schema.FieldType{Name: "decimal"}},
					{ID: "2", Name: "qty", Type: schema.FieldType{Name: "int"}},
					{ID: "3", Name: "total", Type: schema.FieldType{Name: "decimal"}, GeneratedExpr: "price * qty", GeneratedType: "STORED"},
				},
			},
		},
	}

	dir := t.TempDir()
	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "products.md"))
	doc := string(content)

	if !strings.Contains(doc, "STORED") {
		t.Error("missing generated column type")
	}
	if !strings.Contains(doc, "price * qty") || !strings.Contains(doc, "`price * qty`") {
		t.Error("missing generated column expression")
	}
}

func TestGenerate_DefaultValues(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		Tables: []schema.Table{
			{
				ID:   "1",
				Name: "users",
				Fields: []schema.Field{
					{ID: "1", Name: "status", Type: schema.FieldType{Name: "varchar"}, Default: "'active'"},
					{ID: "2", Name: "created", Type: schema.FieldType{Name: "timestamp"}, Default: "CURRENT_TIMESTAMP"},
				},
			},
		},
	}

	dir := t.TempDir()
	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	if !strings.Contains(doc, "'active'") {
		t.Error("missing string default value")
	}
	if !strings.Contains(doc, "CURRENT_TIMESTAMP") {
		t.Error("missing timestamp default value")
	}
}

func TestGenerate_NullableIndicator(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		Tables: []schema.Table{
			{
				ID:   "1",
				Name: "users",
				Fields: []schema.Field{
					{ID: "1", Name: "id", Type: schema.FieldType{Name: "int"}, Nullable: false},
					{ID: "2", Name: "name", Type: schema.FieldType{Name: "varchar"}, Nullable: true},
				},
			},
		},
	}

	dir := t.TempDir()
	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	// Should have YES and NO for nullable
	if !strings.Contains(doc, "| NO |") {
		t.Error("missing NO for non-nullable")
	}
	if !strings.Contains(doc, "| YES |") {
		t.Error("missing YES for nullable")
	}
}

func TestGenerate_MultipleIndexColumns(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		Tables: []schema.Table{
			{
				ID:   "1",
				Name: "users",
				Fields: []schema.Field{
					{ID: "1", Name: "first_name", Type: schema.FieldType{Name: "varchar"}},
					{ID: "2", Name: "last_name", Type: schema.FieldType{Name: "varchar"}},
				},
				Indexes: []schema.Index{
					{ID: "1", Name: "idx_name", FieldIDs: []string{"1", "2"}},
				},
			},
		},
	}

	dir := t.TempDir()
	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	doc := string(content)

	if !strings.Contains(doc, "first_name, last_name") || !strings.Contains(doc, "first_name") {
		t.Error("missing composite index columns")
	}
}

func TestGenerate_StoredProcParameters(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		StoredProcedures: []schema.StoredProcedure{
			{
				ID:   "1",
				Name: "update_user",
				Type: "PROCEDURE",
				Parameters: []schema.ProcedureParam{
					{Name: "user_id", Type: "int", Mode: "IN"},
					{Name: "new_name", Type: "varchar", Mode: "IN"},
					{Name: "result", Type: "int", Mode: "OUT"},
				},
			},
		},
	}

	dir := t.TempDir()
	Generate(s, dir, Options{})

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	if !strings.Contains(readme, "IN user_id int") {
		t.Error("missing IN parameter")
	}
	if !strings.Contains(readme, "OUT result int") {
		t.Error("missing OUT parameter")
	}
}

func TestGenerate_InvalidOutputDir(t *testing.T) {
	s := &schema.Schema{Name: "test"}

	err := Generate(s, "/nonexistent/deep/path/that/cannot/be/created\x00invalid", Options{})
	if err == nil {
		t.Error("expected error for invalid output directory")
	}
}

func TestDetectDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		tables   []schema.Table
		expected string
	}{
		{
			name:     "underscore majority",
			tables:   []schema.Table{{Name: "dpa_users"}, {Name: "dpa_orders"}, {Name: "other"}},
			expected: "_",
		},
		{
			name:     "dot majority",
			tables:   []schema.Table{{Name: "sales.orders"}, {Name: "sales.items"}, {Name: "other"}},
			expected: ".",
		},
		{
			name:     "hyphen majority",
			tables:   []schema.Table{{Name: "user-accounts"}, {Name: "user-roles"}, {Name: "other"}},
			expected: "-",
		},
		{
			name:     "no clear majority",
			tables:   []schema.Table{{Name: "users"}, {Name: "orders"}, {Name: "items"}},
			expected: "",
		},
		{
			name:     "empty tables",
			tables:   []schema.Table{},
			expected: "",
		},
		{
			name:     "dot takes precedence over underscore",
			tables:   []schema.Table{{Name: "sales.user_orders"}, {Name: "sales.user_items"}, {Name: "other"}},
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectDelimiter(tt.tables)
			if got != tt.expected {
				t.Errorf("detectDelimiter() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetTableGroup(t *testing.T) {
	tests := []struct {
		name      string
		table     *schema.Table
		delimiter string
		groupBy   string
		expected  string
	}{
		{
			name:      "auto with underscore",
			table:     &schema.Table{Name: "dpa_web_user", Schema: "mydb"},
			delimiter: "_",
			groupBy:   "auto",
			expected:  "dpa_web",
		},
		{
			name:      "prefix mode",
			table:     &schema.Table{Name: "dpa_web_user", Schema: "mydb"},
			delimiter: ".",
			groupBy:   "prefix",
			expected:  "dpa_web",
		},
		{
			name:      "schema mode",
			table:     &schema.Table{Name: "users", Schema: "sales"},
			delimiter: "_",
			groupBy:   "schema",
			expected:  "sales",
		},
		{
			name:      "schema mode with empty schema",
			table:     &schema.Table{Name: "users", Schema: ""},
			delimiter: "_",
			groupBy:   "schema",
			expected:  "default",
		},
		{
			name:      "none mode",
			table:     &schema.Table{Name: "dpa_web_user", Schema: "mydb"},
			delimiter: "_",
			groupBy:   "none",
			expected:  "Tables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTableGroup(tt.table, tt.delimiter, tt.groupBy)
			if got != tt.expected {
				t.Errorf("getTableGroup() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGenerate_GroupByNone(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		Tables: []schema.Table{
			{ID: "1", Name: "dpa_users"},
			{ID: "2", Name: "dpa_orders"},
			{ID: "3", Name: "other_table"},
		},
	}

	dir := t.TempDir()
	err := Generate(s, dir, Options{GroupBy: "none"})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	// Should have single "Tables" group
	if !strings.Contains(readme, "### Tables") {
		t.Error("README missing Tables group header")
	}
	// Should NOT have prefix-based groups
	if strings.Contains(readme, "### dpa_") {
		t.Error("README should not have prefix groups with GroupBy=none")
	}
}

func TestGenerate_GroupBySchema(t *testing.T) {
	s := &schema.Schema{
		Name: "testdb",
		Tables: []schema.Table{
			{ID: "1", Name: "users", Schema: "sales"},
			{ID: "2", Name: "orders", Schema: "sales"},
			{ID: "3", Name: "products", Schema: "inventory"},
		},
	}

	dir := t.TempDir()
	err := Generate(s, dir, Options{GroupBy: "schema"})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	// Should have schema-based groups
	if !strings.Contains(readme, "### sales") {
		t.Error("README missing sales schema group")
	}
	if !strings.Contains(readme, "### inventory") {
		t.Error("README missing inventory schema group")
	}
}

func TestGenerate_UnicodeContent(t *testing.T) {
	s := &schema.Schema{
		Name:         "日本語DB",
		DatabaseType: "mysql",
		Tables: []schema.Table{
			{
				ID:      "1",
				Name:    "用户",
				Comment: "用户信息表 🎉",
				Fields: []schema.Field{
					{ID: "1", Name: "姓名", Type: schema.FieldType{Name: "varchar"}, Comment: "Full name"},
				},
			},
		},
	}

	dir := t.TempDir()
	err := Generate(s, dir, Options{})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(content)

	if !strings.Contains(readme, "日本語DB") {
		t.Error("missing Unicode database name")
	}
}

func TestGenerate_LargeSchema(t *testing.T) {
	s := &schema.Schema{
		Name: "largedb",
	}

	// Create 50 tables with 20 columns each
	for i := 0; i < 50; i++ {
		table := schema.Table{
			ID:   string(rune('0' + i)),
			Name: "table_" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
		}
		for j := 0; j < 20; j++ {
			table.Fields = append(table.Fields, schema.Field{
				ID:   string(rune(i*20 + j)),
				Name: "col_" + string(rune('a'+j%26)),
				Type: schema.FieldType{Name: "varchar"},
			})
		}
		s.Tables = append(s.Tables, table)
	}

	dir := t.TempDir()
	err := Generate(s, dir, Options{})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify all table files created
	files, _ := os.ReadDir(filepath.Join(dir, "tables"))
	if len(files) != 50 {
		t.Errorf("expected 50 table files, got %d", len(files))
	}
}
