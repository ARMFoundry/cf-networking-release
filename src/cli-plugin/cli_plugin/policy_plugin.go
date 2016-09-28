package cli_plugin

import (
	"cli-plugin/styles"
	"crypto/tls"
	"flag"
	"fmt"
	"lib/policy_client"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry/cli/plugin"
)

type Plugin struct {
	Styler       *styles.StyleGroup
	Logger       *log.Logger
	PolicyClient policy_client.ExternalPolicyClient
}

type ValidArgs struct {
	SourceAppName string
	DestAppName   string
	Protocol      string
	Port          int
}

const AllowCommand = "access-allow"
const ListCommand = "access-list"
const DenyCommand = "access-deny"

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
					Usage: fmt.Sprintf("cf %s SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port [1-65535]", AllowCommand),
					Options: map[string]string{
						"-protocol": "Protocol to connect apps with. (required)",
						"-port":     "Port to connect to destination app with. (required)",
					},
				},
			},
			plugin.Command{
				Name:     ListCommand,
				HelpText: "List policy for direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage:   fmt.Sprintf("cf %s [--app appName]", ListCommand),
					Options: map[string]string{"-app": "Application to filter results by. (optional)"},
				},
			},
			plugin.Command{
				Name:     DenyCommand,
				HelpText: "Remove direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage: fmt.Sprintf("cf %s SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port [1-65535]", DenyCommand),
					Options: map[string]string{
						"-protocol": "Protocol to connect apps with. (required)",
						"-port":     "Port to connect to destination app with. (required)",
					},
				},
			},
		},
	}
}

func (p *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	output, err := p.RunWithErrors(cliConnection, args)
	if err != nil {
		p.Logger.Printf(p.Styler.ApplyStyles(p.Styler.AddStyle("FAILED", "red")))
		p.Logger.Fatalf("%s", err)
	}

	p.Logger.Printf(p.Styler.ApplyStyles(p.Styler.AddStyle("OK\n", "green")))
	p.Logger.Print(p.Styler.ApplyStyles(output))
}

func (p *Plugin) RunWithErrors(cliConnection plugin.CliConnection, args []string) (string, error) {
	apiEndpoint, err := cliConnection.ApiEndpoint()
	if err != nil {
		return "", fmt.Errorf("getting api endpoint: %s", err)
	}
	skipSSL, err := cliConnection.IsSSLDisabled()
	if err != nil {
		return "", fmt.Errorf("checking if ssl disabled: %s", err)
	}

	runner := &CommandRunner{
		Styler: p.Styler,
		Logger: p.Logger,
		PolicyClient: policy_client.NewExternal(
			lager.NewLogger("command"),
			&http.Client{
				Transport: &http.Transport{
					Dial: (&net.Dialer{
						Timeout: 3 * time.Second,
					}).Dial,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: skipSSL,
					},
				},
			},
			apiEndpoint),
		CliConnection: cliConnection,
		Args:          args,
	}

	switch args[0] {
	case AllowCommand:
		return runner.Allow()
	case ListCommand:
		return runner.List()
	case DenyCommand:
		return runner.Deny()
	}

	return "", nil
}

func validateUsage(cliConnection plugin.CliConnection, regex string, args []string) error {
	rx := regexp.MustCompile(regex)
	if !rx.MatchString(strings.Join(args, " ")) {
		return errorWithUsage("", args[0], cliConnection)
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
	err := flags.Parse(args[3:])
	if err != nil {
		return ValidArgs{}, errorWithUsage(err.Error(), args[0], cliConnection)
	}

	port, err := strconv.Atoi(*portString)
	if err != nil {
		return ValidArgs{}, errorWithUsage(fmt.Sprintf("Port is not valid: %s", *portString), args[0], cliConnection)
	}

	validArgs.SourceAppName = srcAppName
	validArgs.DestAppName = dstAppName
	validArgs.Protocol = *protocol
	validArgs.Port = port

	return validArgs, nil
}

func errorWithUsage(errorString, cmd string, cliConnection plugin.CliConnection) error {
	output, err := cliConnection.CliCommandWithoutTerminalOutput("help", cmd)
	if err != nil {
		return fmt.Errorf("cf cli error: %s", err)
	}
	return fmt.Errorf("Incorrect usage. %s\n\n%s", errorString, strings.Join(output, "\n"))
}
