package services

import (
	"log"
	"os"
	"path"

	"github.com/devdeck-app/devdeck-server/commands"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func LoadConfig(cmds *[]commands.Command, layout *Layout) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting the user config dir: %s\n", err)
	}
	configDir := path.Join(home, ".config", "devdeck/")

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err := os.Mkdir(configDir, 0755)
		if err != nil {
			log.Fatalf("Error creating the config dir: %v\n", err)
		}
	}

	viper.AddConfigPath(configDir)
	viper.SetConfigType("toml")
	viper.SetConfigName("devdeck")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalln("Config file not found")
		}
	}

	var commands *[]commands.Command

	err = viper.UnmarshalKey("commands", &commands)
	if err != nil {
		log.Fatalf("Error parsing commands: %v", err)
	}

	// TODO: fixme
	(*cmds) = *commands

	err = viper.UnmarshalKey("layout", &layout)
	if err != nil {
		log.Fatalf("Error parsing layout: %v", err)
	}

	if cmds != nil {
		for i := range *cmds {
			if (*cmds)[i].UUID == "" {
				(*cmds)[i].UUID = uuid.New().String()
			}
		}
	}
}
