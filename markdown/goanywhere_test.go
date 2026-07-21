// Test strategy: Markdown output validation with realistic GoAnywhere data.
// Validates README structure (header, TOC, ERD, relationships table) and per-table docs
// (columns, indexes, FK references, back links). Tests FK chain navigability and domain grouping.
package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"criticalsys.net/dbminer/schema"
)

// TestGoAnywhereMarkdown_Generate tests markdown generation with realistic GoAnywhere data
func TestGoAnywhereMarkdown_Generate(t *testing.T) {
	raw, err := schema.LoadRaw("../testdata/goanywhere_sample.json")
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	s := schema.ConvertRawToSchema(raw)
	dir := t.TempDir()

	err = Generate(s, dir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify README exists and has correct structure
	readmeData, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("README.md not created: %v", err)
	}
	readme := string(readmeData)

	// Check header
	if !strings.Contains(readme, "# goanydb Database Schema") {
		t.Error("README missing database name header")
	}
	if !strings.Contains(readme, "**Database Type:** mysql") {
		t.Error("README missing database type")
	}
	if !strings.Contains(readme, "**Tables:** 10") {
		t.Error("README missing correct table count")
	}
	if !strings.Contains(readme, "**Relationships:** 7") {
		t.Error("README missing correct relationship count")
	}

	// Check Mermaid ERD present
	if !strings.Contains(readme, "```mermaid") {
		t.Error("README missing Mermaid diagram")
	}

	// Check table of contents has all tables
	expectedTables := []string{
		"dpa_addr_book", "dpa_web_user", "dpa_project",
		"dpa_job", "dpa_job_log", "dpa_trading_partner",
	}
	for _, tbl := range expectedTables {
		if !strings.Contains(readme, tbl) {
			t.Errorf("README missing table: %s", tbl)
		}
	}

	// Check relationships table
	if !strings.Contains(readme, "## Relationships Overview") {
		t.Error("README missing Relationships Overview section")
	}
}

func TestGoAnywhereMarkdown_TableDocs(t *testing.T) {
	raw, _ := schema.LoadRaw("../testdata/goanywhere_sample.json")
	s := schema.ConvertRawToSchema(raw)
	dir := t.TempDir()

	Generate(s, dir)

	// Verify all table docs exist
	tablesDir := filepath.Join(dir, "tables")
	files, err := os.ReadDir(tablesDir)
	if err != nil {
		t.Fatalf("tables directory not created: %v", err)
	}

	if len(files) != 10 {
		t.Errorf("expected 10 table files, got %d", len(files))
	}

	// Check specific table doc content
	t.Run("dpa_web_user", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(tablesDir, "dpa_web_user.md"))
		if err != nil {
			t.Fatalf("dpa_web_user.md not created: %v", err)
		}
		doc := string(data)

		// Header
		if !strings.Contains(doc, "# dpa_web_user") {
			t.Error("missing table header")
		}
		if !strings.Contains(doc, "**Type:** Table") {
			t.Error("missing type")
		}
		if !strings.Contains(doc, "**Schema:** goanydb") {
			t.Error("missing schema")
		}

		// Columns section
		if !strings.Contains(doc, "## Columns") {
			t.Error("missing Columns section")
		}
		if !strings.Contains(doc, "`user_id`") {
			t.Error("missing user_id column")
		}
		if !strings.Contains(doc, "`username`") {
			t.Error("missing username column")
		}
		if !strings.Contains(doc, "PK") {
			t.Error("missing PK indicator for user_id")
		}

		// Indexes section
		if !strings.Contains(doc, "## Indexes") {
			t.Error("missing Indexes section")
		}
		if !strings.Contains(doc, "idx_username") {
			t.Error("missing idx_username index")
		}

		// Relationships - dpa_web_user is referenced by dpa_project
		if !strings.Contains(doc, "Referenced By") {
			t.Error("missing Referenced By section")
		}
		if !strings.Contains(doc, "dpa_project") {
			t.Error("missing reference from dpa_project")
		}

		// Back link
		if !strings.Contains(doc, "[← Back to Schema](../README.md)") {
			t.Error("missing back link")
		}
	})

	t.Run("dpa_job_log", func(t *testing.T) {
		data, _ := os.ReadFile(filepath.Join(tablesDir, "dpa_job_log.md"))
		doc := string(data)

		// Should have outbound FK to dpa_job
		if !strings.Contains(doc, "References (this table") {
			t.Error("missing outbound references section")
		}
		if !strings.Contains(doc, "dpa_job") {
			t.Error("missing FK reference to dpa_job")
		}

		// Should have Mermaid ERD (has relationships)
		if !strings.Contains(doc, "```mermaid") {
			t.Error("missing Mermaid ERD")
		}
	})

	t.Run("dpa_trading_partner", func(t *testing.T) {
		data, _ := os.ReadFile(filepath.Join(tablesDir, "dpa_trading_partner.md"))
		doc := string(data)

		// No relationships - should NOT have Mermaid ERD
		if strings.Contains(doc, "## Entity Relationship Diagram") {
			t.Error("isolated table should not have ERD section")
		}
	})
}

func TestGoAnywhereMarkdown_FKChainVisualization(t *testing.T) {
	raw, _ := schema.LoadRaw("../testdata/goanywhere_sample.json")
	s := schema.ConvertRawToSchema(raw)
	dir := t.TempDir()

	Generate(s, dir)

	// Check that FK chain job_log -> job -> project -> web_user is navigable
	tablesDir := filepath.Join(dir, "tables")

	// job_log references job
	jobLogDoc, _ := os.ReadFile(filepath.Join(tablesDir, "dpa_job_log.md"))
	if !strings.Contains(string(jobLogDoc), "[dpa_job](dpa_job.md)") {
		t.Error("job_log should link to job")
	}

	// job references project
	jobDoc, _ := os.ReadFile(filepath.Join(tablesDir, "dpa_job.md"))
	if !strings.Contains(string(jobDoc), "[dpa_project](dpa_project.md)") {
		t.Error("job should link to project")
	}

	// project references web_user
	projectDoc, _ := os.ReadFile(filepath.Join(tablesDir, "dpa_project.md"))
	if !strings.Contains(string(projectDoc), "[dpa_web_user](dpa_web_user.md)") {
		t.Error("project should link to web_user")
	}
}

func TestGoAnywhereMarkdown_ColumnDetails(t *testing.T) {
	raw, _ := schema.LoadRaw("../testdata/goanywhere_sample.json")
	s := schema.ConvertRawToSchema(raw)
	dir := t.TempDir()

	Generate(s, dir)

	tablesDir := filepath.Join(dir, "tables")
	data, _ := os.ReadFile(filepath.Join(tablesDir, "dpa_web_user.md"))
	doc := string(data)

	// Check column types with length
	if !strings.Contains(doc, "varchar(100)") {
		t.Error("missing varchar(100) type for username")
	}
	if !strings.Contains(doc, "varchar(255)") {
		t.Error("missing varchar(255) type for email")
	}

	// Check nullable indicators
	if !strings.Contains(doc, "| NO |") {
		t.Error("missing NO for non-nullable columns")
	}
	if !strings.Contains(doc, "| YES |") {
		t.Error("missing YES for nullable columns")
	}

	// Check default values
	if !strings.Contains(doc, "CURRENT_TIMESTAMP") {
		t.Error("missing CURRENT_TIMESTAMP default")
	}
}

func TestGoAnywhereMarkdown_IndexDetails(t *testing.T) {
	raw, _ := schema.LoadRaw("../testdata/goanywhere_sample.json")
	s := schema.ConvertRawToSchema(raw)
	dir := t.TempDir()

	Generate(s, dir)

	// Check composite PK index in dpa_addr_book_con_group_map
	data, _ := os.ReadFile(filepath.Join(dir, "tables", "dpa_addr_book_con_group_map.md"))
	doc := string(data)

	if !strings.Contains(doc, "## Indexes") {
		t.Error("missing Indexes section")
	}

	// Should show composite index columns
	if !strings.Contains(doc, "contact_id") && !strings.Contains(doc, "group_id") {
		t.Error("missing composite index columns")
	}
}

func TestGoAnywhereMarkdown_DomainGrouping(t *testing.T) {
	raw, _ := schema.LoadRaw("../testdata/goanywhere_sample.json")
	s := schema.ConvertRawToSchema(raw)
	dir := t.TempDir()

	Generate(s, dir)

	data, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readme := string(data)

	// Tables should be grouped by prefix (dpa_addr, dpa_job, etc.)
	// The TOC uses ### for group headers
	if !strings.Contains(readme, "### dpa_") {
		t.Error("README should group tables by prefix")
	}
}

// TestGoAnywhereMarkdown_FullSchema tests against the complete GoAnywhere schema
func TestGoAnywhereMarkdown_FullSchema(t *testing.T) {
	fullPath := os.Getenv("GOANYWHERE_SCHEMA_PATH")
	if fullPath == "" {
		t.Skip("Set GOANYWHERE_SCHEMA_PATH to run full schema markdown test")
	}

	raw, err := schema.LoadRaw(fullPath)
	if err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	s := schema.ConvertRawToSchema(raw)
	dir := t.TempDir()

	err = Generate(s, dir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify scale
	files, _ := os.ReadDir(filepath.Join(dir, "tables"))
	if len(files) < 200 {
		t.Errorf("expected 200+ table files, got %d", len(files))
	}

	// README should exist and be substantial
	readmeInfo, _ := os.Stat(filepath.Join(dir, "README.md"))
	if readmeInfo.Size() < 10000 {
		t.Errorf("README too small for 200+ tables: %d bytes", readmeInfo.Size())
	}

	// Spot check a few table docs exist
	spotChecks := []string{"dpa_web_user.md", "dpa_project.md", "dpa_job.md"}
	for _, f := range spotChecks {
		if _, err := os.Stat(filepath.Join(dir, "tables", f)); os.IsNotExist(err) {
			t.Errorf("missing table doc: %s", f)
		}
	}
}
