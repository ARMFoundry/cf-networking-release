package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"garden-external-networker/cni"
	"garden-external-networker/config"
	"garden-external-networker/controller"
	"garden-external-networker/filelock"
	"garden-external-networker/port_allocator"
	"io/ioutil"
	"os"

	"github.com/coreos/go-iptables/iptables"

	"code.cloudfoundry.org/lager"
)

var (
	action            string
	handle            string
	cfg               config.Config
	encodedProperties string
	gardenNetworkSpec string
	deprecatedNetwork string
)

func parseArgs(allArgs []string) error {
	var configFilePath string

	flagSet := flag.NewFlagSet("", flag.ContinueOnError)

	flagSet.StringVar(&action, "action", "", "")
	flagSet.StringVar(&handle, "handle", "", "")
	flagSet.StringVar(&deprecatedNetwork, "network", "", "")
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
	logger := lager.NewLogger("garden-external-networker")
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

	cniLoader := &cni.CNILoader{
		PluginDir: cfg.CniPluginDir,
		ConfigDir: cfg.CniConfigDir,
		Logger:    logger,
	}

	networks, err := cniLoader.GetNetworkConfigs()
	if err != nil {
		die(logger, "load-cni-plugins", err)
	}

	cniController := &cni.CNIController{
		Logger:         logger,
		CNIConfig:      cniLoader.GetCNIConfig(),
		NetworkConfigs: networks,
	}

	mounter := &controller.Mounter{}

	ipt, err := iptables.New()
	if err != nil {
		die(logger, "iptables-new", err)
	}

	locker := &filelock.Locker{Path: cfg.StateFilePath}
	tracker := &port_allocator.Tracker{
		Logger:    logger,
		StartPort: cfg.StartPort,
		Capacity:  cfg.TotalPorts,
	}
	serializer := &port_allocator.Serializer{}
	portAllocator := &port_allocator.PortAllocator{
		Tracker:    tracker,
		Serializer: serializer,
		Locker:     locker,
	}

	manager := &controller.Manager{
		Logger:         logger,
		CNIController:  cniController,
		Mounter:        mounter,
		BindMountRoot:  cfg.BindMountDir,
		PortAllocator:  portAllocator,
		OverlayNetwork: cfg.OverlayNetwork,
		IPTables:       ipt,
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
	case "net-out":
		err = manager.NetOut(handle, encodedProperties)
		if err != nil {
			die(logger, "manager-net-out", err)
		}
	case "net-in":
		netInResult, err := manager.NetIn(handle, encodedProperties)
		if err != nil {
			die(logger, "manager-net-in", err)
		}
		err = json.NewEncoder(os.Stdout).Encode(netInResult)
		if err != nil {
			die(logger, "writing-net-in-result", err)
		}
	default:
		die(logger, "unknown-action", fmt.Errorf("unrecognized action: %s", action))
	}
}
