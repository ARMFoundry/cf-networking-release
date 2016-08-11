package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"guardian-cni-adapter/config"
	"guardian-cni-adapter/controller"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/lager"
)

var (
	action            string
	handle            string
	cfg               config.Config
	encodedProperties string
)

func parseArgs(allArgs []string) error {
	var gardenNetworkSpec, configFilePath string

	flagSet := flag.NewFlagSet("", flag.ContinueOnError)

	flagSet.StringVar(&action, "action", "", "")
	flagSet.StringVar(&handle, "handle", "", "")
	flagSet.StringVar(&gardenNetworkSpec, "network", "", "")
	flagSet.StringVar(&encodedProperties, "properties", "", "")
	flagSet.StringVar(&configFilePath, "configFile", "", "")

	err := flagSet.Parse(allArgs[1:])
	if err != nil {
		return err
	}
	if len(flagSet.Args()) > 0 {
		return fmt.Errorf("unexpected extra args: %+v", flagSet.Args())
	}

	if handle == "" {
		return fmt.Errorf("missing required flag 'handle'")
	}

	if configFilePath == "" {
		return fmt.Errorf("missing required flag 'configFile'")
	}

	cfg, err = config.New(configFilePath)
	if err != nil {
		return err
	}

	if action == "" {
		return fmt.Errorf("missing required flag 'action'")
	}

	return nil
}

func die(logger lager.Logger, action string, err error, data ...lager.Data) {
	logger.Error(action, err, data...)
	os.Exit(1)
}

func main() {
	logger := lager.NewLogger("guardian-cni-adapter")
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.INFO))

	if len(os.Args) == 1 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, "this is used by garden-runc.  don't run it directly.")
		os.Exit(1)
	}

	inputBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		die(logger, "read-stdin", err)
	}

	err = parseArgs(os.Args)
	if err != nil {
		die(logger, "parse-args", err)
	}

	var containerState struct {
		Pid int
	}
	if action == "up" {
		err = json.Unmarshal(inputBytes, &containerState)
		if err != nil {
			die(logger, "reading-stdin", err, lager.Data{"stdin": string(inputBytes)})
		}
	}

	cniController := &controller.CNIController{
		PluginDir: cfg.CniPluginDir,
		ConfigDir: cfg.CniConfigDir,
		Logger:    logger,
	}

	mounter := &controller.Mounter{}

	manager := &controller.Manager{
		CNIController: cniController,
		Mounter:       mounter,
		BindMountRoot: cfg.BindMountDir,
	}

	logger.Info("action", lager.Data{"action": action})

	switch action {
	case "up":
		properties, err := manager.Up(containerState.Pid, handle, encodedProperties)
		if err != nil {
			die(logger, "manager-up", err)
		}
		err = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{"properties": properties})
		if err != nil {
			die(logger, "writing-properties", err)
		}
	case "down":
		err = manager.Down(handle, encodedProperties)
		if err != nil {
			die(logger, "manager-down", err)
		}
	default:
		die(logger, "unknown-action", fmt.Errorf("unrecognized action: %s", action))
	}
}
