// Package schema defines the unified internal schema representation and provides
// functions for loading raw SQL collector output, converting to the internal format,
// and exporting to various output formats (JSON, NDJSON, TSV).
package schema

import "time"

const (
	DefaultX = 100
	DefaultY = 100
)

// Schema represents the unified internal schema format
type Schema struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	DatabaseType     string            `json:"databaseType"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
	Tables           []Table           `json:"tables"`
	Relationships    []Relationship    `json:"relationships"`
	Triggers         []Trigger         `json:"triggers,omitempty"`
	StoredProcedures []StoredProcedure `json:"storedProcedures,omitempty"`
}

type Table struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Schema    string  `json:"schema"`
	Comment   string  `json:"comment"`
	IsView    bool    `json:"isView"`
	Fields    []Field `json:"fields"`
	Indexes   []Index `json:"indexes"`
	Color     string  `json:"color"`
	X         int     `json:"x"`
	Y         int     `json:"y"`
	CreatedAt int64   `json:"createdAt"`
}

type Field struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          FieldType `json:"type"`
	PrimaryKey    bool      `json:"primaryKey"`
	Unique        bool      `json:"unique"`
	Nullable      bool      `json:"nullable"`
	Default       string    `json:"default"`
	Comment       string    `json:"comment"`
	Collation     string    `json:"collation"`
	CharMaxLen    string    `json:"characterMaximumLength"`
	GeneratedExpr string    `json:"generatedExpression,omitempty"` // Expression for computed columns
	GeneratedType string    `json:"generatedType,omitempty"`       // STORED or VIRTUAL
}

type FieldType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Index struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Unique    bool     `json:"unique"`
	FieldIDs  []string `json:"fieldIds"`
	CreatedAt int64    `json:"createdAt"`
}

type Relationship struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	SourceTableID     string `json:"sourceTableId"`
	TargetTableID     string `json:"targetTableId"`
	SourceFieldID     string `json:"sourceFieldId"`
	TargetFieldID     string `json:"targetFieldId"`
	SourceCardinality string `json:"sourceCardinality"`
	TargetCardinality string `json:"targetCardinality"`
}

type Trigger struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	Timing    string `json:"timing"`    // BEFORE, AFTER, INSTEAD OF
	Event     string `json:"event"`     // INSERT, UPDATE, DELETE
	Statement string `json:"statement"` // Trigger body (may be empty)
}

type StoredProcedure struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Schema     string           `json:"schema"`
	Type       string           `json:"type"` // PROCEDURE or FUNCTION
	ReturnType string           `json:"returnType,omitempty"`
	Parameters []ProcedureParam `json:"parameters,omitempty"`
	Comment    string           `json:"comment,omitempty"`
}

type ProcedureParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Mode     string `json:"mode"` // IN, OUT, INOUT
	Position int    `json:"position"`
}
