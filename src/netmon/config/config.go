package config

type Netmon struct {
	PollInterval  int    `json:"poll_interval"`
	MetronAddress string `json:"metron_address"`
	InterfaceName string `json:"interface_name"`
}
