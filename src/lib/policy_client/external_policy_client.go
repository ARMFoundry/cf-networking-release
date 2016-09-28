package policy_client

import (
	"encoding/json"
	"lib/marshal"
	"lib/models"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/external_policy_client.go --fake-name ExternalPolicyClient . ExternalPolicyClient
type ExternalPolicyClient interface {
	GetPolicies() ([]models.Policy, error)
	DeletePolicies(policies []models.Policy, token string) error
	AddPolicies(policies []models.Policy, token string) error
}

type ExternalClient struct {
	JsonClient jsonClient
}

func NewExternal(logger lager.Logger, httpClient httpClient, url string) *ExternalClient {
	return &ExternalClient{
		JsonClient: &JsonClient{
			Logger:      logger,
			HttpClient:  httpClient,
			Url:         url,
			Marshaler:   marshal.MarshalFunc(json.Marshal),
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		},
	}
}

func (c *ExternalClient) GetPolicies() ([]models.Policy, error) {
	var policies struct {
		Policies []models.Policy `json:"policies"`
	}
	err := c.JsonClient.Do("GET", "/networking/v0/external/policies", nil, &policies, "")
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}

func (c *ExternalClient) AddPolicies(policies []models.Policy, token string) error {
	reqPolicies := map[string][]models.Policy{
		"policies": policies,
	}
	err := c.JsonClient.Do("POST", "/networking/v0/external/policies", reqPolicies, nil, token)
	if err != nil {
		return err
	}
	return nil
}

func (c *ExternalClient) DeletePolicies(policies []models.Policy, token string) error {
	reqPolicies := map[string][]models.Policy{
		"policies": policies,
	}
	err := c.JsonClient.Do("DELETE", "/networking/v0/external/policies", reqPolicies, nil, token)
	if err != nil {
		return err
	}
	return nil
}
