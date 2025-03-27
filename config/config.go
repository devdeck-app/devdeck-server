package config

import (
	"log"
	"os"
	"path"
)

func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting the user config dir: %s\n", err)
	}
	configDir := path.Join(home, ".config", "devdeck/")

	return configDir
}
