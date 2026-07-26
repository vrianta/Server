package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/vrianta/gulog"
)

// Joson pattern will be
/*
{
 "primary_key": {
	// table components including the primary key
 }
}
*/
// Loads a component JSON file and stores it in the model's components map
func (m *meta) loadComponentFromDisk() {
	log.Info("[component] Loading component for table: %s\n", m.TableName)

	path := filepath.Join(componentsDir, m.TableName+".component.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Try a case-insensitive lookup in the components directory. Some projects
		// may have created component files with different casing (e.g. "Settings.component.json").
		if dirEntries, derr := os.ReadDir(componentsDir); derr == nil {
			found := ""
			target := m.TableName + ".component.json"
			for _, de := range dirEntries {
				if strings.EqualFold(de.Name(), target) {
					found = filepath.Join(componentsDir, de.Name())
					break
				}
			}
			if found != "" {
				path = found
			} else {
				log.Error("[component] No JSON file found for: %s\n", m.TableName)
				m.components = make(components)
				return
			}
		} else {
			log.Error("[component] No JSON file found for: %s\n", m.TableName)
			m.components = make(components)
			return
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Error("Failed to read %s: %v\n", path, err)
		return
	}

	var raw components
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Error("[component] Error unmarshaling %s: %v\n", path, err)
		return
	}

	m.components = raw
	log.Debug("Component for table '%s' loaded with %d items\n", m.TableName, len(raw))
}

// Saves the model's in-memory components to its JSON file
func (m *meta) saveComponentToDisk() error {
	path := filepath.Join(componentsDir, m.TableName+".component.json")
	bytes, err := json.MarshalIndent(m.components, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}

/*
 * syncComponentWithDB ensures that local components and database components match.
 *
 * Steps:
 * 1. If there are no local components, exit.
 * 2. Fetch current components from the database.
 * 3. If the database is empty, insert all local components into it.
 * 4. Add any local components missing from the database.
 * 5. Remove any database components that don't exist locally.
 * 6. Replace the local components list with the latest data from the database.
 * 7. Save the updated components list to disk.
 *
 * The database is treated as the final source of truth after syncing.
 */
func (m *meta) syncComponentWithDB() error {
	if len(m.components) == 0 {
		fmt.Printf("[component] No components to sync for: %s\n", m.TableName)
		return nil
	}

	dbResults, err := m.Get().Fetch()
	if err != nil {
		return fmt.Errorf("[component] fetch error for %s: %w", m.TableName, err)
	}

	if len(dbResults) == 0 {
		for _, localItem := range m.components {
			if err := m.InsertRow(localItem); err != nil {
				log.Error("[component] Insert failed: %v\n", err)
			}
		}
		return nil
	}

	// Add missing
	for k, v := range m.components {
		if m.primary.Type == FieldTypes.Int {
			if int_k, err := strconv.Atoi(k); err != nil {
				return err
			} else {
				// int64 because whe I added "0" in the index of component and unmarshal it that converts that in int64
				if _, ok := dbResults[int64(int_k)]; !ok {
					log.Warn("[component] DB missing component %s in table %s", k, m.TableName)
					log.Info("We are adding the component %s into the table %s", k, m.TableName)
					if err := m.InsertRow(v); err != nil {
						panic("Failed to update the Component :" + err.Error())
					}
					dbResults[k] = Result(v)
				}
			}
		} else {
			if _, ok := dbResults[k]; !ok {
				log.Warn("[component] DB missing component '%s' in table %s\n", k, m.TableName)
				if err := m.InsertRow(v); err != nil {
					panic("Failed to update the Component :" + err.Error())
				}
				dbResults[k] = Result(v)
			}
		}
	}

	// Remove stale
	for k := range dbResults {
		if _, ok := m.components[fmt.Sprint(k)]; !ok {
			_ = m.Delete().Where(m.primary).Is(k).Exec()
		}
	}

	// Update component file with DB contents
	updated := make(components)
	for k, v := range dbResults {
		c := component(v)
		updated[fmt.Sprint(k)] = c
	}
	m.components = updated

	return m.saveComponentToDisk()
}

// Refreshes model's in-memory components from DB and rewrites JSON
func (m *meta) refreshComponentFromDB() {
	if !m.HasPrimaryKey() {
		log.Error("[component] Model %s missing primary key\n", m.TableName)
		return
	}
	results, err := m.Get().Fetch()
	if err != nil {
		log.Error("[component] Fetch error for %s: %v\n", m.TableName, err)
		return
	}
	updated := make(components)
	for k, v := range results {
		c := component(v)
		updated[fmt.Sprint(k)] = c
	}

	if len(updated) == 0 && len(m.components) > 0 {
		// means the local component file has data in it but the database does not have
		// we would update the database in this stage, but ask the user to confirm
		log.Info("Database is empty but the local file has data do you want to update the Database?(y/n):")
		// reader := bufio.NewReader(os.Stdin)
		var input string
		fmt.Scanln(&input)
		switch input {
		case "y":
			// update the database
			m.syncComponentWithDB()
		case "n":
			m.components = updated
			_ = m.saveComponentToDisk()
		default:
			log.Error("Passed Wrong Input: %s", input)
			m.refreshComponentFromDB()
		}
	} else {
		m.components = updated
		_ = m.saveComponentToDisk()
	}

}

func (m *meta) GetComponents() components {
	if m.components == nil {
		m.loadComponentFromDisk()
	}
	return m.components
}

func (m *meta) GetComponent(id string) (component, bool) {
	// have to add conditional checks before returning
	component, ok := m.components[id]
	return component, ok
}

// pass the id of the component you want to update and the value you want to put
func (m *meta) UpdateComponent(id string, value component) error {

	if _, ok := m.components[id]; ok {
		m.components[id] = value
	} else {
		return fmt.Errorf("no componnet found with such name")
	}

	q := m.Update(nil).Where(m.primary).Is(id)
	for idx, val := range value {
		q = q.SetWithFieldName(idx).To(val)
	}

	if err := q.Exec(); err != nil {
		return err
	}

	if _, ok := m.components[id]; ok {
		m.components[id] = value
	} else {
		return fmt.Errorf("no componnet found with such name")
	}

	return nil
}

func (c component) FieldValue(field string) (any, bool) {
	val, ok := c[field]
	return val, ok
}

/*
 * Converting the any datatype to result of possible
 **/
func MakeResult(data any) (Result, error) {
	if res, ok := data.(Result); ok {
		return res, nil
	} else if res, ok := data.(map[string]interface{}); ok {
		return Result(res), nil
	} else {
		return nil, fmt.Errorf("cannot convert to Result")
	}
}
