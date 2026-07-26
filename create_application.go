package main

import (
	"bytes"
	"os"
	"os/exec"
	"text/template"

	log "github.com/vrianta/gulog"
)

/*
function to create application
first it will createa folder of the app name if that folder does not exists -> if exists then it will say application already exists and return
change the location to the app_name directory then create controllers models create default configs
create folder css, js, static
copy bootstrap css and js to the desired location
copy a readme.template with basic details
return change directoy to the previous location
*/
func create_application() {
	if !f.create_app {
		return
	}

	if f.application_path == "" {
		f.application_path = f.app_name
	}
	app_name := f.app_name

	// Check if directory already exists and is not empty
	if entries, err := os.ReadDir(f.application_path); err == nil && len(entries) > 0 && f.application_path != "." {
		log.Error("❌ Application '%s' already exists.", app_name)
		return
	}

	// Create application directory
	if err := os.Mkdir(f.application_path, os.ModePerm); err != nil && f.application_path != "." {
		log.Error("❌ Failed to create application '%s': %v", app_name, err)
		return
	}

	// // Store current directory and switch to app directory
	// root, _ := os.Getwd()
	if err := os.Chdir(app_name); err != nil && f.application_path != "." {
		log.Error("❌ Failed to enter application folder: %v", err)
		return
	}

	// Create necessary folders
	create_desired_folders()

	models_to_create := []string{
		"users",
		"users_details",
		"settings",
		"roles",
		"users_roles",
	}

	for _, model_name := range models_to_create {
		// Create default user model
		if model, err := templates.ReadFile("templates/models/" + model_name); err != nil {
			log.Error("❌ Failed to read "+model_name+".model.go.template: %v", err)
		} else {
			err = os.WriteFile("models/"+model_name+".model.go", model, 0644)
			if err != nil {
				log.Error("❌ Failed to write user model file: %v", err)
			}
		}
	}

	components_to_create := []string{
		"settings",
		"roles",
		"users",
		"user_details",
		"user_roles",
	}

	for _, component_name := range components_to_create {
		// Create default Settings Component
		if component, err := templates.ReadFile("templates/components/" + component_name); err != nil {
			log.Error("❌ Failed to read "+component_name+".component.json.template: %v", err)
		} else {
			if tpl, tpl_err := template.New(component_name + ".component.json").Parse(string(component)); tpl_err != nil {
				log.Error("Failed to create template of %s due to - %v", "templates.components."+component_name, tpl_err)
			} else {
				buf := bytes.Buffer{}
				tpl.Execute(&buf, map[string]string{
					"app_name": app_name,
				})
				if err := os.WriteFile("components/"+component_name+".component.json", buf.Bytes(), 0644); err != nil {
					log.Error("❌ Failed to write user model file: %v", err)
				}
			}

		}
	}

	// Copy embedded folders to actual css/ and js/
	copyDirFromEmbed(templates, "templates/css", "css")
	// copyDirFromEmbed(templates, "templates/css/bootstrap-Icons", "css/bootstrap-icons")
	copyDirFromEmbed(templates, "templates/js", "js")

	// Copy embedded .vscode folder to actual .vscode/
	copyDirFromEmbed(templates, "templates/.vscode", ".vscode")

	// Default home component to create
	f.controller_names_to_create = append(f.controller_names_to_create, "home")
	f.create_view = true
	f.view_names_to_create = append(f.view_names_to_create, "home")

	// create main.go
	if main_go, err := templates.ReadFile("templates/main.go.template"); err != nil {
		log.Error("❌ Failed to read main.go.template: %v", err)
	} else {
		if tmpl, err := template.New("main").Parse(string(main_go)); err != nil {
			log.Error("Main to Load Main File Template %v", err)
		} else {
			var buff bytes.Buffer
			tmpl.Execute(&buff, map[string]string{
				"app_name": app_name,
			})
			if err := os.WriteFile("main.go", buff.Bytes(), 0644); err != nil {
				log.Error("❌ Failed to write user model file: %v", err)
			}
		}

	}

	// create routes.go
	if routes, err := templates.ReadFile("templates/routes.go.template"); err != nil {
		log.Error("❌ Failed to read routes.go.template: %v", err)
	} else {
		if tpl, tpl_err := template.New("routes.go.template").Parse(string(routes)); tpl_err != nil {
			log.Error("Failed to create template of %s due to - %v", "templates/routes.go.template", tpl_err)
		} else {
			buf := bytes.Buffer{}
			tpl.Execute(&buf, map[string]string{
				"app_name": app_name,
			})
			if err := os.WriteFile("routes.go", buf.Bytes(), 0644); err != nil {
				log.Error("❌ Failed to write user model file: %v", err)
			}
		}

	}

	create_configs() // create different configs
	first_setup()    // update user.component.json from user input

	if err := exec.Command("go", "get", "github.com/go-sql-driver/mysql").Run(); err != nil {
		log.Error("Failed to install package github.com/go-sql-driver/mysql: %v", err)
	}

	if err := exec.Command("go", "get", "github.com/vrianta/agai@"+agai_version).Run(); err != nil {
		log.Error("Failed to install package github.com/vrianta/agai: %v", err)
	}
	if err := exec.Command("go", "mod", "init", app_name).Run(); err != nil {
		log.Error("Failed to initialize go module: %v", err)
	}

	if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
		log.Error("Failed to tidy go module: %v", err)
	}

	if err := exec.Command("go", "run", ".", "-mm", "-mc").Run(); err != nil {
		log.Error("Failed to run agai CLI setup: %v", err)
	}

}
