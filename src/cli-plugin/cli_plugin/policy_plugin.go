package cli_plugin

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/marshal"
	"log"
	"netman-agent/models"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/cloudfoundry/cli/plugin"
)

type Plugin struct {
	Marshaler   marshal.Marshaler
	Unmarshaler marshal.Unmarshaler
}

type ValidArgs struct {
	SourceAppName string
	DestAppName   string
	Protocol      string
	Port          int
}

const AllowCommand = "allow-access"
const ListCommand = "list-access"
const DenyCommand = "deny-access"

var STYLES = map[string]string{
	"<RESET>": "\x1B[0m",
	"<CLR_R>": "\x1B[31;1m",
	"<CLR_G>": "\x1B[32;1m",
	"<CLR_C>": "\x1B[36;1m",
	"<BOLD>":  "\x1B[;1m",
}

var ListUsageRegex = fmt.Sprintf(`\A%s\s*(--app(\s+|=)\S+\z|\z)`, ListCommand)
var AllowUsageRegex = fmt.Sprintf(`\A%s\s+\S+\s+\S+\s+(--|-)\w+(\s+|=)\w+\s+(--|-)\w+(\s+|=)\w+\z`, AllowCommand)
var DenyUsageRegex = fmt.Sprintf(`\A%s\s+\S+\s+\S+\s+(--|-)\w+(\s+|=)\w+\s+(--|-)\w+(\s+|=)\w+\z`, DenyCommand)

func (p *Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "network-policy",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 15,
		},
		Commands: []plugin.Command{
			plugin.Command{
				Name:     AllowCommand,
				HelpText: "Allow direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage: "cf allow-access SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port [1-65535]",
					Options: map[string]string{
						"--protocol": "Protocol to connect apps with. (required)",
						"--port":     "Port to connect to destination app with. (required)",
					},
				},
			},
			plugin.Command{
				Name:     ListCommand,
				HelpText: "List policy for direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage:   "cf list-access [--app appName]",
					Options: map[string]string{"--app": "Application to filter results by. (optional)"},
				},
			},
			plugin.Command{
				Name:     DenyCommand,
				HelpText: "Remove direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage: "cf deny-access SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port [1-65535]",
					Options: map[string]string{
						"--protocol": "Protocol to connect apps with. (required)",
						"--port":     "Port to connect to destination app with. (required)",
					},
				},
			},
		},
	}
}

func (p *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	logger := log.New(os.Stdout, "", 0)

	output, err := p.RunWithErrors(cliConnection, args)
	if err != nil {
		logger.Printf("%sFAILED%s", STYLES["<CLR_R>"], STYLES["<RESET>"])
		logger.Fatalf("%s", err)
	}

	logger.Printf("%sOK%s\n", STYLES["<CLR_G>"], STYLES["<RESET>"])
	logger.Print(output)
}

func (p *Plugin) RunWithErrors(cliConnection plugin.CliConnection, args []string) (string, error) {
	switch args[0] {
	case AllowCommand:
		return p.AllowCommand(cliConnection, args)
	case ListCommand:
		return p.ListCommand(cliConnection, args)
	case DenyCommand:
		return p.DenyCommand(cliConnection, args)
	}

	return "", nil
}

func (p *Plugin) ListCommand(cliConnection plugin.CliConnection, args []string) (string, error) {
	err := validateUsage(cliConnection, ListUsageRegex, args)
	if err != nil {
		return "", err
	}

	flags := flag.NewFlagSet("cf list-policies", flag.ContinueOnError)
	appName := flags.String("app", "", "app name to filter results")
	flags.Parse(args[1:])

	path := "/networking/v0/external/policies"
	if *appName != "" {
		app, err := cliConnection.GetApp(*appName)
		if err != nil {
			return "", fmt.Errorf("getting app: %s", err)
		}
		path = fmt.Sprintf("%s?id=%s", path, app.Guid)
	}

	apps, err := cliConnection.GetApps()
	if err != nil {
		return "", fmt.Errorf("getting apps: %s", err)
	}

	var policiesResponse = struct {
		Policies []models.Policy `json:"policies"`
	}{}

	policiesJSON, err := cliConnection.CliCommandWithoutTerminalOutput("curl", path)
	if err != nil {
		return "", fmt.Errorf("getting policies: %s", err)
	}

	buffer := &bytes.Buffer{}
	tabWriter := tabwriter.NewWriter(buffer, 0, 8, 2, '\t', tabwriter.FilterHTML)
	fmt.Fprintf(tabWriter, "%sSource\tDestination\tProtocol\tPort%s\n", STYLES["<BOLD>"], STYLES["<RESET>"])

	err = p.Unmarshaler.Unmarshal([]byte(policiesJSON[0]), &policiesResponse)
	if err != nil {
		return "", fmt.Errorf("unmarshaling: %s", err)
	}
	policies := policiesResponse.Policies

	for _, policy := range policies {
		srcName := ""
		dstName := ""
		for _, app := range apps {
			if policy.Source.ID == app.Guid {
				srcName = app.Name
			}
			if policy.Destination.ID == app.Guid {
				dstName = app.Name
			}
		}
		if srcName != "" && dstName != "" {
			fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%d\n", srcName, dstName, policy.Destination.Protocol, policy.Destination.Port)
		}
	}

	tabWriter.Flush()
	outBytes, err := ioutil.ReadAll(buffer)
	if err != nil {
		//untested
		return "", fmt.Errorf("formatting output: %s", err)
	}

	return string(outBytes), nil
}

func (p *Plugin) AllowCommand(cliConnection plugin.CliConnection, args []string) (string, error) {
	err := validateUsage(cliConnection, AllowUsageRegex, args)
	if err != nil {
		return "", err
	}

	validArgs, err := ValidateArgs(cliConnection, args)
	if err != nil {
		return "", err
	}

	srcAppModel, err := cliConnection.GetApp(validArgs.SourceAppName)
	if err != nil {
		return "", fmt.Errorf("resolving source app: %s", err)
	}
	if srcAppModel.Guid == "" {
		return "", fmt.Errorf("resolving source app: %s not found", validArgs.SourceAppName)
	}
	dstAppModel, err := cliConnection.GetApp(validArgs.DestAppName)
	if err != nil {
		return "", fmt.Errorf("resolving destination app: %s", err)
	}
	if dstAppModel.Guid == "" {
		return "", fmt.Errorf("resolving destination app: %s not found", validArgs.DestAppName)
	}

	policy := models.Policy{
		Source: models.Source{
			ID: srcAppModel.Guid,
		},
		Destination: models.Destination{
			ID:       dstAppModel.Guid,
			Protocol: validArgs.Protocol,
			Port:     validArgs.Port,
		},
	}

	var policies = struct {
		Policies []models.Policy `json:"policies"`
	}{
		[]models.Policy{policy},
	}

	payload, err := p.Marshaler.Marshal(policies)
	if err != nil {
		return "", fmt.Errorf("payload cannot be marshaled: %s", err)
	}

	output, err := cliConnection.CliCommandWithoutTerminalOutput(
		"curl", "-X", "POST", "/networking/v0/external/policies", "-d", "'"+string(payload)+"'",
	)
	if err != nil {
		return "", fmt.Errorf("policy creation failed: %s", err)
	}

	if output[0] != "{}" {
		var policyError struct {
			Error string `json:"error"`
		}
		err = p.Unmarshaler.Unmarshal([]byte(output[0]), &policyError)
		if err != nil {
			return "", fmt.Errorf("error unmarshaling policy response: %s", err)
		}
		return "", fmt.Errorf("error creating policy: %s", policyError.Error)
	}

	return "", nil
}

func (p *Plugin) DenyCommand(cliConnection plugin.CliConnection, args []string) (string, error) {
	err := validateUsage(cliConnection, DenyUsageRegex, args)
	if err != nil {
		return "", err
	}

	validArgs, err := ValidateArgs(cliConnection, args)
	if err != nil {
		return "", err
	}

	srcAppModel, err := cliConnection.GetApp(validArgs.SourceAppName)
	if err != nil {
		return "", fmt.Errorf("resolving source app: %s", err)
	}
	if srcAppModel.Guid == "" {
		return "", fmt.Errorf("resolving source app: %s not found", validArgs.SourceAppName)
	}
	dstAppModel, err := cliConnection.GetApp(validArgs.DestAppName)
	if err != nil {
		return "", fmt.Errorf("resolving destination app: %s", err)
	}
	if dstAppModel.Guid == "" {
		return "", fmt.Errorf("resolving destination app: %s not found", validArgs.DestAppName)
	}

	policy := models.Policy{
		Source: models.Source{
			ID: srcAppModel.Guid,
		},
		Destination: models.Destination{
			ID:       dstAppModel.Guid,
			Protocol: validArgs.Protocol,
			Port:     validArgs.Port,
		},
	}

	var policies = struct {
		Policies []models.Policy `json:"policies"`
	}{
		[]models.Policy{policy},
	}

	payload, err := p.Marshaler.Marshal(policies)
	if err != nil {
		return "", fmt.Errorf("payload cannot be marshaled: %s", err)
	}

	_, err = cliConnection.CliCommandWithoutTerminalOutput(
		"curl", "-X", "DELETE", "/networking/v0/external/policies", "-d", "'"+string(payload)+"'",
	)
	if err != nil {
		return "", fmt.Errorf("policy creation failed: %s", err)
	}

	return "", nil
}

func validateUsage(cliConnection plugin.CliConnection, regex string, args []string) error {
	rx := regexp.MustCompile(regex)
	if !rx.MatchString(strings.Join(args, " ")) {
		output, err := cliConnection.CliCommandWithoutTerminalOutput("help", args[0])
		if err != nil {
			panic(err)
		}
		return errors.New(strings.Join(output, "\n"))
	}
	return nil
}

func ValidateArgs(cliConnection plugin.CliConnection, args []string) (ValidArgs, error) {
	validArgs := ValidArgs{}

	srcAppName := args[1]
	dstAppName := args[2]

	flags := flag.NewFlagSet("cf "+args[0]+" <src> <dest>", flag.ContinueOnError)
	protocol := flags.String("protocol", "", "the protocol allowed")
	portString := flags.String("port", "", "the destination port")
	flags.Parse(args[3:])

	port, err := strconv.Atoi(*portString)
	if err != nil {
		output, err := cliConnection.CliCommandWithoutTerminalOutput("help", args[0])
		if err != nil {
			panic(err)
		}
		usageError := fmt.Sprintf("Incorrect usage. Port is not valid: %s\n\n%s", *portString, strings.Join(output, "\n"))
		return ValidArgs{}, errors.New(usageError)
	}

	validArgs.SourceAppName = srcAppName
	validArgs.DestAppName = dstAppName
	validArgs.Protocol = *protocol
	validArgs.Port = port

	return validArgs, nil
}
