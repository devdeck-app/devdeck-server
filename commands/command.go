package commands

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

type CommandType string

const (
	ActionType  = CommandType("action")
	ContextType = CommandType("context")
)

type Command struct {
	UUID        string      `mapstructure:"uuid" json:"uuid"`
	Description string      `mapstructure:"description" json:"description"`
	App         string      `mapstructure:"app" json:"app"`
	Action      string      `mapstructure:"action" json:"action"`
	Icon        string      `mapstructure:"icon,omitempty" json:"icon,omitempty"`
	Type        CommandType `mapstructure:"type" json:"type"`
	Context     string      `mapstructure:"context" json:"context"`
	Main        bool        `mapstructure:"main" json:"-"`
}

func (c Command) OpenApplication() error {
	cmd := exec.Command("open", "-a", c.App)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c Command) Execute() error {
	cmdArgs := strings.Split(c.Action, " ")
	log.Printf("Command args: %v\n", cmdArgs)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = os.Environ()

	log.Println("PATH =", os.Getenv("PATH"))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
