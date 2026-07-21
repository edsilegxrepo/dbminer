// Package markdown generates human-readable documentation from database schemas.
// Produces a README with schema overview and Mermaid ERD, plus per-table documentation
// showing columns, indexes, relationships, and table-specific ERD diagrams.
package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"criticalsys.net/dbminer/schema"
)

const FileMode = 0o644

var (
	mermaidIDRegex    = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	safeFilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-.]`)
)

// Options configures markdown generation behavior.
type Options struct {
	GroupBy string // auto, prefix, schema, none
}

// Generate creates markdown documentation for the schema. Produces README.md
// with overview and domain-level ERD, plus individual table docs in tables/ subdirectory.
func Generate(s *schema.Schema, outputDir string, opts Options) error {
	if opts.GroupBy == "" {
		opts.GroupBy = "auto"
	}
	tableMap := make(map[string]*schema.Table)
	fieldMap := make(map[string]*schema.Field)

	for i := range s.Tables {
		t := &s.Tables[i]
		tableMap[t.ID] = t
		for j := range t.Fields {
			fieldMap[t.Fields[j].ID] = &t.Fields[j]
		}
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil { // #nosec G301 -- 0755 is standard for user-created directories
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := generateReadme(s, outputDir, tableMap, fieldMap, opts); err != nil {
		return fmt.Errorf("generating README: %w", err)
	}

	tablesDir := filepath.Join(outputDir, "tables")
	if err := os.MkdirAll(tablesDir, 0o755); err != nil { // #nosec G301 -- 0755 is standard for user-created directories
		return fmt.Errorf("creating tables directory: %w", err)
	}

	for i := range s.Tables {
		t := &s.Tables[i]
		if err := generateTableDoc(t, s, tablesDir, tableMap, fieldMap); err != nil {
			return fmt.Errorf("generating doc for %s: %w", t.Name, err)
		}
	}

	return nil
}

// detectDelimiter analyzes table names to find the most common delimiter.
// Returns the delimiter if >50% of tables contain it, empty string otherwise.
func detectDelimiter(tables []schema.Table) string {
	if len(tables) == 0 {
		return ""
	}
	counts := map[string]int{".": 0, "_": 0, "-": 0}
	for _, t := range tables {
		for delim := range counts {
			if strings.Contains(t.Name, delim) {
				counts[delim]++
			}
		}
	}
	threshold := len(tables) / 2
	// Check in order of precedence: dot (schema), underscore, hyphen
	for _, delim := range []string{".", "_", "-"} {
		if counts[delim] > threshold {
			return delim
		}
	}
	return ""
}

// getTablePrefix extracts domain prefix from table name based on delimiter.
func getTablePrefix(name, delimiter string) string {
	if delimiter == "" {
		return "other"
	}
	parts := strings.Split(name, delimiter)
	if len(parts) >= 2 {
		return parts[0] + delimiter + parts[1]
	}
	return "other"
}

// getTableGroup returns the grouping key for a table based on Options.GroupBy.
func getTableGroup(t *schema.Table, delimiter string, groupBy string) string {
	switch groupBy {
	case "none":
		return "Tables"
	case "schema":
		if t.Schema != "" {
			return t.Schema
		}
		return "default"
	case "prefix":
		return getTablePrefix(t.Name, "_")
	case "auto":
		fallthrough
	default:
		return getTablePrefix(t.Name, delimiter)
	}
}

// generateMermaidERD creates a domain-level flowchart showing table groups and cross-domain relationships.
func generateMermaidERD(s *schema.Schema, tableMap map[string]*schema.Table, fieldMap map[string]*schema.Field, opts Options) string {
	var sb strings.Builder

	delimiter := detectDelimiter(s.Tables)
	groupRelCounts := make(map[string]map[string]int)
	tableGroups := make(map[string][]string)
	connectedGroups := make(map[string]bool)

	for i := range s.Tables {
		t := &s.Tables[i]
		group := getTableGroup(t, delimiter, opts.GroupBy)
		tableGroups[group] = append(tableGroups[group], t.Name)
	}

	for _, rel := range s.Relationships {
		srcTable := tableMap[rel.SourceTableID]
		tgtTable := tableMap[rel.TargetTableID]
		if srcTable == nil || tgtTable == nil {
			continue
		}

		srcGroup := getTableGroup(srcTable, delimiter, opts.GroupBy)
		tgtGroup := getTableGroup(tgtTable, delimiter, opts.GroupBy)

		if srcGroup != tgtGroup {
			connectedGroups[srcGroup] = true
			connectedGroups[tgtGroup] = true

			if groupRelCounts[srcGroup] == nil {
				groupRelCounts[srcGroup] = make(map[string]int)
			}
			groupRelCounts[srcGroup][tgtGroup]++
		}
	}

	sb.WriteString("```mermaid\nflowchart LR\n")

	for group := range connectedGroups {
		count := len(tableGroups[group])
		fmt.Fprintf(&sb, "    %s[%s<br/>%d tables]\n", sanitizeMermaidID(group), escapeMermaidLabel(group), count)
	}

	sb.WriteString("\n")

	seen := make(map[string]bool)
	for srcGroup, targets := range groupRelCounts {
		for tgtGroup, count := range targets {
			key := srcGroup + "|" + tgtGroup
			reverseKey := tgtGroup + "|" + srcGroup
			if seen[key] || seen[reverseKey] {
				continue
			}
			seen[key] = true

			fmt.Fprintf(&sb, "    %s -->|%d| %s\n",
				sanitizeMermaidID(srcGroup), count, sanitizeMermaidID(tgtGroup))
		}
	}

	sb.WriteString("```\n")
	return sb.String()
}

// generateTableMermaidERD creates a table-centric ERD showing direct relationships (inbound and outbound).
func generateTableMermaidERD(table *schema.Table, s *schema.Schema, tableMap map[string]*schema.Table, fieldMap map[string]*schema.Field) string {
	var sb strings.Builder

	var outbound, inbound []schema.Relationship
	for _, rel := range s.Relationships {
		if rel.SourceTableID == table.ID {
			outbound = append(outbound, rel)
		}
		if rel.TargetTableID == table.ID {
			inbound = append(inbound, rel)
		}
	}

	if len(outbound) == 0 && len(inbound) == 0 {
		return ""
	}

	sb.WriteString("```mermaid\nflowchart LR\n")

	fmt.Fprintf(&sb, "    %s[<strong>%s</strong>]\n", sanitizeMermaidID(table.Name), escapeMermaidLabel(table.Name))
	fmt.Fprintf(&sb, "    style %s fill:#4a90d9,stroke:#333,stroke-width:2px,color:#fff\n", sanitizeMermaidID(table.Name))

	outboundTables := make(map[string][]string)
	for _, rel := range outbound {
		tgtTable := tableMap[rel.TargetTableID]
		srcField := fieldMap[rel.SourceFieldID]
		if tgtTable == nil || srcField == nil {
			continue
		}
		outboundTables[tgtTable.Name] = append(outboundTables[tgtTable.Name], srcField.Name)
	}

	outboundNames := make([]string, 0, len(outboundTables))
	for name := range outboundTables {
		outboundNames = append(outboundNames, name)
	}
	sort.Strings(outboundNames)

	for _, tgtName := range outboundNames {
		fields := outboundTables[tgtName]
		sort.Strings(fields)
		fmt.Fprintf(&sb, "    %s[%s]\n", sanitizeMermaidID(tgtName), escapeMermaidLabel(tgtName))
		fmt.Fprintf(&sb, "    %s -->|%s| %s\n",
			sanitizeMermaidID(table.Name), escapeMermaidLabel(strings.Join(fields, ", ")), sanitizeMermaidID(tgtName))
	}

	inboundTables := make(map[string][]string)
	for _, rel := range inbound {
		srcTable := tableMap[rel.SourceTableID]
		srcField := fieldMap[rel.SourceFieldID]
		if srcTable == nil || srcField == nil {
			continue
		}
		inboundTables[srcTable.Name] = append(inboundTables[srcTable.Name], srcField.Name)
	}

	inboundNames := make([]string, 0, len(inboundTables))
	for name := range inboundTables {
		inboundNames = append(inboundNames, name)
	}
	sort.Strings(inboundNames)

	for _, srcName := range inboundNames {
		fields := inboundTables[srcName]
		sort.Strings(fields)
		if _, exists := outboundTables[srcName]; !exists {
			fmt.Fprintf(&sb, "    %s[%s]\n", sanitizeMermaidID(srcName), escapeMermaidLabel(srcName))
		}
		fmt.Fprintf(&sb, "    %s -->|%s| %s\n",
			sanitizeMermaidID(srcName), escapeMermaidLabel(strings.Join(fields, ", ")), sanitizeMermaidID(table.Name))
	}

	sb.WriteString("```\n")
	return sb.String()
}

func sanitizeMermaidID(s string) string {
	return mermaidIDRegex.ReplaceAllString(s, "_")
}

func escapeMarkdownCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func escapeMermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "|", "_")
	s = strings.ReplaceAll(s, "[", "_")
	s = strings.ReplaceAll(s, "]", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	return s
}

func sanitizeFilename(name string) string {
	safe := safeFilenameRegex.ReplaceAllString(name, "_")
	if safe == "" || safe == "." || safe == ".." {
		safe = "_invalid_"
	}
	if len(safe) > 200 {
		safe = safe[:200]
	}
	return safe
}

func generateReadme(s *schema.Schema, outputDir string, tableMap map[string]*schema.Table, fieldMap map[string]*schema.Field, opts Options) error {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s Database Schema\n\n", escapeMarkdownCell(s.Name))
	fmt.Fprintf(&sb, "**Database Type:** %s  \n", escapeMarkdownCell(s.DatabaseType))
	fmt.Fprintf(&sb, "**Generated:** %s  \n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "**Tables:** %d  \n", len(s.Tables))
	fmt.Fprintf(&sb, "**Relationships:** %d  \n", len(s.Relationships))
	if len(s.Triggers) > 0 {
		fmt.Fprintf(&sb, "**Triggers:** %d  \n", len(s.Triggers))
	}
	if len(s.StoredProcedures) > 0 {
		fmt.Fprintf(&sb, "**Stored Procedures/Functions:** %d  \n", len(s.StoredProcedures))
	}
	sb.WriteString("\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Schema Overview (by Domain)\n\n")
	sb.WriteString(generateMermaidERD(s, tableMap, fieldMap, opts))

	sb.WriteString("\n---\n\n")
	sb.WriteString("## Table of Contents\n\n")

	tables := make([]schema.Table, len(s.Tables))
	copy(tables, s.Tables)
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})

	delimiter := detectDelimiter(s.Tables)
	groups := make(map[string][]schema.Table)
	for i := range tables {
		t := &tables[i]
		group := getTableGroup(t, delimiter, opts.GroupBy)
		groups[group] = append(groups[group], *t)
	}

	groupKeys := make([]string, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	sort.Strings(groupKeys)

	for _, group := range groupKeys {
		fmt.Fprintf(&sb, "### %s\n\n", escapeMarkdownCell(group))
		for _, t := range groups[group] {
			icon := "📋"
			if t.IsView {
				icon = "👁️"
			}
			fmt.Fprintf(&sb, "- %s [%s](tables/%s.md) — %d columns\n", icon, escapeMarkdownCell(t.Name), sanitizeFilename(t.Name), len(t.Fields))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("## Relationships Overview\n\n")
	sb.WriteString("| Source Table | Source Column | → | Target Table | Target Column | Cardinality |\n")
	sb.WriteString("|--------------|---------------|---|--------------|---------------|-------------|\n")

	for _, rel := range s.Relationships {
		srcTable := tableMap[rel.SourceTableID]
		tgtTable := tableMap[rel.TargetTableID]
		srcField := fieldMap[rel.SourceFieldID]
		tgtField := fieldMap[rel.TargetFieldID]

		if srcTable == nil || tgtTable == nil || srcField == nil || tgtField == nil {
			continue
		}

		cardinality := fmt.Sprintf("%s → %s", rel.SourceCardinality, rel.TargetCardinality)
		fmt.Fprintf(&sb, "| [%s](tables/%s.md) | %s | → | [%s](tables/%s.md) | %s | %s |\n",
			escapeMarkdownCell(srcTable.Name), sanitizeFilename(srcTable.Name), escapeMarkdownCell(srcField.Name),
			escapeMarkdownCell(tgtTable.Name), sanitizeFilename(tgtTable.Name), escapeMarkdownCell(tgtField.Name),
			cardinality)
	}

	// Triggers section
	if len(s.Triggers) > 0 {
		sb.WriteString("\n---\n\n")
		sb.WriteString("## Triggers\n\n")
		sb.WriteString("| Table | Trigger Name | Timing | Event |\n")
		sb.WriteString("|-------|--------------|--------|-------|\n")

		for _, trig := range s.Triggers {
			// Only link to table if it exists in our tables list
			tableCell := escapeMarkdownCell(trig.Table)
			for _, t := range s.Tables {
				if t.Name == trig.Table && (trig.Schema == "" || t.Schema == "" || trig.Schema == t.Schema) {
					tableCell = fmt.Sprintf("[%s](tables/%s.md)", escapeMarkdownCell(trig.Table), sanitizeFilename(trig.Table))
					break
				}
			}
			fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
				tableCell,
				escapeMarkdownCell(trig.Name), escapeMarkdownCell(trig.Timing), escapeMarkdownCell(trig.Event))
		}
	}

	// Stored procedures section
	if len(s.StoredProcedures) > 0 {
		sb.WriteString("\n---\n\n")
		sb.WriteString("## Stored Procedures & Functions\n\n")
		sb.WriteString("| Name | Type | Parameters | Return Type |\n")
		sb.WriteString("|------|------|------------|-------------|\n")

		for _, sp := range s.StoredProcedures {
			paramStrs := make([]string, 0, len(sp.Parameters))
			for _, p := range sp.Parameters {
				paramStrs = append(paramStrs, fmt.Sprintf("%s %s %s", p.Mode, p.Name, p.Type))
			}
			params := strings.Join(paramStrs, ", ")
			if params == "" {
				params = "-"
			}
			returnType := sp.ReturnType
			if returnType == "" {
				returnType = "-"
			}
			fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
				escapeMarkdownCell(sp.Name), escapeMarkdownCell(sp.Type),
				escapeMarkdownCell(params), escapeMarkdownCell(returnType))
		}
	}

	return os.WriteFile(filepath.Join(outputDir, "README.md"), []byte(sb.String()), FileMode)
}

func generateTableDoc(table *schema.Table, s *schema.Schema, outputDir string, tableMap map[string]*schema.Table, fieldMap map[string]*schema.Field) error {
	var sb strings.Builder

	tableType := "Table"
	if table.IsView {
		tableType = "View"
	}

	fmt.Fprintf(&sb, "# %s\n\n", escapeMarkdownCell(table.Name))
	fmt.Fprintf(&sb, "**Type:** %s  \n", tableType)
	fmt.Fprintf(&sb, "**Schema:** %s  \n", escapeMarkdownCell(table.Schema))
	if table.Comment != "" {
		fmt.Fprintf(&sb, "**Description:** %s  \n", escapeMarkdownCell(table.Comment))
	}
	fmt.Fprintf(&sb, "**Columns:** %d  \n\n", len(table.Fields))

	mermaidERD := generateTableMermaidERD(table, s, tableMap, fieldMap)
	if mermaidERD != "" {
		sb.WriteString("## Entity Relationship Diagram\n\n")
		sb.WriteString(mermaidERD)
		sb.WriteString("\n")
	}

	sb.WriteString("## Columns\n\n")
	sb.WriteString("| # | Name | Type | Nullable | Default | Key | Generated | Description |\n")
	sb.WriteString("|---|------|------|----------|---------|-----|-----------|-------------|\n")

	for i, field := range table.Fields {
		nullable := "YES"
		if !field.Nullable {
			nullable = "NO"
		}

		key := ""
		if field.PrimaryKey {
			key = "🔑 PK"
		} else if field.Unique {
			key = "🔒 UQ"
		}

		typeName := escapeMarkdownCell(field.Type.Name)
		if field.CharMaxLen != "" && field.CharMaxLen != "null" {
			typeName = fmt.Sprintf("%s(%s)", escapeMarkdownCell(field.Type.Name), escapeMarkdownCell(field.CharMaxLen))
		}

		defaultVal := escapeMarkdownCell(field.Default)
		if defaultVal == "" {
			defaultVal = "-"
		}

		generated := "-"
		if field.GeneratedExpr != "" && field.GeneratedType != "" {
			// Escape backticks in expression to avoid breaking markdown code spans
			escapedExpr := strings.ReplaceAll(escapeMarkdownCell(field.GeneratedExpr), "`", "'")
			generated = fmt.Sprintf("%s: `%s`", field.GeneratedType, escapedExpr)
		} else if field.GeneratedType != "" {
			generated = field.GeneratedType
		}

		comment := escapeMarkdownCell(field.Comment)
		if comment == "" {
			comment = "-"
		}

		fmt.Fprintf(&sb, "| %d | `%s` | %s | %s | %s | %s | %s | %s |\n",
			i+1, escapeMarkdownCell(field.Name), typeName, nullable, defaultVal, key, generated, comment)
	}

	if len(table.Indexes) > 0 {
		sb.WriteString("\n## Indexes\n\n")
		sb.WriteString("| Name | Columns | Unique |\n")
		sb.WriteString("|------|---------|--------|\n")

		for _, idx := range table.Indexes {
			var cols []string
			for _, fid := range idx.FieldIDs {
				if f := fieldMap[fid]; f != nil {
					cols = append(cols, escapeMarkdownCell(f.Name))
				}
			}

			unique := "No"
			if idx.Unique {
				unique = "Yes"
			}

			name := escapeMarkdownCell(idx.Name)
			if name == "" {
				name = "(primary)"
			}

			fmt.Fprintf(&sb, "| %s | %s | %s |\n", name, strings.Join(cols, ", "), unique)
		}
	}

	// Triggers for this table (match by name and schema)
	var tableTriggers []schema.Trigger
	for _, trig := range s.Triggers {
		if trig.Table == table.Name && (trig.Schema == "" || table.Schema == "" || trig.Schema == table.Schema) {
			tableTriggers = append(tableTriggers, trig)
		}
	}

	if len(tableTriggers) > 0 {
		sb.WriteString("\n## Triggers\n\n")
		sb.WriteString("| Name | Timing | Event |\n")
		sb.WriteString("|------|--------|-------|\n")

		for _, trig := range tableTriggers {
			fmt.Fprintf(&sb, "| %s | %s | %s |\n",
				escapeMarkdownCell(trig.Name), escapeMarkdownCell(trig.Timing), escapeMarkdownCell(trig.Event))
		}
	}

	var inbound, outbound []schema.Relationship
	for _, rel := range s.Relationships {
		if rel.SourceTableID == table.ID {
			outbound = append(outbound, rel)
		}
		if rel.TargetTableID == table.ID {
			inbound = append(inbound, rel)
		}
	}

	if len(outbound) > 0 || len(inbound) > 0 {
		sb.WriteString("\n## Relationships\n\n")

		if len(outbound) > 0 {
			sb.WriteString("### References (this table → other)\n\n")
			sb.WriteString("| Column | → | Referenced Table | Referenced Column |\n")
			sb.WriteString("|--------|---|------------------|-------------------|\n")

			for _, rel := range outbound {
				tgtTable := tableMap[rel.TargetTableID]
				srcField := fieldMap[rel.SourceFieldID]
				tgtField := fieldMap[rel.TargetFieldID]

				if tgtTable == nil || srcField == nil || tgtField == nil {
					continue
				}

				fmt.Fprintf(&sb, "| `%s` | → | [%s](%s.md) | `%s` |\n",
					escapeMarkdownCell(srcField.Name), escapeMarkdownCell(tgtTable.Name), sanitizeFilename(tgtTable.Name), escapeMarkdownCell(tgtField.Name))
			}
			sb.WriteString("\n")
		}

		if len(inbound) > 0 {
			sb.WriteString("### Referenced By (other → this table)\n\n")
			sb.WriteString("| Referencing Table | Referencing Column | → | Column |\n")
			sb.WriteString("|-------------------|--------------------|----|--------|\n")

			for _, rel := range inbound {
				srcTable := tableMap[rel.SourceTableID]
				srcField := fieldMap[rel.SourceFieldID]
				tgtField := fieldMap[rel.TargetFieldID]

				if srcTable == nil || srcField == nil || tgtField == nil {
					continue
				}

				fmt.Fprintf(&sb, "| [%s](%s.md) | `%s` | → | `%s` |\n",
					escapeMarkdownCell(srcTable.Name), sanitizeFilename(srcTable.Name), escapeMarkdownCell(srcField.Name), escapeMarkdownCell(tgtField.Name))
			}
		}
	}

	sb.WriteString("\n---\n")
	sb.WriteString("\n[← Back to Schema](../README.md)\n")

	return os.WriteFile(filepath.Join(outputDir, sanitizeFilename(table.Name)+".md"), []byte(sb.String()), FileMode)
}
