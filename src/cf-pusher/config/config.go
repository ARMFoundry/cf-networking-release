package config

type Config struct {
	Api               string `json:"api"`
	AdminUser         string `json:"admin_user"`
	AdminPassword     string `json:"admin_password"`
	AppsDomain        string `json:"apps_domain"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	Applications      int    `json:"test_applications"`
	AppInstances      int    `json:"test_app_instances"`
	Policies          int    `json:"test_policies"`
	ProxyInstances    int    `json:"proxy_instances"`
	Concurrency       int    `json:"concurrency"`
}
