package api_test

import (
	hfakes "code.cloudfoundry.org/cf-networking-helpers/fakes"
	"encoding/json"
	"errors"
	"policy-server/api"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressPolicyMapper", func() {
	var mapper *api.EgressPolicyMapper

	BeforeEach(func() {
		mapper = &api.EgressPolicyMapper{
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
			Marshaler:   marshal.MarshalFunc(json.Marshal),
		}
	})

	Describe("AsStoreEgressPolicy", func() {
		It("maps a payload with api.EgressPolicy to a slice of store.EgressPolicy", func() {
			payloadBytes := []byte(`{
				"egress_policies": [
                    {
						"source": { "id": "some-src-id", "type": "app" },
						"destination": { "id": "some-dst-id" }
					},
                    {
						"source": { "id": "some-src-id-2", "type": "space"  },
						"destination": { "id": "some-dst-id-2" }
					}
				]
			}`)

			policies, err := mapper.AsStoreEgressPolicy(payloadBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(policies).To(HaveLen(2))
			Expect(policies[0].Source.ID).To(Equal("some-src-id"))
			Expect(policies[0].Source.Type).To(Equal("app"))
			Expect(policies[0].Destination.GUID).To(Equal("some-dst-id"))
			Expect(policies[1].Source.ID).To(Equal("some-src-id-2"))
			Expect(policies[1].Source.Type).To(Equal("space"))
			Expect(policies[1].Destination.GUID).To(Equal("some-dst-id-2"))
		})

		Context("when unmarshalling fails", func() {
			It("wraps and returns an error", func() {
				_, err := mapper.AsStoreEgressPolicy([]byte("garbage"))
				Expect(err).To(MatchError(errors.New("unmarshal json: invalid character 'g' looking for beginning of value")))
			})
		})
	})

	Describe("AsBytes", func() {
		var egressPolicies []store.EgressPolicy

		BeforeEach(func() {
			egressPolicies = []store.EgressPolicy{
				{
					Source:      store.EgressSource{ID: "some-src-id", Type: "app"},
					Destination: store.EgressDestination{GUID: "some-dst-id"},
				},
				{
					Source:      store.EgressSource{ID: "some-src-id-2", Type: "space"},
					Destination: store.EgressDestination{GUID: "some-dst-id-2"},
				},
			}
		})

		It("maps a payload with api.EgressPolicy to a slice of store.EgressPolicy", func() {
			mappedBytes, err := mapper.AsBytes(egressPolicies)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(mappedBytes)).To(MatchJSON(`{
					"total_egress_policies": 2,
					"egress_policies": [
            	        {
							"source": { "id": "some-src-id", "type": "app" },
							"destination": { "id": "some-dst-id" }
						},
               	    	{
							"source": { "id": "some-src-id-2", "type": "space" },
							"destination": { "id": "some-dst-id-2" }
						}
					]
				}`))
		})

		Context("when marshalling fails", func() {
			BeforeEach(func() {
				marshaler := &hfakes.Marshaler{}
				marshaler.MarshalReturns([]byte{}, errors.New("failed to marshal bytes"))
				mapper.Marshaler = marshaler
			})

			It("wraps and returns an error", func() {
				_, err := mapper.AsBytes(egressPolicies)
				Expect(err).To(MatchError(errors.New("marshal json: failed to marshal bytes")))
			})
		})
	})
})
