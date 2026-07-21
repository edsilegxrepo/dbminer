package schema

import (
	"encoding/json"
	"os"
)

// Raw JSON structures (from SQL collector output)
type RawSchema struct {
	FKInfo           []RawFKInfo          `json:"fk_info"`
	PKInfo           []RawPKInfo          `json:"pk_info"`
	Columns          []RawColumn          `json:"columns"`
	Indexes          []RawIndex           `json:"indexes"`
	Tables           []RawTable           `json:"tables"`
	Views            []RawView            `json:"views"`
	Triggers         []RawTrigger         `json:"triggers,omitempty"`
	StoredProcedures []RawStoredProcedure `json:"stored_procedures,omitempty"`
	CheckConstraints []RawCheckConstraint `json:"check_constraints,omitempty"`
	CustomTypes      []RawCustomType      `json:"custom_types,omitempty"`
	DatabaseName     string               `json:"database_name"`
	Version          string               `json:"version"`
}

type RawTrigger struct {
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	Name      string `json:"name"`
	Timing    string `json:"timing"`    // BEFORE, AFTER, INSTEAD OF
	Event     string `json:"event"`     // INSERT, UPDATE, DELETE
	Statement string `json:"statement"` // Trigger body (optional, may be empty for security)
}

type RawStoredProcedure struct {
	Schema     string              `json:"schema"`
	Name       string              `json:"name"`
	Type       string              `json:"type"`        // PROCEDURE or FUNCTION
	ReturnType string              `json:"return_type"` // For functions
	Parameters []RawProcedureParam `json:"parameters,omitempty"`
	Comment    string              `json:"comment,omitempty"`
}

type RawProcedureParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Mode     string `json:"mode"` // IN, OUT, INOUT
	Position int    `json:"position"`
}

type RawCheckConstraint struct {
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	Expression string `json:"expression"`
}

type RawCustomType struct {
	Schema string   `json:"schema"`
	Type   string   `json:"type"`
	Kind   string   `json:"kind"`
	Values []string `json:"values,omitempty"`
}

type RawFKInfo struct {
	Schema          string `json:"schema"`
	Table           string `json:"table"`
	Column          string `json:"column"`
	ForeignKeyName  string `json:"foreign_key_name"`
	ReferenceSchema string `json:"reference_schema"`
	ReferenceTable  string `json:"reference_table"`
	ReferenceColumn string `json:"reference_column"`
	FKDef           string `json:"fk_def"`
}

type RawPKInfo struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Column string `json:"column"`
	PKDef  string `json:"pk_def"`
}

type RawColumn struct {
	Schema        string      `json:"schema"`
	Table         string      `json:"table"`
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	CharMaxLen    string      `json:"character_maximum_length"`
	Precision     interface{} `json:"precision"`
	OrdinalPos    int         `json:"ordinal_position"`
	Nullable      interface{} `json:"nullable"`
	Default       string      `json:"default"`
	Collation     string      `json:"collation"`
	IsIdentity    interface{} `json:"is_identity"`
	Comment       string      `json:"comment"`
	GeneratedExpr string      `json:"generated_expression,omitempty"` // For computed/generated columns
	GeneratedType string      `json:"generated_type,omitempty"`       // STORED or VIRTUAL
}

type RawIndex struct {
	Schema      string      `json:"schema"`
	Table       string      `json:"table"`
	Name        string      `json:"name"`
	Size        interface{} `json:"size"`
	Column      string      `json:"column"`
	IndexType   string      `json:"index_type"`
	Cardinality interface{} `json:"cardinality"`
	Direction   string      `json:"direction"`
	ColPosition int         `json:"column_position"`
	Unique      interface{} `json:"unique"`
}

type RawTable struct {
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	Rows      int    `json:"rows"`
	Type      string `json:"type"`
	Engine    string `json:"engine"`
	Collation string `json:"collation"`
	Comment   string `json:"comment"`
}

type RawView struct {
	Schema         string `json:"schema"`
	ViewName       string `json:"view_name"`
	ViewDefinition string `json:"view_definition"`
}

// LoadRaw reads a raw schema JSON file from SQL collector output
func LoadRaw(filename string) (*RawSchema, error) {
	data, err := os.ReadFile(filename) // #nosec G304 -- User-provided path is by design for CLI tool
	if err != nil {
		return nil, err
	}

	var raw RawSchema
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	return &raw, nil
}
