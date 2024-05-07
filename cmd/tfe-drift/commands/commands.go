package commands

import (
	"context"
	"io"

	"github.com/alecthomas/kingpin/v2"

	"github.com/hashicorp/go-tfe"
	"github.com/slok/tfe-drift/internal/log"
)

const (
	// LoggerTypeDefault is the logger default type.
	LoggerTypeDefault = "default"
	// LoggerTypeJSON is the logger json type.
	LoggerTypeJSON = "json"
)

const (
	defaultFitCliID = "tfe-drift"
)

// Command represents an application command, all commands that want to be executed
// should implement and setup on main.
type Command interface {
	Name() string
	Run(ctx context.Context) error
}

// RootCommand represents the root command configuration and global configuration
// for all the commands.
type RootCommand struct {
	// Global flags.
	Debug      bool
	NoLog      bool
	NoColor    bool
	LoggerType string
	AppID      string
	TFEOrg     string
	TFEToken   string
	TFEAddress string

	// Global instances.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Logger log.Logger
}

// NewRootCommand initializes the main root configuration.
func NewRootCommand(app *kingpin.Application) *RootCommand {
	c := &RootCommand{}

	app.Flag("debug", "Enable debug mode.").BoolVar(&c.Debug)
	app.Flag("no-log", "Disable logger.").BoolVar(&c.NoLog)
	app.Flag("no-color", "Disable logger color.").BoolVar(&c.NoColor)
	app.Flag("logger", "Selects the logger type.").Default(LoggerTypeDefault).EnumVar(&c.LoggerType, LoggerTypeDefault, LoggerTypeJSON)
	app.Flag("app-id", "ID to identify the app.").Default(defaultFitCliID).StringVar(&c.AppID)
	app.Flag("tfe-organization", "The Terraform cloud or enterprise organization.").Required().StringVar(&c.TFEOrg)
	app.Flag("tfe-token", "The Terraform cloud or enterprise API token.").Required().StringVar(&c.TFEToken)
	app.Flag("tfe-address", "The address of the Terraform Enterprise API.").Default(tfe.DefaultAddress).StringVar(&c.TFEAddress)

	return c
}
