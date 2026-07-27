package model

import (
	"fmt"
	"strings"

	DatabaseHandler "github.com/vrianta/agai/v1/database"
)

// ===============================
// queryBuilder Builder for ModelsHandler
// ===============================
//
// This file provides a human-friendly, chainable queryBuilder builder for working with database tables using Go structs.
// It is designed to make database operations (SELECT, UPDATE, DELETE) easy to read, write, and maintain.
//
// The builder mimics the style of popular Object-Relational Mappers (ORMs), allowing you to construct queries
// in a fluent, readable way. This means you can chain methods together to build up complex queries step by step.
//
// Example usage:
//
//   users := UserModel.Get().Where("age").GreaterThan(18).OrderBy("name").Fetch()
//
// Supported features include:
//   - WHERE, AND, OR conditions
//   - LIMIT, OFFSET for pagination
//   - GROUP BY, ORDER BY for sorting and grouping
//   - Comparison operators (>, <, =, !=, IN, NOT IN, BETWEEN, LIKE, IS NULL, etc.)
//   - UPDATE and DELETE operations
//
// Each function is documented in detail below. If you are new to this code, read the comments for each function
// to understand what it does and how to use it. The goal is to make database access as intuitive as possible.
//
// If you are not a Go developer, don't worry! The comments explain the logic in plain English.

// Returning all results matching the queryBuilder as a map of primary key values to row data (as Result).
func (m *meta) All() (Results, error) {
	return m.Get().Fetch()
}

// Get upto Certain index of results matching the queryBuilder as a map of primary key values to row data (as Result).
func (m *meta) GetUpto(limit int64) (Results, error) {
	return m.Get().Limit(limit).Fetch()
}

// Get Between certain index of results matching the queryBuilder as a map of primary key values to row data (as Result).
func (m *meta) GetBetween(offset, limit int64) (Results, error) {
	return m.Get().Offset(offset).Limit(limit).Fetch()
}

// Entry point: create a new queryBuilder for the given model struct.
// This function starts a new queryBuilder chain. By default, it prepares for a SELECT operation.
// Example: UserModel.Get() returns a queryBuilder object you can chain more methods onto.
func (m *meta) Get() *queryBuilder {
	return &queryBuilder{
		model:     m,        // The model (table) this queryBuilder is for
		operation: "select", // Default operation is SELECT
	}
}

// Refactored: Table.Create now returns an InsertRowBuilder for InsertRow operations.
func (m *meta) Create() *InsertRowBuilder {
	return &InsertRowBuilder{
		model:               m,
		InsertRowFieldTypes: make(map[string]any),
	}
}

func (m *meta) Update(f *Field) *queryBuilder {
	q := &queryBuilder{
		model:     m,
		operation: "update",
	}

	if f != nil {
		q.Set(f)
	}

	return q
}

// =======================
// DELETE queryBuilder Function
// =======================

// Delete executes a DELETE queryBuilder using the built WHERE and LIMIT clauses.
// It removes matching rows from the database table.
// Prints the number of affected rows for debugging.
// Delete deletes rows matching the queryBuilder from the table.
// Delete starts a DELETE queryBuilder chain.
// Usage: UserModel.Delete().Where("id").Is(5).Exec()
func (m *meta) Delete() *queryBuilder {
	return &queryBuilder{
		model:     m,
		operation: "delete",
	}
}

// =======================
// WHERE Clause Functions
// =======================

// Where begins a WHERE clause, specifying the column to filter on.
// Example: .Where("age")
func (q *queryBuilder) Where(f *Field) *queryBuilder {
	q.lastColumn = f.name // Remember which column the next condition is for
	return q
}

// Is adds an equality condition to the WHERE clause.
// Example: .Where("age").Is(30)  // WHERE age = 30
func (q *queryBuilder) Is(value any) *queryBuilder {
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` = ?", q.lastColumn)) // Add an equality condition for the last column
	q.whereArgs = append(q.whereArgs, value)                                       // Add the value to the arguments for the queryBuilder
	q.lastColumn = ""                                                              // Reset lastColumn for safety
	return q                                                                       // Return the queryBuilder object for chaining
}

// IsNot adds a NOT EQUAL condition (`!=`) to the WHERE clause for the previously specified column.
// It is used after calling .Where("columnName").
//
// Example:
//
//	queryBuilder.Where("status").IsNot("inactive")
//
// Generates:
//
//	WHERE `status` != 'inactive'
func (q *queryBuilder) IsNot(value any) *queryBuilder {
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` != ?", q.lastColumn))
	q.whereArgs = append(q.whereArgs, value)
	q.lastColumn = ""
	return q
}

// Like adds a LIKE condition to the WHERE clause for SQL pattern matching (e.g., for wildcards like `%value%`).
// It is used after calling .Where("columnName").
//
// Example:
//
//	queryBuilder.Where("username").Like("%pritam%")
//
// Generates:
//
//	WHERE `username` LIKE '%pritam%'
func (q *queryBuilder) Like(value string) *queryBuilder {
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` LIKE ?", q.lastColumn))
	q.whereArgs = append(q.whereArgs, value)
	q.lastColumn = ""
	return q
}

// And appends a logical AND operator between WHERE conditions.
// It should be used between chained .Where() clauses.
//
// Example:
//
//	queryBuilder.Where("role").Is("admin").And().Where("active").Is(true)
//
// Generates:
//
//	WHERE `role` = 'admin' AND `active` = true
func (q *queryBuilder) And() *queryBuilder {
	q.whereClauses = append(q.whereClauses, "AND")
	return q
}

// Or appends a logical OR operator between WHERE conditions.
// It should be used between chained .Where() clauses.
//
// Example:
//
//	queryBuilder.Where("role").Is("admin").Or().Where("role").Is("moderator")
//
// Generates:
//
//	WHERE `role` = 'admin' OR `role` = 'moderator'
func (q *queryBuilder) Or() *queryBuilder {
	q.whereClauses = append(q.whereClauses, "OR")
	return q
}

// In adds an IN condition to the WHERE clause for checking if a column's value exists in a set of values.
// It is used after .Where("columnName") and accepts a variadic list of values.
//
// Example:
//
//	queryBuilder.Where("userId").In(1, 2, 3)
//
// Generates:
//
//	WHERE `userId` IN (1, 2, 3)
//
// Note: The values passed are safely parameterized using `?` placeholders to prevent SQL injection.
func (q *queryBuilder) In(values ...any) *queryBuilder {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(values)), ",")
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` IN (%s)", q.lastColumn, placeholders))
	q.whereArgs = append(q.whereArgs, values...)
	q.lastColumn = ""
	return q
}

// NotIn adds a NOT IN condition to the WHERE clause for excluding values.
// Usage: .Where("status").NotIn("inactive", "banned")
func (q *queryBuilder) NotIn(values ...any) *queryBuilder {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(values)), ",")
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` NOT IN (%s)", q.lastColumn, placeholders))
	q.whereArgs = append(q.whereArgs, values...)
	q.lastColumn = ""
	return q
}

// GreaterThan adds a "greater than" condition to the WHERE clause.
// Usage: .Where("score").GreaterThan(100)
func (q *queryBuilder) GreaterThan(value any) *queryBuilder {
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` > ?", q.lastColumn))
	q.whereArgs = append(q.whereArgs, value)
	q.lastColumn = ""
	return q
}

// LessThan adds a "less than" condition to the WHERE clause.
// Usage: .Where("score").LessThan(50)
func (q *queryBuilder) LessThan(value any) *queryBuilder {
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` < ?", q.lastColumn))
	q.whereArgs = append(q.whereArgs, value)
	q.lastColumn = ""
	return q
}

// Between adds a BETWEEN condition to the WHERE clause for a range.
// Usage: .Where("created_at").Between(start, end)
func (q *queryBuilder) Between(min, max any) *queryBuilder {
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` BETWEEN ? AND ?", q.lastColumn))
	q.whereArgs = append(q.whereArgs, min, max)
	q.lastColumn = ""
	return q
}

// IsNull adds an IS NULL condition to the WHERE clause.
// Usage: .Where("deleted_at").IsNull()
func (q *queryBuilder) IsNull() *queryBuilder {
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` IS NULL ", q.lastColumn))
	q.lastColumn = ""
	return q
}

// IsNotNull adds an IS NOT NULL condition to the WHERE clause.
// Usage: .Where("deleted_at").IsNotNull()
func (q *queryBuilder) IsNotNull() *queryBuilder {
	q.whereClauses = append(q.whereClauses, fmt.Sprintf("`%s` IS NOT NULL ", q.lastColumn))
	q.lastColumn = ""
	return q
}

// =======================
// UPDATE queryBuilder Functions
// =======================

// Set marks the start of an UPDATE operation, specifying which field to update.
// Call this before .To().
// Example: .Set("name")
func (q *queryBuilder) Set(field *Field) *queryBuilder {
	if field == nil {
		panic("Field can not be nil or empty while setting it")
	}
	q.lastSet = field.name
	if q.operation == "" {
		q.operation = "update" // default fallback
	}
	return q
}

func (q *queryBuilder) SetWithFieldName(field string) *queryBuilder {
	q.lastSet = field
	if q.operation == "" {
		q.operation = "update" // default fallback
	}
	return q
}

// To specifies the value to set for the previously specified field in an UPDATE.
// Example: .Set("name").To("Alice")
func (q *queryBuilder) To(value any) *queryBuilder {
	switch q.operation {
	case "update":
		q.setClauses = append(q.setClauses, fmt.Sprintf("`%s` = ?", q.lastSet))
		q.setArgs = append(q.setArgs, value)
	case "InsertRow":
		q.InsertRowFieldTypes[q.lastSet] = value
	}
	q.lastSet = ""
	return q
}

// Set marks the start of an InsertRow operation, specifying which field to InsertRow.
func (q *InsertRowBuilder) Set(field *Field) *InsertRowBuilder {
	q.lastSet = field.name
	return q
}

// To specifies the value to set for the previously specified field in an InsertRow.
// Example: .Set("name").To("Alice")
func (q *InsertRowBuilder) To(value any) *InsertRowBuilder {
	if q.lastSet != "" {
		q.InsertRowFieldTypes[q.lastSet] = value
		q.lastSet = ""
	}
	return q
}

// =======================
// SELECT queryBuilder Functions
// =======================

// Limit restricts the number of results returned by the queryBuilder.
// Example: .Limit(10)
func (q *queryBuilder) Limit(n int64) *queryBuilder {
	q.limit = n // Store the limit for later
	return q
}

// Fetch executes the built SELECT queryBuilder and returns all matching rows as a slice of meta pointers.
//
// ---
// LAYMAN'S EXPLANATION:
//
// Fetch is like asking the database: "Give me all the rows that match my conditions."
// It builds a SELECT SQL queryBuilder using the filters (WHERE), limits (LIMIT), and other options you set up by chaining methods.
//
// 1. It gets a database connection.
// 2. It builds the WHERE and LIMIT parts of the SQL queryBuilder.
// 3. It creates the full SQL queryBuilder string (e.g., SELECT * FROM users WHERE age > 18 LIMIT 10).
// 4. It runs this queryBuilder on the database and gets back the rows.
// 5. For each row:
//   - It creates a new meta (like a Go object for a row).
//   - It copies the model's field definitions.
//   - It fills in the values from the database into the meta FieldTypes.
//   - It adds this meta to the results list.
//
// 6. It returns the list of Structs (rows) and any error.
//
// Key variables:
//
//	db: database connection
//	where, limit: SQL WHERE and LIMIT parts
//	queryBuilder: the SQL queryBuilder string
//	rows: the result set from the database
//	columns: column names in the result
//	results: the list of Structs to return
func (q *queryBuilder) Fetch() (Results, error) {
	db, err := DatabaseHandler.GetDatabase()
	if err != nil {
		return nil, err
	}

	where := q.buildWhere()
	limit := q.buildLimit()
	offset := q.buildOffset()

	order := ""
	if q.orderBy != "" {
		order = "ORDER BY " + q.orderBy
	}
	group := ""
	if q.groupBy != "" {
		group = "GROUP BY " + q.groupBy
	}

	clauses := []string{}
	if where != "" {
		clauses = append(clauses, where)
	}
	if group != "" {
		clauses = append(clauses, group)
	}
	if order != "" {
		clauses = append(clauses, order)
	}
	if limit != "" {
		clauses = append(clauses, limit)
	}
	if offset != "" {
		clauses = append(clauses, offset)
	}

	queryBuilder := fmt.Sprintf("SELECT * FROM `%s` %s", q.model.TableName, strings.TrimSpace(strings.Join(clauses, " ")))
	if strings.TrimSpace(strings.Join(clauses, " ")) == "" {
		queryBuilder = fmt.Sprintf("SELECT * FROM `%s`", q.model.TableName)
	}

	rows, err := db.Query(queryBuilder, q.whereArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make(Results)

	for rows.Next() {
		pointers := make([]any, len(columns))
		holders := make([]any, len(columns))
		for i := range columns {
			pointers[i] = &holders[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		row := make(Result)
		for i, col := range columns {
			val := holders[i]
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			row[col] = val
		}

		// Extract the primary key value from the row
		primary := q.model.GetPrimaryKey()
		primaryVal := row[primary.name]
		results[primaryVal] = row
	}

	return results, rows.Err()
}

// First executes the built SELECT queryBuilder and returns only the first matching row (or nil if none).
//
// ---
// LAYMAN'S EXPLANATION:
//
// First is a shortcut for "just give me the first row that matches my queryBuilder."
//
// 1. If you didn't set a limit, it sets the limit to 1 (so only one row is fetched).
// 2. It calls Fetch to get the results.
// 3. If there are no results, it returns nil.
// 4. If there is at least one result, it returns the first one.
//
// Key variables:
//
//	rows: the list of results from Fetch
//	q.limit: the maximum number of results to get (set to 1 here)
func (q *queryBuilder) First() (Result, error) {
	if q.limit == 0 {
		q.limit = 1
	}
	resMap, err := q.Fetch()
	if err != nil {
		return nil, err
	}
	for _, row := range resMap {
		return row, nil // Return first row encountered
	}
	return nil, nil
}

// Count executes the built count query and returns the number of matching rows.
// Usage: UserModel.Get().Where(...).Is(...).Count()
func (q *queryBuilder) Count() (int64, error) {
	db, err := DatabaseHandler.GetDatabase()
	if err != nil {
		return 0, err
	}

	where := q.buildWhere()
	group := ""
	if q.groupBy != "" {
		group = "GROUP BY " + q.groupBy
	}

	clauses := []string{}
	if where != "" {
		clauses = append(clauses, where)
	}
	if group != "" {
		clauses = append(clauses, group)
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s` %s", q.model.TableName, strings.TrimSpace(strings.Join(clauses, " ")))
	if strings.TrimSpace(strings.Join(clauses, " ")) == "" {
		query = fmt.Sprintf("SELECT COUNT(*) FROM `%s`", q.model.TableName)
	}

	row := db.QueryRow(query, q.whereArgs...)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// =======================
// UPDATE queryBuilder Execution
// =======================

// Exec executes an UPDATE queryBuilder using the built SET and WHERE clauses.
// Only works if the operation is set to "update" (via Set).
// Prints the number of affected rows for debugging.
//
// ---
// LAYMAN'S EXPLANATION:
//
// Exec is used to update rows in the database. It's like saying: "Change these FieldTypes for all rows that match my conditions."
//
// 1. It checks if you're actually doing an update (not a select or delete).
// 2. It gets a database connection.
// 3. It checks if you specified any FieldTypes to update. If not, it returns an error.
// 4. It builds the SET part (FieldTypes and new values) and the WHERE part (which rows to update).
// 5. It creates the SQL UPDATE queryBuilder (e.g., UPDATE users SET name = 'Alice' WHERE id = 1).
// 6. It combines all the values for the SET and WHERE clauses.
// 7. It runs the update on the database.
// 8. It prints how many rows were updated (for debugging).
// 9. It returns any error that happened.
//
// Key variables:
//
//	db: database connection
//	set: SET clause (FieldTypes and new values)
//	where: WHERE clause (which rows to update)
//	queryBuilder: the SQL update statement
//	args: all the values to use in the queryBuilder
//	result: the result of running the update
func (q *queryBuilder) Exec() error {
	db, err := DatabaseHandler.GetDatabase()
	if err != nil {
		return err
	}

	switch q.operation {
	case "update":
		if len(q.setClauses) == 0 {
			return fmt.Errorf("update failed: no FieldTypes to update")
		}

		where := q.buildWhere()
		if where == "" {
			return fmt.Errorf("unsafe update: WHERE clause is required")
		}

		queryBuilder := fmt.Sprintf(
			"UPDATE `%s` SET %s %s",
			q.model.TableName,
			strings.Join(q.setClauses, ", "),
			where,
		)

		args := append(q.setArgs, q.whereArgs...)

		result, err := db.Exec(queryBuilder, args...)
		if err != nil {
			fmt.Printf("[Update Error] queryBuilder: %s | Error: %v\n", queryBuilder, err)
			return err
		}

		if affected, err := result.RowsAffected(); err == nil {
			fmt.Printf("[Update] Table: %s | Rows Affected: %d\n", q.model.TableName, affected)
		} else {
			fmt.Printf("[Update] Table: %s | Executed (affected count unknown)\n", q.model.TableName)
		}
	case "InsertRow":
		if len(q.InsertRowFieldTypes) == 0 {
			return fmt.Errorf("no FieldTypes to InsertRow")
		}
		cols := []string{}
		vals := []string{}
		args := []any{}

		for k, v := range q.InsertRowFieldTypes {
			cols = append(cols, fmt.Sprintf("`%s`", k))
			vals = append(vals, "?")
			args = append(args, v)
		}

		queryBuilder := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			q.model.TableName,
			strings.Join(cols, ", "),
			strings.Join(vals, ", "),
		)

		result, err := db.Exec(queryBuilder, args...)
		if err != nil {
			return err
		}
		if id, err := result.LastInsertId(); err == nil {
			fmt.Printf("[InsertRow] Table: %s | Last InsertRowed ID: %d\n", q.model.TableName, id)
		} else {
			fmt.Printf("[InsertRow] Table: %s | Row InsertRowed\n", q.model.TableName)
		}
	case "delete":
		where := q.buildWhere()
		limit := q.buildLimit()

		if where == "" {
			return fmt.Errorf("unsafe delete: WHERE clause is required")
		}

		queryBuilder := fmt.Sprintf("DELETE FROM `%s` %s %s", q.model.TableName, where, limit)
		result, err := db.Exec(queryBuilder, q.whereArgs...)
		if err != nil {
			fmt.Printf("[Delete] Errored queryBuilder: %s\n", queryBuilder)
			return err
		}

		if affected, err := result.RowsAffected(); err == nil {
			fmt.Printf("[Delete] Table: %s | Rows Affected: %d\n", q.model.TableName, affected)
		} else {
			fmt.Printf("[Delete] Table: %s | Executed (affected rows unknown)\n", q.model.TableName)
		}
	default:
		return fmt.Errorf("invalid Exec call: unknown operation '%s'", q.operation)
	}

	q.model.syncComponentWithDB()
	return nil

}

// Exec executes the InsertRow operation.
func (q *InsertRowBuilder) Exec() error {
	db, err := DatabaseHandler.GetDatabase()
	if err != nil {
		return err
	}
	if len(q.InsertRowFieldTypes) == 0 {
		return fmt.Errorf("no FieldTypes to InsertRow")
	}
	cols := []string{}
	vals := []string{}
	args := []any{}
	for k, v := range q.InsertRowFieldTypes {
		cols = append(cols, fmt.Sprintf("`%s`", k))
		vals = append(vals, "?")
		args = append(args, v)
	}
	queryBuilder := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		q.model.TableName,
		strings.Join(cols, ", "),
		strings.Join(vals, ", "),
	)
	result, err := db.Exec(queryBuilder, args...)
	if err != nil {
		return err
	}
	if _, err := result.LastInsertId(); err != nil {
		// 	fmt.Printf("[InsertRow] Table: %s | Last InsertRowed ID: %d\n", q.model.TableName, id)
		// } else {
		fmt.Printf("[InsertRow] Table: %s | Row InsertRowion failed: %s\n", q.model.TableName, err.Error())
	}
	return nil
}

// =======================
// Sorting and Grouping
// =======================

// OrderBy sets the ORDER BY clause for sorting results.
// Usage: .OrderBy("created_at DESC")
func (q *queryBuilder) OrderBy(clause string) *queryBuilder {
	q.orderBy = clause
	return q
}

// GroupBy sets the GROUP BY clause for grouping results.
// Usage: .GroupBy("status")
func (q *queryBuilder) GroupBy(clause string) *queryBuilder {
	q.groupBy = clause
	return q
}

// =======================
// Pagination
// =======================

// Offset sets the OFFSET for skipping a number of rows (for pagination).
// Usage: .Offset(20)
func (q *queryBuilder) Offset(n int64) *queryBuilder {
	q.offset = n
	return q
}

// Page sets both LIMIT and OFFSET for paginated queries.
// Usage: .Page(2, 10) // page 2, 10 results per page
func (q *queryBuilder) Page(page int64, pageSize int64) *queryBuilder {
	if page < 1 {
		page = 1
	}
	q.limit = pageSize
	q.offset = (page - 1) * pageSize
	return q
}

// =======================
// Helper Functions
// =======================

// buildWhere constructs the WHERE clause from the accumulated conditions.
// Returns an empty string if there are no conditions.
func (q *queryBuilder) buildWhere() string {
	if len(q.whereClauses) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(q.whereClauses, " ")
}

// buildLimit constructs the LIMIT clause if a limit is set.
// Returns an empty string if no limit is specified.
func (q *queryBuilder) buildLimit() string {
	if q.limit > 0 {
		return fmt.Sprintf("LIMIT %d", q.limit)
	}
	return ""
}

func (q *queryBuilder) buildOffset() string {
	if q.offset > 0 {
		return fmt.Sprintf("OFFSET %d", q.offset)
	}
	return ""
}

func (q *queryBuilder) Clone() *queryBuilder {
	copy := *q
	copy.whereClauses = append([]string{}, q.whereClauses...)
	copy.whereArgs = append([]any{}, q.whereArgs...)
	copy.setClauses = append([]string{}, q.setClauses...)
	copy.setArgs = append([]any{}, q.setArgs...)
	return &copy
}
