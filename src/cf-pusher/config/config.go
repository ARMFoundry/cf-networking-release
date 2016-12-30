package config

type Config struct {
	Api               string `json:"api"`
	AdminUser         string `json:"admin_user"`
	AdminPassword     string `json:"admin_password"`
	AdminSecret       string `json:"admin_secret"`
	AppsDomain        string `json:"apps_domain"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	Applications      int    `json:"test_applications"`
	AppInstances      int    `json:"test_app_instances"`
	ExtraListenPorts  int    `json:"extra_listen_ports"`
	ProxyInstances    int    `json:"proxy_instances"`
	Concurrency       int    `json:"concurrency"`
	Prefix            string `json:"prefix"`
	SampleSize        int    `json:"sample_size"`
	ASGSize           int    `json:"asg_size"`
}
