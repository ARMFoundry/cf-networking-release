package planner

import (
	"errors"
	"lib/datastore"
	"lib/models"
	"lib/rules"
	"time"
	"vxlan-policy-agent/agent_metrics"
	"vxlan-policy-agent/enforcer"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/policy_client.go --fake-name PolicyClient . policyClient
type policyClient interface {
	GetPolicies() ([]models.Policy, error)
}

type VxlanPolicyPlanner struct {
	Logger            lager.Logger
	Datastore         datastore.Datastore
	PolicyClient      policyClient
	VNI               int
	CollectionEmitter agent_metrics.TimeMetricsEmitter
	Chain             enforcer.Chain
}

type Container struct {
	Handle  string
	IP      string
	GroupID string
}

var missingPolicyGroupIdError error = errors.New("Container metadata is missing key policy_group_id. Check version of CloudController.")

func (p *VxlanPolicyPlanner) getContainersMap(allContainers map[string]datastore.Container) (map[string][]string, error) {
	containers := map[string][]string{}
	for _, container := range allContainers {
		if container.Metadata == nil {
			continue
		}
		groupID, ok := container.Metadata["policy_group_id"].(string)
		if !ok {
			p.Logger.Error("container-metadata-policy-group-id", missingPolicyGroupIdError, lager.Data{"container_handle": container.Handle})
			continue
		}
		containers[groupID] = append(containers[groupID], container.IP)
	}
	return containers, nil
}

func (p *VxlanPolicyPlanner) GetRules() (enforcer.RulesWithChain, error) {
	containerMetadataStartTime := time.Now()
	containerMetadata, err := p.Datastore.ReadAll()
	if err != nil {
		p.Logger.Error("datastore", err)
		return enforcer.RulesWithChain{}, err
	}

	containers, err := p.getContainersMap(containerMetadata)
	if err != nil {
		p.Logger.Error("container-info", err)
		return enforcer.RulesWithChain{}, err
	}
	containerMetadataDuration := time.Now().Sub(containerMetadataStartTime)
	p.Logger.Debug("got-containers", lager.Data{"containers": containers})

	policyServerStartRequestTime := time.Now()
	policies, err := p.PolicyClient.GetPolicies()
	if err != nil {
		p.Logger.Error("policy-client-get-policies", err)
		return enforcer.RulesWithChain{}, err
	}
	policyServerPollDuration := time.Now().Sub(policyServerStartRequestTime)
	p.CollectionEmitter.EmitAll(map[string]time.Duration{
		agent_metrics.MetricContainerMetadata: containerMetadataDuration,
		agent_metrics.MetricPolicyServerPoll:  policyServerPollDuration,
	})

	marksRuleset := []rules.IPTablesRule{}
	filterRuleset := []rules.IPTablesRule{}

	for _, policy := range policies {
		srcContainerIPs, srcOk := containers[policy.Source.ID]
		dstContainerIPs, dstOk := containers[policy.Destination.ID]

		if dstOk {
			// there are some containers on this host that are dests for the policy
			for _, dstContainerIP := range dstContainerIPs {
				filterRuleset = append(
					filterRuleset,
					rules.NewMarkAllowRule(
						dstContainerIP,
						policy.Destination.Protocol,
						policy.Destination.Port,
						policy.Source.Tag,
						policy.Source.ID,
						policy.Destination.ID,
					),
				)
			}
		}

		if srcOk {
			// there are some containers on this host that are sources for the policy
			for _, srcContainerIP := range srcContainerIPs {
				marksRuleset = append(
					marksRuleset,
					rules.NewMarkSetRule(srcContainerIP, policy.Source.Tag, policy.Source.ID),
				)
			}
		}
	}
	ruleset := append(marksRuleset, filterRuleset...)
	p.Logger.Debug("generated-rules", lager.Data{"rules": ruleset})
	return enforcer.RulesWithChain{
		Chain: p.Chain,
		Rules: ruleset,
	}, nil
}
