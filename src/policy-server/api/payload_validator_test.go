package api_test

import (
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"policy-server/api"
	"policy-server/api/fakes"
)

var _ = Describe("PayloadValidator", func() {

	var (
		validator             *api.PayloadValidator
		payload               *api.PoliciesPayload
		policyValidator       *fakes.Validator
		egressPolicyValidator *fakes.EgressValidator
	)
	BeforeEach(func() {
		policyValidator = &fakes.Validator{}
		egressPolicyValidator = &fakes.EgressValidator{}

		validator = &api.PayloadValidator{
			PolicyValidator:       policyValidator,
			EgressPolicyValidator: egressPolicyValidator,
		}
		payload = &api.PoliciesPayload{
			Policies: []api.Policy{
				{},
			},
			EgressPolicies: []api.EgressPolicy{
				{},
			},
		}

	})

	It("returns no error if the payload is valid", func() {
		Expect(validator.ValidatePayload(payload)).To(Succeed())
	})

	It("returns an error if both policy and egress policies are empty", func() {
		err := validator.ValidatePayload(&api.PoliciesPayload{})
		Expect(err).To(MatchError("expected policy or egress policy"))
	})

	It("returns delegates to the policy validator and the egress policy validator", func() {
		validator.ValidatePayload(payload)
		Expect(policyValidator.ValidatePoliciesCallCount()).To(Equal(1))
		Expect(policyValidator.ValidatePoliciesArgsForCall(0)).To(Equal(payload.Policies))

		Expect(egressPolicyValidator.ValidateEgressPoliciesCallCount()).To(Equal(1))
		Expect(egressPolicyValidator.ValidateEgressPoliciesArgsForCall(0)).To(Equal(payload.EgressPolicies))
	})

	It("returns an error if the policy validator returns an error", func() {
		policyValidator.ValidatePoliciesReturns(errors.New("policy validator error"))
		Expect(validator.ValidatePayload(payload)).To(MatchError("policy validator error"))
	})

	It("returns an error if the egress policy validator returns an error", func() {
		egressPolicyValidator.ValidateEgressPoliciesReturns(errors.New("egress policy validator error"))
		Expect(validator.ValidatePayload(payload)).To(MatchError("egress policy validator error"))
	})

	It("does not invoke policy validator if policies are empty", func() {
		payload.Policies = []api.Policy{}
		validator.ValidatePayload(payload)
		Expect(policyValidator.ValidatePoliciesCallCount()).To(Equal(0))
	})

	It("does not invoke egress policy validator if egress policies are empty", func() {
		payload.EgressPolicies = []api.EgressPolicy{}
		validator.ValidatePayload(payload)
		Expect(egressPolicyValidator.ValidateEgressPoliciesCallCount()).To(Equal(0))
	})
})
