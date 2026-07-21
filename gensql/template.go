package gensql

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/*.sql.tmpl templates/*.js.tmpl
var templateFS embed.FS

// templateNames maps driver names to their template extensions
var templateExtensions = map[string]string{
	"mysql":     "sql",
	"mysql57":   "sql",
	"mariadb":   "sql",
	"postgres":  "sql",
	"sqlite":    "sql",
	"mssql":     "sql",
	"mssql2016": "sql",
	"oracle":    "sql",
	"mongodb":   "js",
}

func init() {
	for name, ext := range templateExtensions {
		tmplPath := fmt.Sprintf("templates/%s.%s.tmpl", name, ext)
		if _, err := template.ParseFS(templateFS, tmplPath); err != nil {
			panic(fmt.Sprintf("failed to parse embedded template %s: %v", name, err))
		}
	}
}

// TemplateVars contains variables passed to SQL templates
type TemplateVars struct {
	IncludeAdmin bool   // Include queries requiring admin privileges
	StoredProc   bool   // Generate stored procedure wrapper
	DBName       string // Database name (for USE statements)
	SPName       string // Stored procedure name
	SampleSize   int    // MongoDB: documents to sample per collection (default 100)
	MaxDepth     int    // MongoDB: nested object expansion depth (default 2)
}

// ExecuteTemplate renders a SQL/JS template with the given variables
func ExecuteTemplate(name string, vars TemplateVars) (string, error) {
	ext, ok := templateExtensions[name]
	if !ok {
		ext = "sql" // default to SQL
	}
	tmplPath := fmt.Sprintf("templates/%s.%s.tmpl", name, ext)

	tmpl, err := template.ParseFS(templateFS, tmplPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// WrapWithUser adds user impersonation comment/commands
func WrapWithUser(sql, execUser, dbType string) string {
	if execUser == "" {
		return sql
	}

	// Escape single quotes to prevent SQL injection
	safeUser := strings.ReplaceAll(execUser, "'", "''")

	switch dbType {
	case "mssql", "mssql2016":
		return fmt.Sprintf("-- Run as: %s\nEXECUTE AS USER = '%s';\nGO\n\n%s\nREVERT;\nGO\n", safeUser, safeUser, sql)
	default:
		return fmt.Sprintf("-- Run as: %s\n\n%s", safeUser, sql)
	}
}
