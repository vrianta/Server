package model

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/vrianta/agai/v1/config"
	DatabaseHandler "github.com/vrianta/agai/v1/database"
	log "github.com/vrianta/gulog"
)

// Function to get the table topology and compare with the latest FieldTypes and generate a new SQL queryBuilder to alter the table
// This function will be used to update the table structure if there are any changes in the FieldTypes
// syncTableSchema synchronizes the database table schema with the application's model schema.
// It iterates through the fields defined in the application's model and compares them
// against the existing schema in the database.  It performs the following actions:
//
//  1. Adds fields that exist in the model but not in the database.  This addition is
//     conditional based on user confirmation in build mode.
//
//  2. Modifies fields where discrepancies are found between the model and the database,
//     such as type mismatches, length differences, default value differences,
//     nullable constraints, or auto-increment settings. Modifications are also
//     conditional based on user confirmation in build mode.
//
//  3. Synchronizes index properties (UNIQUE, PRIMARY KEY, INDEX) between the model and
//     the database.  Index synchronization is conditional based on user confirmation
//     in build mode.
//
//  4. Removes fields that exist in the database but not in the model.  Removal is
//     conditional based on user confirmation in build mode.
//
// The function uses a `bufio.Reader` to prompt the user for confirmation when running
// in build mode (`config.GetBuild()`).  If not in build mode, it attempts to sync the
// schema without prompting.
//
// The function utilizes helper methods on the `meta` struct (m) such as `addField`,
// `modifyDBField`, `syncUniqueIndex`, `syncPrimaryKey`, `syncIndex`, and `removeDBField`
// to perform the actual database schema modifications.
//
// The function uses FieldTypeset and schemaMap to improve the lookup performance.
func (m *meta) syncTableSchema() {
	schemaMap := make(map[string]schema, len(m.schemas))
	for _, s := range m.schemas {
		schemaMap[s.field] = s
	}

	FieldTypeset := make(FieldTypeset, len(m.FieldTypes))
	for _, f := range m.FieldTypes {
		FieldTypeset[f.name] = f
	}

	var pendingAddFields []*Field

	reader := bufio.NewReader(os.Stdin)

	/* --------------------------------------------------------------------------
	   syncFields compares the model’s field definitions (`m.FieldTypes`)
	   with the live database schema (`schemaMap`) and:

	   • queues brand‑new columns for creation
	   • interactively (in build mode) or automatically modifies mismatched
	     columns and indexes so that DB and model stay in sync
	   --------------------------------------------------------------------------*/
	for _, field := range m.FieldTypes {

		// Look up the column in the database schema.
		schema, exists := schemaMap[field.name]
		if !exists {
			// ────────────── Field not found in DB ──────────────
			if config.GetBuild() { // interactive “build” mode?
				fmt.Printf("Field '%s' not in DB. Add? (y/n): ", field.name) // ask user
				if input, _ := reader.ReadString('\n'); strings.TrimSpace(input) != "y" {
					fmt.Printf("[AddField] Skipped: %s\n", field.name) // user said “no”
					continue
				}
			}
			// Defer actual DDL until later; collect it now.
			pendingAddFields = append(pendingAddFields, field)
			continue
		}

		// ────────────── Field exists – check for drift ──────────────
		filed_type, field_length := schema.parseSQLType() // DB column type & length
		shouldChange := false
		reasons := []string{} // track what’s different for user prompt

		if !field.Compare(filed_type) { // type mismatch?
			reasons = append(reasons, fmt.Sprintf("type mismatch(old:%s,new:%s)", filed_type, field.Type.string()))
			shouldChange = true
		}
		// Length mismatch (0 in model means “unspecified” so treat 1↔0 special).
		if !(field_length == 1 && field.Length == 0) && field_length != field.Length {
			reasons = append(reasons, fmt.Sprintf("length mismatch(old:%d:new:%d)", field_length, field.Length))
			shouldChange = true
		}
		if schema.defaultVal.String != field.DefaultValue { // default value mismatch?
			// some edge cases
			if field.Type != FieldTypes.Timestamp {
				reasons = append(reasons, "default mismatch")
				shouldChange = true
			}
		}
		// Nullable flag mismatches (DB says YES/NO vs model bool).
		if schema.nullable == "YES" && !field.Nullable ||
			schema.nullable == "NO" && field.Nullable {
			reasons = append(reasons, "nullable mismatch")
			shouldChange = true
		}
		// Auto‑increment mismatch.
		// log.Info("incriment Settings: %s", schema.extra)
		if (schema.extra == "auto_increment" && !field.AutoIncrement) || (schema.extra == "" && field.AutoIncrement) {
			reasons = append(reasons, "auto_increment mismatch")
			shouldChange = true
		}

		// If anything differs, optionally prompt user then patch DB.
		if shouldChange {
			if config.GetBuild() { // interactive mode
				fmt.Printf("Field '%s' requires update (%s). Proceed? (y/n): ",
					field.name, strings.Join(reasons, ", "))
				input, _ := reader.ReadString('\n')
				if strings.TrimSpace(input) != "y" {
					fmt.Printf("\n[Modify] Skipped update of: %s\n", field.name)
					goto indexCheck // skip to index comparison
				}
			}
			log.Debug("[Modify] Updating field: %s (%s)", field.name, strings.Join(reasons, ", "))
			m.modifyDBField(field) // apply column alterations
		}

	indexCheck:
		// ────────────── Index consistency checks ──────────────
		// UNIQUE
		if schema.isunique != field.Index.Unique {
			if config.GetBuild() {
				fmt.Printf("UNIQUE index mismatch on '%s'. Sync? (y/n): ", field.name)
				input, _ := reader.ReadString('\n')
				if strings.TrimSpace(input) != "y" {
					fmt.Printf("[Index] Skipped UNIQUE sync on: %s\n", field.name)
				} else {
					m.syncUniqueIndex(field, &schema)
				}
			} else {
				m.syncUniqueIndex(field, &schema)
			}
		}

		// PRIMARY KEY
		if schema.isprimary != field.Index.PrimaryKey {
			if config.GetBuild() {
				fmt.Printf("PRIMARY KEY mismatch on '%s'. Sync? (y/n): ", field.name)
				input, _ := reader.ReadString('\n')
				if strings.TrimSpace(input) != "y" {
					fmt.Printf("[Index] Skipped PRIMARY KEY sync on: %s\n", field.name)
				} else {
					m.syncPrimaryKey(field, &schema)
				}
			} else {
				m.syncPrimaryKey(field, &schema)
			}
		}

		// Regular INDEX
		if schema.isindex != field.Index.Index {
			if config.GetBuild() {
				fmt.Printf("INDEX mismatch on '%s'. Sync? (y/n): ", field.name)
				input, _ := reader.ReadString('\n')
				if strings.TrimSpace(input) != "y" {
					fmt.Printf("[Index] Skipped INDEX sync on: %s\n", field.name)
				} else {
					m.syncIndex(field, &schema)
				}
			} else {
				m.syncIndex(field, &schema)
			}
		}
	}

	// Check for fields to delete
	for _, schema := range m.schemas {
		if _, exists := FieldTypeset[schema.field]; !exists {
			if config.GetBuild() {
				fmt.Printf("Field '%s' exists in DB but not in model. Delete? (y/n): ", schema.field)
				input, err := reader.ReadString('\n')
				if err == nil && strings.TrimSpace(input) == "y" {
					m.removeDBField(schema.field)
				} else {
					fmt.Printf("[Delete] Skipped: %s\n", schema.field)
				}
			} else {
				fmt.Printf("Do you want to delete %s (y/n): ", schema.field)
				input, err := reader.ReadString('\n')
				if err == nil && strings.TrimSpace(input) == "y" {
					m.removeDBField(schema.field)
				} else {
					fmt.Printf("[Delete] Skipped: %s\n", schema.field)
				}
			}
		}
	}

	// Add all pending new fields now going to add in the table scema
	for _, field := range pendingAddFields {
		m.addField(field)
	}

}

// SyncModelSchema loads the current structure of the associated database table,
// including column definitions and index metadata (primary, unique, and standard indexes),
// and stores it in the model's internal schema list (m.schemas).
//
// This is used to detect schema differences for migration, validation, or syncing purposes.
// If the table does not exist, the function will exit early without error.
//
// Note: This function panics on database errors and should be called only when
// database availability is guaranteed.
func (m *meta) SyncModelSchema() {
	// Get the active database connection
	databaseObj, err := DatabaseHandler.GetDatabase()
	if err != nil {
		panic("Error getting database: " + err.Error())
	}

	// Check if the table for this model actually exists in the database
	checkqueryBuilder := `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`
	var count int
	err = databaseObj.QueryRow(checkqueryBuilder, m.TableName).Scan(&count)
	if err != nil {
		panic("Error checking table existence: " + err.Error())
	}
	if count == 0 {
		// If table does not exist, log and exit
		fmt.Printf("Table '%s' does not exist.\n", m.TableName)
		return
	}

	// Query the structure of the existing table
	rows, err := databaseObj.Query("SHOW COLUMNS FROM `" + m.TableName + "`")
	if err != nil {
		panic("Error getting old table structure: " + err.Error())
	}
	defer rows.Close() // Ensure result rows are closed

	// Clear any previously cached schema info
	m.schemas = nil

	// Query template to get index information for a column
	indexqueryBuilder := `
	SELECT 
	column_name, 
	index_name,
	non_unique
	FROM information_schema.statistics
	WHERE table_schema = ?
	AND table_name = ?
	AND column_name = ?`

	// Iterate through each column of the table
	for rows.Next() {
		_scema := schema{}
		// Scan each column's structure into the _scema struct
		if err := rows.Scan(&_scema.field, &_scema.fieldType, &_scema.nullable, &_scema.key, &_scema.defaultVal, &_scema.extra); err != nil {
			panic("Error scanning row: " + err.Error())
		}

		// Query the index info for this column
		if idxRows, err := databaseObj.Query(indexqueryBuilder, config.GetDatabaseConfig().Database, m.TableName, _scema.field); err != nil {
			panic("Error getting index information: " + err.Error())
		} else {
			defer idxRows.Close()

			// Process each index row
			for idxRows.Next() {
				var columnName, indexName string
				var nonUnique int
				if err := idxRows.Scan(&columnName, &indexName, &nonUnique); err != nil {
					panic("Error scanning index row: " + err.Error())
				}

				// Check if it's a primary key
				if indexName == "PRIMARY" {
					_scema.isprimary = true
				} else {
					// Determine if it's a standard index or unique constraint based on naming convention
					suffix := strings.Split(indexName, "_")
					switch suffix[0] {
					case "idx":
						_scema.isindex = true
					case "unq":
						_scema.isunique = true
					}
				}
			}
		}

		// Add the parsed schema to the model's schema list
		m.schemas = append(m.schemas, _scema)
	}
}
