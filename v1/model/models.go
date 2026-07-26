package model

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/vrianta/agai/v1/config"
	"github.com/vrianta/agai/v1/database"
	"github.com/vrianta/agai/v1/internal/flags"
	log "github.com/vrianta/gulog"
)

func Init() {

	if !database.Initialized {
		if config.GetDatabaseConfig().Host == "" {
			fmt.Printf("[Warning] Database not initialized, skipping table creation/modification\n")
			return
		} else {
			panic("[Models] Error: Connecting the Database, please check the database configuration.\n")
		}
	}

	if initialsed {
		fmt.Println("[Warning] - Models are already initialsed Skipping it")
	}

	create_model := func(model *meta) {
		if model == nil {
			// means the table is already deleted
			return
		}
		model.CreateTableIfNotExists()

		if flags.SyncDatabaseEnabled {
			log.Info("[Models] Initializing model and syncing database tables for: %s", model.TableName)
			model.SyncModelSchema()
			model.syncTableSchema()

			model.initialised = true
		}
		delete(ModelsRegistry, model.TableName)
	}

	model_for_component := maps.Clone(ModelsRegistry)
	for _, model := range ModelsRegistry {
		for _, depends_on := range model.depends_on {
			create_model(ModelsRegistry[depends_on])
		}
		model.depends_on = nil // clear depends_on after processing
		create_model(model)
		delete(ModelsRegistry, model.TableName)
	}

	fmt.Println("---------------------------------------------------------")

	for _, model := range model_for_component {

		_, err := os.Stat(filepath.Join(componentsDir, model.TableName+".component.json"))
		if !os.IsNotExist(err) {
			model.loadComponentFromDisk()
			if flags.SyncComponentsEnabled {
				model.syncComponentWithDB()
				model.loadComponentFromDisk()
			} else {
				// means the file exists in the disk
				model.refreshComponentFromDB()
			}
		}
	}
	initialsed = true
}

/*
 * This Package is to handle model in the database checking and creating tables and providing default functions to handle them
 * It will create the table,
 * It will update the table accordingly during the initial program startup only if the build is not true
 * So Dynaimic Table Updation will be handled during development only
 * It will provide the default functions to handle the model like Create, Read, Update, Delete
 */
func newModel(tableName string, FieldTypes FieldTypeset, depends_on []string) meta {

	for _, field := range FieldTypes {
		if field.fk != nil {
			depends_on = append(depends_on, field.fk.referenceTable)
		}
		field.table_name = tableName // Set the table name for each field
		// }
	}

	_model := meta{
		components: make(components),
		TableName:  tableName,
		FieldTypes: FieldTypes,
		primary: func(FieldTypes FieldTypeset) *Field {
			for _, field := range FieldTypes {
				if field.Index.PrimaryKey {
					return field // Return the pointer directly from the map
				}
			}
			return nil
		}(FieldTypes),
		depends_on: depends_on,
	}

	_model.validate()

	return _model
}

func New[T any](tableName string, structure T) *Table[T] {
	t := reflect.TypeOf(structure)
	v := reflect.ValueOf(structure)

	if t.Kind() != reflect.Struct {
		panic("structure passed to New must be a struct")
	}

	FieldTypeset := make(FieldTypeset, t.NumField())
	depends_on := []string{}
	for i := 0; i < t.NumField(); i++ {
		structField := t.Field(i)
		valueField := v.Field(i)

		// Handle pointer to Field
		fieldPtr, ok := valueField.Interface().(*Field)
		if !ok {
			panic(fmt.Sprintf("[Model Error] Field '%s' is not of type *model.Field of Model %s", structField.Name, tableName))
		}
		if fieldPtr == nil {
			panic(fmt.Sprintf("[Validation Error] Field '%s' in Talble %s Body is not Defined", structField.Name, tableName))
		}
		// Update metadata
		fieldPtr.name = structField.Name
		if fieldPtr.fk == nil {
			fieldPtr.table_name = tableName
		}

		FieldTypeset[structField.Name] = fieldPtr
	}

	response := &Table[T]{
		meta:   newModel(tableName, FieldTypeset, depends_on),
		Fields: structure,
	}

	ModelsRegistry[tableName] = &response.meta
	return response
}

func (m *meta) CreateTableIfNotExists() {
	sql := "CREATE TABLE IF NOT EXISTS " + m.TableName + " (\n"
	fieldDefs := []string{}

	for _, field := range m.FieldTypes {
		fieldDefs = append(fieldDefs, field.columnDefinition())
	}

	for _, field := range m.FieldTypes {
		indexStatements := field.createIndexStatement()
		if indexStatements != "" {
			fieldDefs = append(fieldDefs, indexStatements)
		}
	}

	sql += strings.Join(fieldDefs, ",\n")
	sql += "\n);"

	databaseObj, err := database.GetDatabase()
	if err != nil {
		panic("Error getting database: " + err.Error())
	}

	_, err = databaseObj.Exec(sql)
	// log.Info("Creating Table Sql Executed : %s", sql)
	if err != nil {
		panic("Error creating table: " + err.Error() + "\nqueryBuilder:" + sql)
	}

}

// Handles adding/dropping PRIMARY KEY
func (m *meta) syncPrimaryKey(field *Field, schema *schema) {
	databaseObj, err := database.GetDatabase()
	if err != nil {
		fmt.Println("Error updating primary key:", err)
		return
	}
	if schema.isprimary && !field.Index.PrimaryKey {
		// Drop primary key
		queryBuilder := fmt.Sprintf("ALTER TABLE `%s` DROP PRIMARY KEY;", m.TableName)
		if _, err := databaseObj.Exec(queryBuilder); err != nil {
			fmt.Println("[Index] Error dropping PRIMARY KEY:", err)
		} else {
			fmt.Printf("[Index] PRIMARY KEY dropped for field: %s\n", field.name)
		}
	}
	if !schema.isprimary && field.Index.PrimaryKey {
		// Add primary key
		queryBuilder := "ALTER TABLE " + m.TableName + " ADD PRIMARY KEY (" + field.name + ")"
		if _, err := databaseObj.Query(queryBuilder); err != nil {
			fmt.Println("[ERROR] failed to Add Primary Key ", err.Error())
			fmt.Println("[FAILED] Failed queryBuilder to Update Primary Key is: ", queryBuilder)
		}
	}
}

func logSection(header string) {
	fmt.Println("---------------------------------------------------------")
	fmt.Println(header)
	fmt.Println("---------------------------------------------------------")
}

// Handles adding/dropping UNIQUE index
func (m *meta) syncUniqueIndex(field *Field, schema *schema) {
	databaseObj, err := database.GetDatabase()
	if err != nil {
		fmt.Println("Error updating unique index:", err)
		return
	}
	indexName := fmt.Sprintf("unq_%s_%s", field.table_name, field.name)
	if schema.isunique && !field.Index.Unique {
		// Drop unique index
		queryBuilder := fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`;", m.TableName, indexName)
		if _, err := databaseObj.Exec(queryBuilder); err != nil {
			fmt.Println("[Index] Error dropping UNIQUE:", err)
		} else {
			fmt.Printf("[Index] UNIQUE dropped for field: %s\n", field.name)
		}
	}
	if !schema.isunique && field.Index.Unique {
		// Add unique index
		queryBuilder := fmt.Sprintf("ALTER TABLE `%s` ADD UNIQUE `%s` (`%s`);", m.TableName, indexName, field.name)
		if _, err := databaseObj.Exec(queryBuilder); err != nil {
			fmt.Println("[Index] Error adding UNIQUE:", err)
		} else {
			fmt.Printf("[Index] UNIQUE added for field: %s\n", field.name)
		}
	}
}

// Handles adding/dropping normal INDEX
func (m *meta) syncIndex(field *Field, schema *schema) {
	databaseObj, err := database.GetDatabase()
	if err != nil {
		fmt.Println("Error updating index:", err)
		return
	}
	indexName := fmt.Sprintf("idx_%s_%s", m.TableName, field.name)
	if schema.isindex && !field.Index.Index {
		// Drop index
		queryBuilder := fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`;", m.TableName, indexName)
		if _, err := databaseObj.Exec(queryBuilder); err != nil {
			fmt.Println("[Index] Error dropping INDEX:", err)
		} else {
			fmt.Printf("[Index] INDEX dropped for field: %s\n", field.name)
		}
	}
	if !schema.isindex && field.Index.Index {
		// Add index
		queryBuilder := fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `%s` (`%s`);", m.TableName, indexName, field.name)
		if _, err := databaseObj.Exec(queryBuilder); err != nil {
			fmt.Println("[Index] Error adding INDEX:", err)
		} else {
			fmt.Printf("[Index] INDEX added for field: %s\n", field.name)
		}
	}
}

// get the table name
func (m *meta) GetTableName() string {
	return m.TableName
}

// Convert the Fetched Data to a of objects
// This function will convert the Table to a map[string]any for easy access and manipulation
// func (m *Table) ToMap() map[string]any {
// 	response := make(map[string]any, len(m.FieldTypes))
// 	var wg sync.WaitGroup
// 	var mu sync.Mutex

// 	for _, field := range m.FieldTypes {
// 		wg.Add(1)

// 		go func(f *Field) {
// 			defer wg.Done()

// 			var value any
// 			if f.value != nil {
// 				value = f.value
// 			} else {
// 				value = nil
// 			}

// 			mu.Lock()
// 			response[f.Name] = value
// 			mu.Unlock()
// 		}(&field)
// 	}

// 	wg.Wait()
// 	return response
// }

// InsertRow InsertRows a new record into the table using the provided values map.
// This is a dedicated Create/InsertRow function that does not overlap with table creation or schema management.
func (m *meta) InsertRow(values map[string]any) error {
	q := m.Create()
	for k, v := range values {
		q.InsertRowFieldTypes[k] = v
	}
	return q.Exec()
}

func (m *meta) GetPrimaryKey() *Field {
	if !m.HasPrimaryKey() {
		panic("Primary Key is Required for but the Model(" + m.TableName + ") ")
	}
	return m.primary
}

/*
To check if the model has primary key or not

true ->  if exists
false -> if not exists
*/
func (m *meta) HasPrimaryKey() bool {
	if m.primary != nil {
		return true
	}
	for _, field := range m.FieldTypes {
		if field.Index.PrimaryKey {
			m.primary = field
			return true // Return the pointer directly from the map
		}
	}

	return false
}

/*
GetField(fieldname) -> return pointer of the field
*/

func (m *meta) GetField(field_name string) *Field {
	field, ok := m.FieldTypes[field_name]
	if !ok {
		return nil
	}
	return field
}

/*
GetField(fieldname) -> return pointer of the field
*/

func (m *meta) GetFieldTypes() *FieldTypeset {
	return &m.FieldTypes
}

// Print the Objects of the models as the good for debug perpose
func (r *Results) PrintAsTable() {
	if len(*r) == 0 {
		return
	}

	// Collect all unique column names across all rows
	colSet := map[string]struct{}{}
	for _, row := range *r {
		for col := range row {
			colSet[col] = struct{}{}
		}
	}

	// Sort column names for consistent display
	var colNames []string
	for col := range colSet {
		colNames = append(colNames, col)
	}
	sort.Strings(colNames)

	// Print header
	for _, col := range colNames {
		fmt.Printf("| %-15s", col)
	}
	fmt.Println("|")

	// Print separator
	fmt.Println(strings.Repeat("-", len(colNames)*18))

	// Print each row
	for _, row := range *r {
		for _, col := range colNames {
			val := row[col]
			fmt.Printf("| %-15v", val)
		}
		fmt.Println("|")
	}
}
