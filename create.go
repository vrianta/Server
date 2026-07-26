package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/vrianta/agai/v1/config"
	log "github.com/vrianta/gulog"
)

func create_controller() {

	if len(f.controller_names_to_create) > 0 {
		log.WriteLog("---------------------------------\n")
		log.WriteLog("Creating Controllers:\n")
		log.WriteLog("---------------------------------\n")
	} else {
		return
	}

	for _, controller_name := range f.controller_names_to_create {

		// Set output location: controller/controller_name/controller_name.controller.go
		controller_output_location := fmt.Sprintf("./%s/%s.controller.go", f.controllers_root, controller_name)

		if file_info, err := os.Stat(controller_output_location); file_info != nil && err == nil {
			log.Warn("⚠️  Skipped: Controller '%s' already exists at %s", controller_name, controller_output_location)
			continue
		}

		log.Info("🔧 Creating controller: %s", controller_name)

		var controller_nameSplit = strings.Split(controller_output_location, "/")
		var controller_nameSplit_lenth = len(controller_nameSplit)
		var package_name = "controllers"

		if controller_nameSplit_lenth < 1 {
			package_name = controller_nameSplit[(controller_nameSplit_lenth - 2)]
		}
		view_name := ""
		if f.create_view {
			view_name = package_name
			f.view_names_to_create = append(f.view_names_to_create, view_name)
		}

		// Read the template from embed
		controller_template, err := templates.ReadFile("templates/controller.go.template")
		if err != nil {
			log.Error("❌ Error: Failed to read controller template: %v", err)
			return
		}

		// targetDir := filepath.Join(f.controllers_root, controller_name)

		// // Create controller directory
		// log.Info("📁 Creating directory: %s", targetDir)
		// if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		// 	log.Error("❌ Error: Could not create directory %s: %v", targetDir, err)
		// 	return
		// }

		// Parse and render the template
		tmpl, err := template.New(controller_name).Parse(string(controller_template))
		if err != nil {
			log.Error("❌ Error: Template parse failed for %s: %v", "controller.go.template", err)
			return
		}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]string{
			"package_name":    package_name,
			"controller_name": capitalize(controller_nameSplit[(controller_nameSplit_lenth - 1)]),
			"view_name":       view_name,
		})
		if err != nil {
			log.Error("❌ Error: Template execution failed for %s: %v", "controller.go.template", err)
			return
		}

		// Write the final file
		err = os.WriteFile(controller_output_location, buf.Bytes(), os.ModePerm)
		if err != nil {
			log.Error("❌ Error: Could not write controller file to %s: %v", controller_output_location, err)
			return
		}

		// create controller view

		log.Info("✅ Controller '%s' created at %s", controller_name, controller_output_location)
		log.Warn("There are no option to update the routes automatically - please make sure you update the routes in routes.go file int the root directory")
	}

	log.WriteLog("---------------------------------")
}

/*
Create View :
check if the file present with the view name
if not it will create a view.php file
*/
func create_view() {
	if len(f.view_names_to_create) == 0 {
		return
	}

	log.WriteLog("---------------------------------")
	log.WriteLog("Creating Views: ")
	log.WriteLog("---------------------------------")

	for _, view_name := range f.view_names_to_create {
		// Normalize view name and allow nested paths like "admin/dashboard"
		viewNameClean := strings.TrimSpace(view_name)
		viewNameClean = strings.Trim(viewNameClean, "/")

		viewRoot := config.GetWebConfig().ViewFolder
		viewDir := filepath.Join(viewRoot, filepath.FromSlash(viewNameClean))

		// Check if view already exists
		if fileInfo, err := os.Stat(viewDir); err == nil {
			if fileInfo.IsDir() {
				log.Warn("⚠️  Skipped: View '%s' already exists at %s", view_name, viewDir)
				continue
			}
			log.Warn("⚠️  Skipped: Path exists and is not a directory: %s", viewDir)
			continue
		} else if !os.IsNotExist(err) {
			log.Error("❌ Error checking view directory %s: %v", viewDir, err)
			continue
		}

		log.Info("🧩 Creating view: %s", view_name)

		viewFile := viewDir + ".php"

		// Read the view template from embedded FS
		viewTemplate, err := templates.ReadFile("templates/view.php.template")
		if err != nil {
			log.Error("❌ Error: Failed to read view template: %v", err)
			return
		}

		// Create the view directory (including parents)
		log.Info("📁 Creating directory: %s", viewDir)
		if err := os.MkdirAll(viewDir, os.ModePerm); err != nil {
			log.Error("❌ Error: Could not create view directory %s: %v", viewDir, err)
			return
		}

		// Parse and render the template
		tmpl, err := template.New(view_name).Parse(string(viewTemplate))
		if err != nil {
			log.Error("❌ Error: Template parse failed for %s: %v", "index.php.template", err)
			return
		}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]string{
			"view_name": capitalize(filepath.Base(viewNameClean)),
		})
		if err != nil {
			log.Error("❌ Error: Template execution failed for %s: %v", "index.php.template", err)
			return
		}

		// Write index.php to view folder
		if err := os.WriteFile(viewFile, buf.Bytes(), 0644); err != nil {
			log.Error("❌ Error: Could not write view file to %s: %v", viewFile, err)
			return
		}

		log.Info("✅ View '%s' created at %s", view_name, viewFile)
	}
	log.WriteLog("---------------------------------")
}

/*
- create_theme is same as create_view but it will create the view in the theme folder instead of views folder and it will not log the warning to update the routes because theme views are not directly linked to routes
- it will creat the folder theme if that does not exists
- then it will create a subfolder with the view name but if the folder exists int will log.Error that the view is already exists and return the function
- inside that it will create a file called index.php
*/
func create_theme() {

	if len(f.view_names_to_create) > 0 {
		log.WriteLog("---------------------------------")
		log.WriteLog("Creating Views: ")
		log.WriteLog("---------------------------------")

	} else {
		return
	}

	for _, view_name := range f.view_names_to_create {

		viewRoot := config.GetWebConfig().ViewFolder
		viewDir := filepath.Join(viewRoot, view_name)
		// Check if view already exists
		if fileInfo, err := os.Stat(viewDir); err == nil && fileInfo.IsDir() {
			log.Warn("⚠️  Skipped: View '%s' already exists at %s", view_name, viewDir)
			continue
		}

		log.Info("🧩 Creating view: %s", view_name)

		viewFile := filepath.Join(viewDir, "index.php")

		// Read the view template from embedded FS
		viewTemplate, err := templates.ReadFile("templates/index.php.template")
		if err != nil {
			log.Error("❌ Error: Failed to read view template: %v", err)
			return
		}

		// Create the view directory
		log.Info("📁 Creating directory: %s", viewDir)
		if err := os.MkdirAll(viewDir, os.ModePerm); err != nil {
			log.Error("❌ Error: Could not create view directory %s: %v", viewDir, err)
			return
		}

		// Parse and render the template
		tmpl, err := template.New(view_name).Parse(string(viewTemplate))
		if err != nil {
			log.Error("❌ Error: Template parse failed for %s: %v", "index.php.template", err)
			return
		}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]string{
			"view_name": capitalize(view_name),
		})
		if err != nil {
			log.Error("❌ Error: Template execution failed for %s: %v", "index.php.template", err)
			return
		}

		// Write index.php to view folder
		err = os.WriteFile(viewFile, buf.Bytes(), 0644)
		if err != nil {
			log.Error("❌ Error: Could not write view file to %s: %v", viewFile, err)
			return
		}

		log.Info("✅ View '%s' created at %s", view_name, viewFile)
	}
	log.WriteLog("---------------------------------")
}

// Creating template Model
func create_models() {

	if len(f.model_names_to_create) > 0 {
		log.WriteLog("---------------------------------")
		log.WriteLog("Creating Models: ")
		log.WriteLog("---------------------------------")
	} else {
		return
	}
	for _, model_name := range f.model_names_to_create {

		model_output_path := fmt.Sprintf("models/%s.model.go", strings.ToLower(model_name))

		// Skip if model file already exists
		if file_info, err := os.Stat(model_output_path); file_info != nil && err == nil {
			log.Warn("⚠️  Skipped: Model '%s' already exists at %s", model_name, model_output_path)
			continue
		}

		log.Info("🛠️  Creating model: %s", model_name)

		if f.create_component {
			f.component_names_to_create = append(f.component_names_to_create, model_name)
		}

		// Read model template
		model_template, err := templates.ReadFile("templates/model.go.template")
		if err != nil {
			log.Error("❌ Error: Failed to read model template: %v", err)
			return
		}

		// Parse template
		tmpl, err := template.New(model_name).Parse(string(model_template))
		if err != nil {
			log.Error("❌ Error: Template parse failed: %v", err)
			return
		}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]string{
			"model_name": capitalize(model_name),
			"table_name": strings.ToLower(model_name),
		})
		if err != nil {
			log.Error("❌ Error: Template execution failed: %v", err)
			return
		}

		// Ensure models directory exists
		if err := os.MkdirAll("models", os.ModePerm); err != nil {
			log.Error("❌ Error: Failed to create models directory: %v", err)
			return
		}

		// Write model file
		err = os.WriteFile(model_output_path, buf.Bytes(), 0644)
		if err != nil {
			log.Error("❌ Error: Failed to write model file: %v", err)
			return
		}

		log.Info("✅ Model '%s' created at: %s", model_name, model_output_path)
	}

	log.WriteLog("---------------------------------")

}

/*
First it will check if the components folder is present or not if not then create it
Then it will check if the component alreadt present or not if present then skip the creation and log error and return
Then check if the same named model is present in the models folder or not
Then eavluate the model in the file and craete component according to that
*/
func create_components() {

	if len(f.component_names_to_create) > 0 {
		log.WriteLog("Creating Components: ")
		log.WriteLog("---------------------------------")
	} else {
		return
	}
	for _, componentName := range f.component_names_to_create {

		componentFile := filepath.Join("components", fmt.Sprintf("%s.component.json", strings.ToLower(componentName)))

		// Check if component already exists
		if _, err := os.Stat(componentFile); err == nil {
			log.Warn("⚠️  Skipped: Component already exists at %s", componentFile)
			continue
		}

		log.Info("🧩 Creating component: %s", componentName)

		modelFile := filepath.Join("models", fmt.Sprintf("%s.model.go", strings.ToLower(componentName)))

		// Ensure components/ directory exists
		if err := os.MkdirAll("components", os.ModePerm); err != nil {
			log.Error("❌ Failed to create components directory: %v", err)
			continue
		}

		// Check if model file exists
		modelContent, err := os.ReadFile(modelFile)
		if err != nil {
			log.Error("❌ Error: Model file not found for component '%s' (%s)", componentName, modelFile)
			continue
		}

		// Parse model file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "", modelContent, parser.AllErrors)
		if err != nil {
			log.Error("❌ Error parsing model file for %s: %v", componentName, err)
			continue
		}

		// Extract struct field names from the model.New call
		fields := map[string]interface{}{}

		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}

			ident, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || ident.Sel.Name != "New" {
				return true
			}

			composite, ok := call.Args[1].(*ast.CompositeLit)
			if !ok {
				return true
			}

			for _, elt := range composite.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key := fmt.Sprintf("%s", kv.Key)
				fields[key] = defaultValueForExpr(kv.Key)
			}
			return false
		})

		// Create JSON map
		componentData := map[string]interface{}{
			"0": fields,
		}

		jsonBytes, err := json.MarshalIndent(componentData, "", "  ")
		if err != nil {
			log.Error("❌ Failed to marshal JSON for component '%s': %v", componentName, err)
			continue
		}

		if err := os.WriteFile(componentFile, jsonBytes, 0644); err != nil {
			log.Error("❌ Failed to write component file: %v", err)
			continue
		}

		log.Info("✅ Component '%s' created at %s", componentName, componentFile)
	}

	log.WriteLog("---------------------------------")

}

// defaultValueForExpr returns a Go zero value for a given field expression
func defaultValueForExpr(expr ast.Expr) interface{} {
	switch v := expr.(type) {
	case *ast.CompositeLit:
		return ""
	case *ast.BasicLit:
		if strings.Contains(v.Value, "\"") {
			return ""
		}
		return 0
	case *ast.Ident:
		if v.Name == "true" || v.Name == "false" {
			return false
		}
		return ""
	default:
		return ""
	}
}

func create_configs() {
	create_web_config()
	create_database_config()
	create_smtp_config()
}

// create_web_config reads /templates/config.web.json.template and writes it as config.web.json in the current directory.
func create_web_config() {
	writeFromEmbed("templates/appsettings.json.template", "appsettings.json")
}

// create_database_web_config reads /templates/config.database.json.template and writes it as config.database.json in the current directory.
func create_database_config() {
	writeFromEmbed("templates/config.database.json.template", "config.database.json")
}

// create_database_smtp_config reads /templates/config.smtp.json.template and writes it as config.smtp.json in the current directory.
func create_smtp_config() {
	writeFromEmbed("templates/config.smtp.json.template", "config.smtp.json")
}

// writeFromEmbed reads a file from embedded FS and writes it to the destination file.
func writeFromEmbed(srcPath, destPath string) {

	if fileInfo, err := os.Stat(destPath); err == nil && fileInfo != nil {
		log.Warn("Config File %s is already present in current solution", destPath)
		return
	}

	data, err := templates.ReadFile(srcPath)
	if err != nil {
		log.Error("❌ Failed to read embedded file %s: %v", srcPath, err)
		return
	}

	err = os.WriteFile(destPath, data, 0644)
	if err != nil {
		log.Error("❌ Failed to write file %s: %v", destPath, err)
		return
	}

	log.Info("✅ Created %s", destPath)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

/*
Create required folders needed for create app folder
folders are models, controllers, components, css, js, static
*/
func create_desired_folders() {
	dirs := []string{
		"models",
		"controllers",
		"components",
		"css",
		"js",
		"static",
		"views",
		".vscode",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Error("❌ Failed to create folder '%s': %v", dir, err)
		}
	}
}

// copyDirFromEmbed copies all files under the given embedded subfolder to a destination folder
func copyDirFromEmbed(efs embed.FS, embedPath string, destPath string) error {
	return fs.WalkDir(efs, embedPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read file content
		data, err := efs.ReadFile(path)
		if err != nil {
			return err
		}

		// Remove the prefix (e.g. templates/css/bootstrap/) to get relative path
		relPath := strings.TrimPrefix(path, embedPath)
		relPath = strings.TrimPrefix(relPath, "/") // remove leading slash if present

		// Build destination file path
		destFilePath := filepath.Join(destPath, relPath)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm); err != nil {
			return err
		}

		// Write file to destination
		return os.WriteFile(destFilePath, data, 0644)
	})
}
