package model

/*
CREATE TABLE IF NOT EXISTS employees (
	id INT AUTO_INCREMENT,
	name VARCHAR(100),
	position VARCHAR(50),
	salary DECIMAL(10, 2),
	hire_date DATE,
	PRIMARY KEY `id` (id),
	INDEX `idx_name` (name)
);
*/

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	log "github.com/vrianta/gulog"
)

// return the string of the total field expression mostly will be used for table creation
func (f *Field) string() string {
	var response string

	// ENUM support
	if f.Type == FieldTypes.Enum {
		enumValues := make([]string, len(f.Definition))
		for i, val := range f.Definition {
			enumValues[i] = "'" + fmt.Sprintf("%v", val) + "'"
		}
		response = fmt.Sprintf("%s ENUM(%s)", f.name, strings.Join(enumValues, ","))
	} else {
		// Basic type and length
		response = f.name + " " + f.Type.string()
		// if the length is greater than 0 then we are setting the length of the field
		// this is mostly used for VARCHAR, CHAR, TEXT, etc.
		if f.Length > 0 {
			response += "(" + fmt.Sprint(f.Length) + ")"
		}
	}

	// if the field is nullable then we are setting it to NULL otherwise NOT NULL
	// this is mostly used for VARCHAR, CHAR, TEXT, etc.
	if f.Nullable {
		response += " NULL "
	} else if !f.Nullable { // if the field is not nullable then we are setting it to NOT NULL
		// if the field is not nullable then we are setting it to NOT NULL
		response += " NOT NULL "
	}

	// if the defaut value is not empty setting the Default values for perticular field
	if f.DefaultValue != "" {
		switch f.Type {
		case FieldTypes.String, FieldTypes.Text:
			response += "DEFAULT '" + f.DefaultValue + "' "
		case FieldTypes.Bool:
			if f.DefaultValue == "true" || f.DefaultValue == "1" {
				response += "DEFAULT TRUE "
			} else {
				response += "DEFAULT FALSE "
			}
		default:
			response += "DEFAULT " + f.DefaultValue + " "
		}
	}

	if f.Index.PrimaryKey {
		response += ", Primary Key pk_" + f.table_name + "_" + string(f.name) + " (" + string(f.name) + ")"
	}
	if f.Index.Index {
		response += ", INDEX idx_" + f.table_name + "_" + string(f.name) + " (" + string(f.name) + ")"
	}
	if f.Index.FullText {
		response += ", FULLTEXT ftxt_" + f.table_name + "_" + string(f.name) + " (" + string(f.name) + ")"
	}
	if f.Index.Spatial {
		response += ", SPATIAL sp_" + f.table_name + "_" + string(f.name) + " (" + string(f.name) + ")"
	}
	if f.Index.Unique {
		response += ", UNIQUE unq_" + f.table_name + "_" + string(f.name) + " (" + string(f.name) + ")"
	}
	if f.fk != nil {
		response += f.foreignKeyConstraint()
	}

	return response
}

func (f *Field) columnDefinition() string {
	var response string

	// ENUM support
	if f.Type == FieldTypes.Enum {

		enumValues := make([]string, len(f.Definition))
		for i, val := range f.Definition {
			enumValues[i] = "'" + fmt.Sprintf("%v", val) + "'"
		}
		response = fmt.Sprintf("%s ENUM(%s)", f.name, strings.Join(enumValues, ","))
	} else {
		response = f.name + " " + f.Type.string()

		// if the length is greater than 0 then we are setting the length of the field
		// this is mostly used for VARCHAR, CHAR, TEXT, etc.
		if f.Length > 0 {
			response += "(" + fmt.Sprint(f.Length) + ")"
		}
	}

	// if f.Length > 0 {
	// 	response += "(" + fmt.Sprint(f.Length) + ") "
	// }

	if f.Nullable {
		response += " NULL "
	} else if !f.Nullable {
		response += " NOT NULL "
	}

	if f.DefaultValue != "" {
		switch f.Type {
		case FieldTypes.String, FieldTypes.Text:
			response += "DEFAULT '" + f.DefaultValue + "' "
		case FieldTypes.Bool:
			if f.DefaultValue == "true" || f.DefaultValue == "1" {
				response += "DEFAULT TRUE "
			} else {
				response += "DEFAULT FALSE "
			}
		case FieldTypes.Enum:
			response += "DEFAULT '" + f.DefaultValue + "' "
		default:
			response += "DEFAULT " + f.DefaultValue + " "
		}
	}

	// AUTO_INCREMENT support
	if f.AutoIncrement {
		response += "AUTO_INCREMENT "
	}

	return strings.TrimSpace(response)
}

// Return -> Index Statements suitable for creating the table
func (f *Field) createIndexStatement() string {
	responseArray := []string{}
	if f.Index.PrimaryKey {
		responseArray = append(responseArray, "PRIMARY KEY pk_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")
	}
	if f.Index.Index {
		responseArray = append(responseArray, "INDEX idx_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")
	}
	if f.Index.FullText {
		responseArray = append(responseArray, "FULLTEXT ftxt_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")

	}
	if f.Index.Spatial {
		responseArray = append(responseArray, "SPATIAL sp_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")
	}
	if f.Index.Unique {
		responseArray = append(responseArray, "UNIQUE unq_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")
	}

	if f.fk != nil {
		responseArray = append(responseArray, f.foreignKeyConstraint())
	}

	if len(responseArray) > 0 {
		return strings.Join(responseArray, ",\n")
	}

	return ""
}

// Return -> Index Statements suitable for altering the table
func (f *Field) addIndexStatement() string {
	responseArray := []string{}
	if f.Index.PrimaryKey {
		responseArray = append(responseArray, "PRIMARY KEY pk_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")
	}
	if f.Index.Index {
		responseArray = append(responseArray, "INDEX idx_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")
	}
	if f.Index.FullText {
		responseArray = append(responseArray, "FULLTEXT ftxt_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")

	}
	if f.Index.Spatial {
		responseArray = append(responseArray, "SPATIAL sp_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")
	}
	if f.Index.Unique {
		responseArray = append(responseArray, "UNIQUE unq_"+f.table_name+"_"+string(f.name)+" ("+string(f.name)+")")
	}

	if f.fk != nil {
		responseArray = append(responseArray, f.foreignKeyConstraint())
	}

	if len(responseArray) > 0 {
		return ", ADD " + strings.Join(responseArray, ", ADD \n")
	}

	return ""
}

func (f *Field) Name() string {
	return f.name
}

// check if the value is compatible with the field type
func (ft fieldType) IsValueCompatible(val string) bool {
	switch ft {
	case FieldTypes.Int, FieldTypes.TinyInt, FieldTypes.SmallInt, FieldTypes.MediumInt, FieldTypes.BigInt:
		_, err := strconv.Atoi(val)
		return err == nil
	case FieldTypes.Float, FieldTypes.Decimal, FieldTypes.Double, FieldTypes.Real:
		_, err := strconv.ParseFloat(val, 64)
		return err == nil
	case FieldTypes.Date, FieldTypes.Time:
		_, err1 := time.Parse("2006-01-02", val)
		_, err2 := time.Parse("2006-01-02 15:04:05", val)
		return err1 == nil || err2 == nil
	case FieldTypes.Timestamp:
		_, err := time.Parse("2006-01-02 15:04:05", val)
		return err == nil || val == "CURRENT_TIMESTAMP" || val == "NOW()"
	case FieldTypes.JSON:
		var js json.RawMessage
		return json.Unmarshal([]byte(val), &js) == nil
	case FieldTypes.VarChar, FieldTypes.Text, FieldTypes.String, FieldTypes.Char:
		return true
	case FieldTypes.Bool:
		return val == "0" || val == "1" || val == "true" || val == "false"
	default:
		return true
	}
}

func isAlphaNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func (f *Field) Compare(fieldTypeStr string) bool {

	// edge case for enum if is returning then it will return all the default values also have to make a condition to split functions
	_type := strings.Split(fieldTypeStr, "(")[0] // so if any type return TYPE() then it will return TYPE

	switch _type {
	case "TINYINT":
		switch f.Type {
		case FieldTypes.TinyInt, FieldTypes.Bool:
			return true
		}
	case "SMALLINT":
		switch f.Type {
		case FieldTypes.SmallInt:
			return true
		}
	case "MEDIUMINT":
		switch f.Type {
		case FieldTypes.MediumInt:
			return true
		}
	case "INT", "INTEGER":
		switch f.Type {
		case FieldTypes.Int:
			return true
		}
	case "BIGINT":
		switch f.Type {
		case FieldTypes.BigInt:
			return true
		}
	case "VARCHAR":
		switch f.Type {
		case FieldTypes.VarChar, FieldTypes.String:
			return true
		}
	case "CHAR":
		switch f.Type {
		case FieldTypes.Char, FieldTypes.String:
			return true
		}
	case "TEXT":
		switch f.Type {
		case FieldTypes.Text, FieldTypes.String:
			return true
		}
	case "LONGTEXT":
		switch f.Type {
		case FieldTypes.LongText, FieldTypes.JSON:
			return true
		}
	case "BOOL", "BOOLEAN":
		switch f.Type {
		case FieldTypes.Bool, FieldTypes.TinyInt:
			return true
		}
	case "DECIMAL":
		switch f.Type {
		case FieldTypes.Decimal:
			return true
		}
	case "FLOAT":
		switch f.Type {
		case FieldTypes.Float:
			return true
		}
	case "DOUBLE", "REAL":
		switch f.Type {
		case FieldTypes.Double, FieldTypes.Real:
			return true
		}
	case "JSON":
		switch f.Type {
		case FieldTypes.JSON:
			return true
		}
	case "BLOB":
		switch f.Type {
		case FieldTypes.Blob:
			return true
		}
	case "DATE":
		switch f.Type {
		case FieldTypes.Date:
			return true
		}
	case "TIME":
		switch f.Type {
		case FieldTypes.Time:
			return true
		}
	case "TIMESTAMP":
		switch f.Type {
		case FieldTypes.Timestamp:
			return true
		}
	case "ENUM":
		// Enum is coming as a functions with values - ENUM('MALE','FEMALE','EXTRA','OTHER')
		enumValues := make([]string, len(f.Definition))
		for i, val := range f.Definition {
			enumValues[i] = "'" + strings.ToUpper(fmt.Sprintf("%v", val)+"'")
		}
		enum := fmt.Sprintf("ENUM(%s)", strings.Join(enumValues, ","))
		// println(enum)
		if fieldTypeStr == enum {
			return true
		}
	case "UUID":
		switch f.Type {
		case FieldTypes.UUID:
			return true
		}
	default:
		return false
	}

	return false
}

func (f *Field) foreignKeyConstraint() string {
	if f.fk == nil {
		return ""
	}

	stmt := fmt.Sprintf("CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
		f.table_name, f.name, f.name, f.fk.referenceTable, f.fk.referenceColumn)

	if f.fk.onDelete != "" {
		stmt += " ON DELETE " + f.fk.onDelete
	}
	if f.fk.onUpdate != "" {
		stmt += " ON UPDATE " + f.fk.onUpdate
	}

	return stmt
}

/*
 * ToForeignKey converts a field to a foreign key with the given onDelete and onUpdate actions.
 * It requires the field to be a primary key and sets the foreign key metadata.
 * It also sets the index and unique properties based on the parameters.
 * @parameters:
 * onDelete, onUpdate - actions for foreign key constraints
 * is_primary_key - whether the field is a primary key
 * is_index - whether the field should be indexed
 * is_unique - whether the field should be unique
 * @returns: a new Field with foreign key constraints set
 */
func (f *Field) ToForeignKey(onDelete, onUpdate string, is_primary_key, is_index, is_unique bool) *Field {

	if !f.Index.PrimaryKey {
		panic("Field " + f.name + " must be a primary key to set as foreign key")
	}

	// TODO: Find a way to get the table name before the field is created
	log.Info("Table Name of the foreingkey: %s", f.table_name)
	return &Field{
		name:          f.name,
		Type:          f.Type,
		Length:        f.Length,
		Nullable:      f.Nullable,
		DefaultValue:  f.DefaultValue,
		AutoIncrement: f.AutoIncrement,
		Index: Index{
			PrimaryKey: is_primary_key,
			Index:      is_index,
			Unique:     is_unique,
		},
		table_name: f.table_name,
		fk: &foreignKey{
			referenceTable:  f.table_name,
			referenceColumn: f.name,
			onDelete:        onDelete,
			onUpdate:        onUpdate,
		},
	}
}

// ForeignKey sets the foreign key constraint for a field
// leave onDelete and onUpdate empty if you do not want to set them
// example onDelete: "CASCADE", "SET NULL", "RESTRICT", etc.
// example onUpdate: "CASCADE", "SET NULL", "RESTRICT", etc.
func ForeignKey(field Field, onDelete, onUpdate string, is_primary_key bool, is_index bool, is_unique bool) Field {
	field.Index.PrimaryKey = is_primary_key
	field.Index.Index = is_index
	field.Index.Unique = is_unique

	field.fk = &foreignKey{
		referenceTable:  field.table_name,
		referenceColumn: field.name,
		onDelete:        onDelete,
		onUpdate:        onUpdate,
	}
	return field
}
