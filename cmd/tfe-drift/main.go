package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"

	"github.com/slok/tfe-drift/cmd/tfe-drift/commands"
	"github.com/slok/tfe-drift/internal/info"
	"github.com/slok/tfe-drift/internal/internalerrors"
	"github.com/slok/tfe-drift/internal/log"
	loglogrus "github.com/slok/tfe-drift/internal/log/logrus"
)

// Run runs the main application.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (err error) {
	app := kingpin.New("tfe-drift", "Automated Terraform cloud drift checker.")
	app.DefaultEnvars()
	rootCmd := commands.NewRootCommand(app)

	// Setup commands (registers flags).
	versionCmd := commands.NewVersionCommand(rootCmd, app)
	runCmd := commands.NewRunCommand(rootCmd, app)
	controllerCmd := commands.NewControllerCommand(rootCmd, app)

	cmds := map[string]commands.Command{
		versionCmd.Name():    versionCmd,
		runCmd.Name():        runCmd,
		controllerCmd.Name(): controllerCmd,
	}

	// Parse commandline.
	cmdName, err := app.Parse(args[1:])
	if err != nil {
		return fmt.Errorf("invalid command configuration: %w", err)
	}

	// Set standard input/output.
	rootCmd.Stdin = stdin
	rootCmd.Stdout = stdout
	rootCmd.Stderr = stderr

	// Set logger.
	rootCmd.Logger = getLogger(ctx, *rootCmd)

	var g run.Group

	// OS signals.
	{
		sigC := make(chan os.Signal, 1)
		exitC := make(chan struct{})
		signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

		g.Add(
			func() error {
				select {
				case s := <-sigC:
					rootCmd.Logger.Infof("Signal %s received", s)
					return nil
				case <-exitC:
					return nil
				}
			},
			func(_ error) {
				close(exitC)
			},
		)
	}

	// Execute command.
	{
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		g.Add(
			func() error {
				err := cmds[cmdName].Run(ctx)
				if err != nil {
					return fmt.Errorf("%q command failed: %w", cmdName, err)
				}
				return nil
			},
			func(_ error) {
				cancel()
			},
		)

	}

	return g.Run()
}

// getLogger returns the application logger.
func getLogger(ctx context.Context, config commands.RootCommand) log.Logger {
	if config.NoLog {
		return log.Noop
	}

	// If not logger disabled use logrus logger.
	logrusLog := logrus.New()
	logrusLog.Out = config.Stderr // By default logger goes to stderr (so it can split stdout prints).
	logrusLogEntry := logrus.NewEntry(logrusLog)

	if config.Debug {
		logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	}

	// Log format.
	switch config.LoggerType {
	case commands.LoggerTypeDefault:
		logrusLogEntry.Logger.SetFormatter(&logrus.TextFormatter{
			ForceColors:   !config.NoColor,
			DisableColors: config.NoColor,
		})
	case commands.LoggerTypeJSON:
		logrusLogEntry.Logger.SetFormatter(&logrus.JSONFormatter{})
	}

	logger := loglogrus.NewLogrus(logrusLogEntry).WithValues(log.Kv{
		"version": info.Version,
	})

	logger.Debugf("Debug level is enabled") // Will log only when debug enabled.

	return logger
}

func main() {
	ctx := context.Background()
	err := Run(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		switch {
		// Detecting drifts is not a regular error: Quiet and other different code.
		case errors.Is(err, internalerrors.ErrDriftDetected):
			fmt.Fprint(os.Stderr, "Drift detected")
			os.Exit(2)
		case errors.Is(err, internalerrors.ErrDriftDetectionPlanFailed):
			fmt.Fprint(os.Stderr, "Drift detection plan failed")
			os.Exit(3)
		}

		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
