package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"natman/config"
	"natman/planner"
	"natman/poller"
	"netman-agent/rules"
	"os"
	"time"

	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"

	"code.cloudfoundry.org/lager"
	"github.com/coreos/go-iptables/iptables"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	conf := &config.Natman{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("natman")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatal("error reading config")
	}

	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		log.Fatal("error unmarshalling config")
	}
	logger.Info("parsed-config", lager.Data{"config": conf})

	pollInterval := time.Duration(conf.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = time.Second
	}

	gardenClient := client.New(connection.New(conf.GardenProtocol, conf.GardenAddress))

	netInPlanner := &planner.NetInPlanner{
		GardenClient: gardenClient,
	}

	netOutPlanner := &planner.NetOutPlanner{
		GardenClient:   gardenClient,
		OverlayNetwork: conf.OverlayNetwork,
	}

	ipt, err := iptables.New()
	if err != nil {
		logger.Fatal("iptables-new", err)
	}

	timestamper := &rules.Timestamper{}
	ruleEnforcer := rules.NewEnforcer(
		logger.Session("rules-enforcer"),
		timestamper,
		ipt,
	)

	netInChain := rules.Chain{
		Table:       "nat",
		ParentChain: "PREROUTING",
		Prefix:      "natman--netin-",
	}

	netOutChain := rules.Chain{
		Table:       "filter",
		ParentChain: "FORWARD",
		Prefix:      "natman--netout-",
	}

	gardenNetInPoller := &poller.Poller{
		Logger:       logger,
		PollInterval: pollInterval,
		Planner:      netInPlanner,
		Enforcer:     ruleEnforcer,
		Chain:        netInChain,
	}

	gardenNetOutPoller := &poller.Poller{
		Logger:       logger,
		PollInterval: pollInterval,
		Planner:      netOutPlanner,
		Enforcer:     ruleEnforcer,
		Chain:        netOutChain,
	}

	members := grouper.Members{
		{"garden_net_in_poller", gardenNetInPoller},
		{"garden_net_out_poller", gardenNetOutPoller},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		logger.Fatal("ifrit monitor", err)
	}
}
