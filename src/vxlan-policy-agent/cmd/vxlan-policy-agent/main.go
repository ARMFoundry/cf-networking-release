package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"lib/flannel"
	"lib/marshal"
	"lib/policy_client"
	"lib/poller"
	"lib/rules"
	"net/http"
	"os"
	"time"
	"vxlan-policy-agent/config"
	"vxlan-policy-agent/planner"

	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"

	"code.cloudfoundry.org/lager"
	"github.com/coreos/go-iptables/iptables"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func die(logger lager.Logger, action string, err error) {
	logger.Error(action, err)
	os.Exit(1)
}

func main() {
	conf := &config.VxlanPolicyAgent{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("vxlan-policy-agent")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		die(logger, "error reading config", err)
	}

	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		die(logger, "error unmarshalling config", err)
	}
	logger.Info("parsed-config", lager.Data{"config": conf})

	pollInterval := time.Duration(conf.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = time.Second
	}

	flannelInfoReader := &flannel.NetworkInfo{
		FlannelSubnetFilePath: conf.FlannelSubnetFile,
	}
	localSubnetCIDR, overlayNetwork, err := flannelInfoReader.DiscoverNetworkInfo()
	if err != nil {
		die(logger, "discovering network info", err)
	}

	policyClient := policy_client.New(
		logger.Session("policy-client"),
		http.DefaultClient,
		conf.PolicyServerURL,
		marshal.UnmarshalFunc(json.Unmarshal),
	)

	gardenClient := client.New(connection.New(conf.GardenProtocol, conf.GardenAddress))

	ipt, err := iptables.New()
	if err != nil {
		die(logger, "iptables-new", err)
	}

	dynamicPlanner := &planner.VxlanPolicyPlanner{
		GardenClient: gardenClient,
		PolicyClient: policyClient,
		Logger:       logger.Session("rules-updater"),
		VNI:          conf.VNI,
	}

	timestamper := &rules.Timestamper{}
	ruleEnforcer := rules.NewEnforcer(
		logger.Session("rules-enforcer"),
		timestamper,
		ipt,
	)

	defaultLocalChain := rules.Chain{
		Table:       "filter",
		ParentChain: "FORWARD",
		Prefix:      "vpa--local-",
	}

	defaultRemoteChain := rules.Chain{
		Table:       "filter",
		ParentChain: "FORWARD",
		Prefix:      "vpa--remote-",
	}

	defaultMasqueradeChain := rules.Chain{
		Table:       "nat",
		ParentChain: "POSTROUTING",
		Prefix:      "vpa--masq-",
	}

	dynamicChain := rules.Chain{
		Table:       "filter",
		ParentChain: "FORWARD",
		Prefix:      "vpa--",
	}

	vxlanDefaultLocalPlanner := planner.VxlanDefaultLocalPlanner{
		Logger:      logger,
		LocalSubnet: localSubnetCIDR,
	}

	vxlanDefaultRemotePlanner := planner.VxlanDefaultRemotePlanner{
		Logger: logger,
		VNI:    conf.VNI,
	}

	vxlanDefaultMasqueradePlanner := planner.VxlanDefaultMasqueradePlanner{
		Logger:         logger,
		LocalSubnet:    localSubnetCIDR,
		OverlayNetwork: overlayNetwork,
	}

	defaultLocalRules, err := vxlanDefaultLocalPlanner.GetRules()
	if err != nil {
		die(logger, "default-local-rules.GetRules", err)
	}

	err = ruleEnforcer.EnforceOnChain(defaultLocalChain, defaultLocalRules)
	if err != nil {
		die(logger, "enforce-default-local", err)
	}

	defaultRemoteRules, err := vxlanDefaultRemotePlanner.GetRules()
	if err != nil {
		die(logger, "default-local-rules.GetRules", err)
	}
	err = ruleEnforcer.EnforceOnChain(defaultRemoteChain, defaultRemoteRules)
	if err != nil {
		die(logger, "enforce-default-remote", err)
	}

	defaultMasqueradeRules, err := vxlanDefaultMasqueradePlanner.GetRules()
	if err != nil {
		die(logger, "default-masquerade-rules.GetRules", err)
	}
	err = ruleEnforcer.EnforceOnChain(defaultMasqueradeChain, defaultMasqueradeRules)
	if err != nil {
		die(logger, "enforce-default-masquerade", err)
	}

	policyPoller := &poller.Poller{
		Logger:       logger,
		PollInterval: pollInterval,
		Planner:      dynamicPlanner,

		Chain:    dynamicChain,
		Enforcer: ruleEnforcer,
	}

	members := grouper.Members{
		{"policy_poller", policyPoller},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		die(logger, "ifrit monitor", err)
	}
}
