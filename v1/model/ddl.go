package model

import (
	"fmt"

	DatabaseHandler "github.com/vrianta/agai/v1/database"
	"github.com/vrianta/agai/v1/internal/flags"
	log "github.com/vrianta/gulog"
)

func (m *meta) addField(field *Field) {
	/*
		ALTER TABLE `users`
		ADD `newel` VARCHAR(20) NULL DEFAULT 'dwads' AFTER `userId`,
		ADD INDEX (`newel`);
	*/

	query := "ALTER TABLE `" + m.TableName + "`\n"
	query += "ADD " + field.columnDefinition() + field.addIndexStatement() + ";"

	if databaseObj, err := DatabaseHandler.GetDatabase(); err != nil {
		panic("Error While Adding new Field to the table" + err.Error())
	} else {
		// add migration schemas while SyncDatabaseEnabled enabled
		if flags.SyncDatabaseEnabled {
			recordMigrationSchema(query)
		}
		if _, sql_err := databaseObj.Exec(query); sql_err != nil {
			panic("Error While Updating the Table Field" + sql_err.Error() + "\nWith Query : " + query)
		} else {
			fmt.Printf("[AddField]      Table: %-20s | Field Added: %-20s\n", m.TableName, field.name)
		}
	}
}

// function to change the field details
func (m *meta) modifyDBField(field *Field) {
	// ALTER TABLE `users` CHANGE `userId` `userId` INT(30) NOT NULL AUTO_INCREMENT;
	response := "ALTER TABLE `" + m.TableName + "`"
	//DROP FOREIGN KEY IF EXISTS `fk_Users_Id`;
	if field.fk != nil {
		response += " DROP FOREIGN KEY IF EXISTS `fk_" + field.table_name + "_" + field.name + "`,\n"
		response += " CHANGE `" + field.name + "` " + field.columnDefinition() + ";"
	}

	if databaseObj, err := DatabaseHandler.GetDatabase(); err != nil {
		panic("\nError While Changing Field" + err.Error())
	} else {
		if _, sql_err := databaseObj.Exec(response); sql_err != nil {
			panic("\nError While Changing the Table Field" + sql_err.Error() + "\nSQL queryBuilder: " + response)
		} else {
			log.Info("\n[modifyDBField]   Table: %-20s | Field Updated: %-20s\n", m.TableName, field.name)
		}
	}
}

// Drop a field from the databasess
func (m *meta) removeDBField(fieldName string) {
	//ALTER TABLE `users` DROP `userId`;
	queryBuilder := "ALTER TABLE `" + m.TableName + "`"
	queryBuilder += " DROP FOREIGN KEY IF EXISTS `fk_" + m.TableName + "_" + fieldName + "`,\n"
	queryBuilder += " DROP `" + fieldName + "`;"
	if databaseObj, err := DatabaseHandler.GetDatabase(); err != nil {
		panic(fmt.Sprintf("\nError While Deleting Field : %s\n queryBuilder: %s", err.Error(), queryBuilder))
	} else {
		if _, sql_err := databaseObj.Exec(queryBuilder); sql_err != nil {
			panic(fmt.Sprintf("\nError While Deleting the Field: %s\n queryBuilder: %s", sql_err.Error(), queryBuilder))
		} else {
			fmt.Printf("\n[removeDBField]     Table: %-20s | Field Dropped: %-20s\n", m.TableName, fieldName)
		}
	}
}
