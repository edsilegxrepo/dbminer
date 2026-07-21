//nolint:errcheck // errWriter pattern captures errors internally

// Export functions write the schema to various output formats: JSON (ChartDB-compatible),
// NDJSON (streaming, LLM-friendly), and TSV (combined or split for spreadsheet analysis).
package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const FileMode = 0o644

// errWriter wraps a writer and captures the first error.
// This allows multiple fmt.Fprint* calls without checking each return value,
// while still propagating any error at the end via ew.err.
// The linter doesn't understand this pattern, so we use //nolint:errcheck on callers.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Write(p []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, err := ew.w.Write(p)
	if err != nil {
		ew.err = err
	}
	return n, err
}

type ExportFormat string

const (
	FormatJSON   ExportFormat = "json"
	FormatNDJSON ExportFormat = "ndjson"
	FormatTSV    ExportFormat = "tsv"
)

// ExportOptions configures export behavior
type ExportOptions struct {
	Format   ExportFormat
	TSVSplit bool // For TSV: split into separate files per entity
}

// Export writes the schema to the specified file in the given format
func Export(s *Schema, filename string, opts ExportOptions) error {
	switch opts.Format {
	case FormatJSON:
		return exportJSON(s, filename)
	case FormatNDJSON:
		return exportNDJSON(s, filename)
	case FormatTSV:
		return exportTSV(s, filename, opts.TSVSplit)
	default:
		return fmt.Errorf("unsupported export format: %s", opts.Format)
	}
}

// exportJSON writes ChartDB-compatible JSON
func exportJSON(s *Schema, filename string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, FileMode)
}

// exportNDJSON writes newline-delimited JSON (one record per line)
func exportNDJSON(s *Schema, filename string) (rerr error) {
	f, err := os.Create(filename) // #nosec G304 -- User-provided path is by design for CLI tool
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && rerr == nil {
			rerr = cerr
		}
	}()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)

	// Write metadata record
	meta := map[string]interface{}{
		"_type":        "metadata",
		"id":           s.ID,
		"name":         s.Name,
		"databaseType": s.DatabaseType,
		"createdAt":    s.CreatedAt,
		"updatedAt":    s.UpdatedAt,
		"tableCount":   len(s.Tables),
		"relCount":     len(s.Relationships),
		"triggerCount": len(s.Triggers),
		"spCount":      len(s.StoredProcedures),
	}
	if err := enc.Encode(meta); err != nil {
		return err
	}

	// Write each table as a record
	for _, t := range s.Tables {
		rec := map[string]interface{}{
			"_type":   "table",
			"id":      t.ID,
			"name":    t.Name,
			"schema":  t.Schema,
			"comment": t.Comment,
			"isView":  t.IsView,
			"fields":  t.Fields,
			"indexes": t.Indexes,
		}
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}

	// Write each relationship as a record
	for _, r := range s.Relationships {
		rec := map[string]interface{}{
			"_type":             "relationship",
			"id":                r.ID,
			"name":              r.Name,
			"sourceTableId":     r.SourceTableID,
			"targetTableId":     r.TargetTableID,
			"sourceFieldId":     r.SourceFieldID,
			"targetFieldId":     r.TargetFieldID,
			"sourceCardinality": r.SourceCardinality,
			"targetCardinality": r.TargetCardinality,
		}
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}

	// Write each trigger as a record
	for _, t := range s.Triggers {
		rec := map[string]interface{}{
			"_type":     "trigger",
			"id":        t.ID,
			"name":      t.Name,
			"schema":    t.Schema,
			"table":     t.Table,
			"timing":    t.Timing,
			"event":     t.Event,
			"statement": t.Statement,
		}
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}

	// Write each stored procedure/function as a record
	for _, sp := range s.StoredProcedures {
		rec := map[string]interface{}{
			"_type":      "stored_procedure",
			"id":         sp.ID,
			"name":       sp.Name,
			"schema":     sp.Schema,
			"type":       sp.Type,
			"returnType": sp.ReturnType,
			"parameters": sp.Parameters,
			"comment":    sp.Comment,
		}
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}

	return nil
}

// exportTSV writes tab-separated values
func exportTSV(s *Schema, filename string, split bool) error {
	if split {
		return exportTSVSplit(s, filename)
	}
	return exportTSVCombined(s, filename)
}

// exportTSVCombined writes all entities to a single TSV file with a type column
func exportTSVCombined(s *Schema, filename string) (rerr error) {
	f, err := os.Create(filename) // #nosec G304 -- User-provided path is by design for CLI tool
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && rerr == nil {
			rerr = cerr
		}
	}()

	// Use errWriter to capture first write error (see errWriter doc for why //nolint:errcheck)
	ew := &errWriter{w: f}

	// Build lookup maps for names
	tableMap := make(map[string]*Table)
	fieldMap := make(map[string]*Field)
	for i := range s.Tables {
		t := &s.Tables[i]
		tableMap[t.ID] = t
		for j := range t.Fields {
			fieldMap[t.Fields[j].ID] = &t.Fields[j]
		}
	}

	// Header
	fmt.Fprintln(ew, "type\ttable_schema\ttable_name\tcolumn_name\tdata_type\tis_pk\tis_nullable\tdefault\tcomment\tfk_ref_table\tfk_ref_column")

	// Tables
	for _, t := range s.Tables {
		kind := "TABLE"
		if t.IsView {
			kind = "VIEW"
		}
		fmt.Fprintf(ew, "%s\t%s\t%s\t\t\t\t\t\t%s\t\t\n",
			kind,
			escapeTSV(t.Schema),
			escapeTSV(t.Name),
			escapeTSV(t.Comment))

		// Columns
		for _, field := range t.Fields {
			pk := ""
			if field.PrimaryKey {
				pk = "Y"
			}
			nullable := ""
			if field.Nullable {
				nullable = "Y"
			}

			// Check if this field is a FK source
			fkRefTable := ""
			fkRefCol := ""
			for _, rel := range s.Relationships {
				if rel.SourceFieldID == field.ID {
					if tgt := tableMap[rel.TargetTableID]; tgt != nil {
						fkRefTable = tgt.Name
					}
					if tgtField := fieldMap[rel.TargetFieldID]; tgtField != nil {
						fkRefCol = tgtField.Name
					}
					break
				}
			}

			fmt.Fprintf(ew, "COLUMN\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				escapeTSV(t.Schema),
				escapeTSV(t.Name),
				escapeTSV(field.Name),
				escapeTSV(field.Type.Name),
				pk,
				nullable,
				escapeTSV(field.Default),
				escapeTSV(field.Comment),
				escapeTSV(fkRefTable),
				escapeTSV(fkRefCol))
		}

		// Indexes
		for _, idx := range t.Indexes {
			var colNames []string
			for _, fid := range idx.FieldIDs {
				if fld := fieldMap[fid]; fld != nil {
					colNames = append(colNames, fld.Name)
				}
			}
			unique := ""
			if idx.Unique {
				unique = "Y"
			}
			fmt.Fprintf(ew, "INDEX\t%s\t%s\t%s\t%s\t\t\t\t\t\t\n",
				escapeTSV(t.Schema),
				escapeTSV(t.Name),
				escapeTSV(idx.Name),
				unique+":"+strings.Join(colNames, ","))
		}
	}

	// Triggers
	for _, trig := range s.Triggers {
		fmt.Fprintf(ew, "TRIGGER\t%s\t%s\t%s\t%s/%s\t\t\t\t\t\t\n",
			escapeTSV(trig.Schema),
			escapeTSV(trig.Table),
			escapeTSV(trig.Name),
			escapeTSV(trig.Timing),
			escapeTSV(trig.Event))
	}

	// Stored Procedures
	for _, sp := range s.StoredProcedures {
		var paramStrs []string
		for _, p := range sp.Parameters {
			paramStrs = append(paramStrs, p.Mode+" "+p.Name+" "+p.Type)
		}
		fmt.Fprintf(ew, "%s\t%s\t\t%s\t%s\t\t\t\t%s\t\t\n",
			escapeTSV(sp.Type),
			escapeTSV(sp.Schema),
			escapeTSV(sp.Name),
			escapeTSV(strings.Join(paramStrs, "; ")),
			escapeTSV(sp.Comment))
	}

	return ew.err
}

// exportTSVSplit writes separate TSV files for tables, columns, relationships, triggers, procedures
func exportTSVSplit(s *Schema, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- 0755 is standard for user-created directories
		return err
	}

	// Build lookup maps
	tableMap := make(map[string]*Table)
	fieldMap := make(map[string]*Field)
	for i := range s.Tables {
		t := &s.Tables[i]
		tableMap[t.ID] = t
		for j := range t.Fields {
			fieldMap[t.Fields[j].ID] = &t.Fields[j]
		}
	}

	// tables.tsv
	if err := writeTSVFile(filepath.Join(dir, "tables.tsv"), func(w *errWriter) {
		fmt.Fprintln(w, "id\tschema\tname\ttype\tcomment")
		for _, t := range s.Tables {
			kind := "TABLE"
			if t.IsView {
				kind = "VIEW"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				t.ID, escapeTSV(t.Schema), escapeTSV(t.Name), kind, escapeTSV(t.Comment))
		}
	}); err != nil {
		return err
	}

	// columns.tsv
	if err := writeTSVFile(filepath.Join(dir, "columns.tsv"), func(w *errWriter) {
		fmt.Fprintln(w, "id\ttable_id\ttable_name\tname\ttype\tis_pk\tis_unique\tis_nullable\tdefault\tgenerated_expr\tgenerated_type\tcomment")
		for _, t := range s.Tables {
			for _, field := range t.Fields {
				pk, unique, nullable := "", "", ""
				if field.PrimaryKey {
					pk = "Y"
				}
				if field.Unique {
					unique = "Y"
				}
				if field.Nullable {
					nullable = "Y"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					field.ID, t.ID, escapeTSV(t.Name), escapeTSV(field.Name),
					escapeTSV(field.Type.Name), pk, unique, nullable,
					escapeTSV(field.Default), escapeTSV(field.GeneratedExpr),
					escapeTSV(field.GeneratedType), escapeTSV(field.Comment))
			}
		}
	}); err != nil {
		return err
	}

	// indexes.tsv
	if err := writeTSVFile(filepath.Join(dir, "indexes.tsv"), func(w *errWriter) {
		fmt.Fprintln(w, "id\ttable_id\ttable_name\tname\tis_unique\tcolumns")
		for _, t := range s.Tables {
			for _, idx := range t.Indexes {
				var colNames []string
				for _, fid := range idx.FieldIDs {
					if fld := fieldMap[fid]; fld != nil {
						colNames = append(colNames, fld.Name)
					}
				}
				unique := ""
				if idx.Unique {
					unique = "Y"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					idx.ID, t.ID, escapeTSV(t.Name), escapeTSV(idx.Name),
					unique, strings.Join(colNames, ","))
			}
		}
	}); err != nil {
		return err
	}

	// relationships.tsv
	if err := writeTSVFile(filepath.Join(dir, "relationships.tsv"), func(w *errWriter) {
		fmt.Fprintln(w, "id\tname\tsrc_table\tsrc_column\ttgt_table\ttgt_column\tcardinality")
		for _, rel := range s.Relationships {
			srcTable, tgtTable := "", ""
			srcCol, tgtCol := "", ""
			if t := tableMap[rel.SourceTableID]; t != nil {
				srcTable = t.Name
			}
			if t := tableMap[rel.TargetTableID]; t != nil {
				tgtTable = t.Name
			}
			if fld := fieldMap[rel.SourceFieldID]; fld != nil {
				srcCol = fld.Name
			}
			if fld := fieldMap[rel.TargetFieldID]; fld != nil {
				tgtCol = fld.Name
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s:%s\n",
				rel.ID, escapeTSV(rel.Name),
				escapeTSV(srcTable), escapeTSV(srcCol),
				escapeTSV(tgtTable), escapeTSV(tgtCol),
				rel.SourceCardinality, rel.TargetCardinality)
		}
	}); err != nil {
		return err
	}

	// triggers.tsv
	if len(s.Triggers) > 0 {
		if err := writeTSVFile(filepath.Join(dir, "triggers.tsv"), func(w *errWriter) {
			fmt.Fprintln(w, "id\tschema\ttable\tname\ttiming\tevent")
			for _, trig := range s.Triggers {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					trig.ID, escapeTSV(trig.Schema), escapeTSV(trig.Table),
					escapeTSV(trig.Name), escapeTSV(trig.Timing), escapeTSV(trig.Event))
			}
		}); err != nil {
			return err
		}
	}

	// procedures.tsv
	if len(s.StoredProcedures) > 0 {
		if err := writeTSVFile(filepath.Join(dir, "procedures.tsv"), func(w *errWriter) {
			fmt.Fprintln(w, "id\tschema\tname\ttype\treturn_type\tparameters\tcomment")
			for _, sp := range s.StoredProcedures {
				var paramStrs []string
				for _, p := range sp.Parameters {
					paramStrs = append(paramStrs, p.Mode+" "+p.Name+" "+p.Type)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					sp.ID, escapeTSV(sp.Schema), escapeTSV(sp.Name),
					escapeTSV(sp.Type), escapeTSV(sp.ReturnType),
					escapeTSV(strings.Join(paramStrs, "; ")), escapeTSV(sp.Comment))
			}
		}); err != nil {
			return err
		}
	}

	return nil
}

// writeTSVFile creates a file and ensures it's closed properly with error checking
func writeTSVFile(path string, writeFn func(w *errWriter)) (rerr error) {
	f, err := os.Create(path) // #nosec G304 -- User-provided path is by design for CLI tool
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && rerr == nil {
			rerr = cerr
		}
	}()
	ew := &errWriter{w: f}
	writeFn(ew)
	return ew.err
}

func escapeTSV(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
