package main

import log "github.com/vrianta/gulog"

func migrate_model_and_component() {
	if f.migrate_model {
		// start migrating the models

		log.Info("Migrating Models")
		migrate_models := new_migrate_model_cmd()

		if output, err := migrate_models.Output(); err != nil {
			log.Error("Model migration failed: %s", err.Error())
		} else {
			log.Info("%s", string(output))
		}
	}

	if f.migrate_component {

		log.Info("Migrating Components")
		migrate_components := new_migrate_component_cmd()

		if output, err := migrate_components.Output(); err != nil {
			log.Error("Component migration failed:\nError: %s\nOutput: %s", err.Error(), string(output))
		} else {
			log.Info("%s", string(output))
		}

		migrate_components.Wait()

	}
}
