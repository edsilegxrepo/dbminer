package schema

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// ConvertRawToSchema transforms raw SQL collector output into the unified Schema format.
// Performs single-pass conversion with O(t+c+i+r) complexity for tables, columns, indexes, and relationships.
// Builds lookup maps for efficient PK detection and FK resolution across schemas.
func ConvertRawToSchema(raw *RawSchema) *Schema {
	now := time.Now()
	schema := &Schema{
		ID:           "0",
		Name:         raw.DatabaseName,
		DatabaseType: detectDatabaseType(raw),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Build lookup maps
	pkColumns := make(map[string]map[string]bool)
	for _, pk := range raw.PKInfo {
		if pkColumns[pk.Table] == nil {
			pkColumns[pk.Table] = make(map[string]bool)
		}
		pkColumns[pk.Table][pk.Column] = true
	}

	// Group columns by table
	tableColumns := make(map[string][]RawColumn)
	for _, col := range raw.Columns {
		tableColumns[col.Table] = append(tableColumns[col.Table], col)
	}

	// Sort columns by ordinal position
	for table := range tableColumns {
		sort.Slice(tableColumns[table], func(i, j int) bool {
			return tableColumns[table][i].OrdinalPos < tableColumns[table][j].OrdinalPos
		})
	}

	// Group indexes by table and name
	tableIndexes := make(map[string]map[string][]RawIndex)
	for _, idx := range raw.Indexes {
		if tableIndexes[idx.Table] == nil {
			tableIndexes[idx.Table] = make(map[string][]RawIndex)
		}
		tableIndexes[idx.Table][idx.Name] = append(tableIndexes[idx.Table][idx.Name], idx)
	}

	// Sort index columns by position
	for table := range tableIndexes {
		for idxName := range tableIndexes[table] {
			sort.Slice(tableIndexes[table][idxName], func(i, j int) bool {
				return tableIndexes[table][idxName][i].ColPosition < tableIndexes[table][idxName][j].ColPosition
			})
		}
	}

	// ID counters
	nextID := 1
	fieldIDMap := make(map[string]map[string]string)

	// Process tables
	for _, rawTable := range raw.Tables {
		tableID := strconv.Itoa(nextID)
		nextID++

		table := Table{
			ID:        tableID,
			Name:      rawTable.Table,
			Schema:    rawTable.Schema,
			Comment:   rawTable.Comment,
			IsView:    false,
			X:         DefaultX,
			Y:         DefaultY,
			CreatedAt: now.UnixMilli(),
		}

		fieldIDMap[rawTable.Table] = make(map[string]string)

		// Add fields
		for _, rawCol := range tableColumns[rawTable.Table] {
			fieldID := strconv.Itoa(nextID)
			nextID++
			fieldIDMap[rawTable.Table][rawCol.Name] = fieldID

			isPK := pkColumns[rawTable.Table][rawCol.Name]
			isUnique := isPK

			// Check if column is unique from indexes
			for idxName, idxCols := range tableIndexes[rawTable.Table] {
				if len(idxCols) == 1 && idxCols[0].Column == rawCol.Name && toBool(idxCols[0].Unique) {
					if idxName != "PRIMARY" {
						isUnique = true
					}
				}
			}

			field := Field{
				ID:            fieldID,
				Name:          rawCol.Name,
				Type:          FieldType{ID: rawCol.Type, Name: rawCol.Type},
				PrimaryKey:    isPK,
				Unique:        isUnique,
				Nullable:      toBool(rawCol.Nullable),
				Default:       rawCol.Default,
				Comment:       rawCol.Comment,
				Collation:     rawCol.Collation,
				CharMaxLen:    rawCol.CharMaxLen,
				GeneratedExpr: rawCol.GeneratedExpr,
				GeneratedType: rawCol.GeneratedType,
			}

			table.Fields = append(table.Fields, field)
		}

		// Add indexes
		for idxName, idxCols := range tableIndexes[rawTable.Table] {
			indexID := strconv.Itoa(nextID)
			nextID++

			var fieldIDs []string
			for _, ic := range idxCols {
				if fid, ok := fieldIDMap[rawTable.Table][ic.Column]; ok {
					fieldIDs = append(fieldIDs, fid)
				}
			}

			idx := Index{
				ID:        indexID,
				Name:      idxName,
				Unique:    toBool(idxCols[0].Unique),
				FieldIDs:  fieldIDs,
				CreatedAt: now.UnixMilli(),
			}

			table.Indexes = append(table.Indexes, idx)
		}

		schema.Tables = append(schema.Tables, table)
	}

	// Process views
	for _, rawView := range raw.Views {
		tableID := strconv.Itoa(nextID)
		nextID++

		table := Table{
			ID:        tableID,
			Name:      rawView.ViewName,
			Schema:    rawView.Schema,
			IsView:    true,
			X:         DefaultX,
			Y:         DefaultY,
			CreatedAt: now.UnixMilli(),
		}

		fieldIDMap[rawView.ViewName] = make(map[string]string)

		// Add fields for view
		for _, rawCol := range tableColumns[rawView.ViewName] {
			fieldID := strconv.Itoa(nextID)
			nextID++
			fieldIDMap[rawView.ViewName][rawCol.Name] = fieldID

			field := Field{
				ID:            fieldID,
				Name:          rawCol.Name,
				Type:          FieldType{ID: rawCol.Type, Name: rawCol.Type},
				PrimaryKey:    false,
				Unique:        false,
				Nullable:      toBool(rawCol.Nullable),
				Default:       rawCol.Default,
				Comment:       rawCol.Comment,
				Collation:     rawCol.Collation,
				CharMaxLen:    rawCol.CharMaxLen,
				GeneratedExpr: rawCol.GeneratedExpr,
				GeneratedType: rawCol.GeneratedType,
			}

			table.Fields = append(table.Fields, field)
		}

		schema.Tables = append(schema.Tables, table)
	}

	// Process triggers
	for _, rawTrig := range raw.Triggers {
		trigID := strconv.Itoa(nextID)
		nextID++

		trig := Trigger{
			ID:        trigID,
			Name:      rawTrig.Name,
			Schema:    rawTrig.Schema,
			Table:     rawTrig.Table,
			Timing:    rawTrig.Timing,
			Event:     rawTrig.Event,
			Statement: rawTrig.Statement,
		}
		schema.Triggers = append(schema.Triggers, trig)
	}

	// Process stored procedures
	for _, rawSP := range raw.StoredProcedures {
		spID := strconv.Itoa(nextID)
		nextID++

		var params []ProcedureParam
		for _, p := range rawSP.Parameters {
			params = append(params, ProcedureParam(p))
		}

		sp := StoredProcedure{
			ID:         spID,
			Name:       rawSP.Name,
			Schema:     rawSP.Schema,
			Type:       rawSP.Type,
			ReturnType: rawSP.ReturnType,
			Parameters: params,
			Comment:    rawSP.Comment,
		}
		schema.StoredProcedures = append(schema.StoredProcedures, sp)
	}

	// Build table ID map (schema.table for uniqueness)
	tableIDMap := make(map[string]string)
	for _, t := range schema.Tables {
		key := t.Name
		if t.Schema != "" {
			key = t.Schema + "." + t.Name
		}
		tableIDMap[key] = t.ID
		tableIDMap[t.Name] = t.ID
	}

	// Process relationships (foreign keys)
	for _, fk := range raw.FKInfo {
		relID := strconv.Itoa(nextID)
		nextID++

		srcKey := fk.Table
		if fk.Schema != "" {
			srcKey = fk.Schema + "." + fk.Table
		}
		tgtKey := fk.ReferenceTable
		if fk.ReferenceSchema != "" {
			tgtKey = fk.ReferenceSchema + "." + fk.ReferenceTable
		}

		srcTableID := tableIDMap[srcKey]
		if srcTableID == "" {
			srcTableID = tableIDMap[fk.Table]
		}
		tgtTableID := tableIDMap[tgtKey]
		if tgtTableID == "" {
			tgtTableID = tableIDMap[fk.ReferenceTable]
		}
		srcFieldID := ""
		tgtFieldID := ""

		if fieldIDMap[fk.Table] != nil {
			srcFieldID = fieldIDMap[fk.Table][fk.Column]
		}
		if fieldIDMap[fk.ReferenceTable] != nil {
			tgtFieldID = fieldIDMap[fk.ReferenceTable][fk.ReferenceColumn]
		}

		if srcTableID == "" || tgtTableID == "" || srcFieldID == "" || tgtFieldID == "" {
			continue
		}

		rel := Relationship{
			ID:                relID,
			Name:              fk.ForeignKeyName,
			SourceTableID:     srcTableID,
			TargetTableID:     tgtTableID,
			SourceFieldID:     srcFieldID,
			TargetFieldID:     tgtFieldID,
			SourceCardinality: "one",
			TargetCardinality: "many",
		}

		schema.Relationships = append(schema.Relationships, rel)
	}

	return schema
}

// detectDatabaseType infers the database type from version strings and schema content.
// Falls back to "mysql" if no specific pattern matches.
func detectDatabaseType(raw *RawSchema) string {
	version := strings.ToLower(raw.Version)
	dbName := strings.ToLower(raw.DatabaseName)

	if strings.Contains(version, "postgresql") || strings.Contains(version, "postgres") {
		return "postgresql"
	}
	if strings.Contains(version, "mariadb") {
		return "mariadb"
	}
	if strings.Contains(version, "microsoft") || strings.Contains(version, "sql server") {
		return "sqlserver"
	}
	if strings.Contains(version, "oracle") || strings.Contains(dbName, "oracle") {
		return "oracle"
	}
	if dbName == "sqlite" || strings.Contains(version, "sqlite") {
		return "sqlite"
	}
	// MongoDB: check for tables with type "COLLECTION" or engine "MongoDB"
	for _, t := range raw.Tables {
		if strings.ToLower(t.Type) == "collection" || strings.ToLower(t.Engine) == "mongodb" {
			return "mongodb"
		}
	}
	if len(raw.CustomTypes) > 0 {
		return "postgresql"
	}
	return "mysql"
}

// toBool coerces various types to boolean. Handles MySQL's "YES"/"NO" nullable format,
// numeric 0/1, and standard boolean strings.
func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		lower := strings.ToLower(val)
		return lower == "true" || lower == "1" || lower == "yes" || lower == "y"
	case float64:
		return val != 0
	case int:
		return val != 0
	default:
		return false
	}
}
