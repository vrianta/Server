package template

import (
	"net/http"
	"os"

	"github.com/vrianta/agai/v1/config"
	"github.com/vrianta/agai/v1/utils"
	log "github.com/vrianta/gulog"
)

var view_folder = utils.JoinPath(".", config.GetViewFolder())
var _templateInfo = make(map[string]templateInfo)

// each folder is a theme and it will be store as theme name in the template registry
func init() {

	if objects, err := os.ReadDir(view_folder); err != nil {
		log.Error("Failed to load read Directory %v", err)
		panic("Failed Loading View Folder")
	} else {
		for _, object := range objects {
			if !object.IsDir() {
				full_file_name := object.Name()          // name of file ex. hello.go
				_fileInfo := GetFileData(full_file_name) // get file info

				// templateComponents[_fileInfo.fileName] = nil

				if _fileInfo.fileName == "404" {
					_fileInfo.Uri = "/404/"
				}
				_templateInfo[_fileInfo.fileName] = _fileInfo

			} else { // register themes
				folder_name := object.Name()
				RegisterTheme(folder_name)
			}
		}
		for key, fileInfo := range _templateInfo {
			templateComponents[key] = createTemplateContext(fileInfo.folderPath, fileInfo.fileName, fileInfo.fileType)
			if fileInfo.fileName == "404" {
				http.HandleFunc(fileInfo.Uri, func(w http.ResponseWriter, r *http.Request) {
					t, _ := templateComponents[key]
					if !config.GetWebConfig().Build {
						// log.WriteLogf("Updating the Template")
						t.Update()
					}

					buf, _ := t.Execute("")
					w.Write(buf)

				})
			}
		}
	}
}

func RegisterTheme(theme_folder string) {

	var found_404 bool = false // to see if the 404 page is present in the current theme or directory
	full_theme_path := utils.JoinPath(view_folder, theme_folder)

	if objects, err := os.ReadDir(full_theme_path); err != nil {
		log.Error("Failed to load read Directory %v", err)
		panic("Failed Loading View Folder")
	} else {
		for _, object := range objects {
			if object.IsDir() {
				RegisterTheme(utils.JoinPath(theme_folder, object.Name()))
			} else {
				full_file_name := object.Name()          // name of file ex. hello.go
				_fileInfo := GetFileData(full_file_name) // get file info

				// templateRegistry[utils.JoinPath(theme_folder, _fileInfo.fileName)] = createTemplateContext(file_full_path, full_file_name, _fileInfo.fileType)
				// templateComponents[theme_folder+"."+_fileInfo.fileName] = createTemplateContext(file_full_path, full_file_name, _fileInfo.fileType)

				if _fileInfo.fileName == "404" {
					found_404 = true
					_fileInfo.Uri = "/" + utils.JoinPath(theme_folder, _fileInfo.fileName)
				}
				_fileInfo.folderPath = utils.JoinPath(full_theme_path, full_file_name) // changing it because for themes full_file_name needs to be pass as fileName
				_templateInfo[theme_folder+"."+_fileInfo.fileName] = _fileInfo

			}
		}
	}

	if !found_404 {
		http.HandleFunc("/"+theme_folder+"/404/", func(w http.ResponseWriter, r *http.Request) {
			w.Write(_404__)
		})
	}
}

func createTemplateContext(view_path, full_file_name, file_type string) *Context {

	// Register the template using the custom Template package
	if _template, err := create(view_path, full_file_name, file_type, config.GetBuild()); err != nil {
		log.Error("Failed to Create the template: %s Error: %v", view_path, err)
		panic("")
	} else {
		return _template
	}

}
