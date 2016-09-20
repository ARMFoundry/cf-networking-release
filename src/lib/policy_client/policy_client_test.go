package policy_client_test

import (
	"encoding/json"
	"errors"
	"lib/fakes"
	"lib/models"
	"lib/policy_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyClient", func() {
	var (
		client     *policy_client.Client
		jsonClient *fakes.JSONClient
	)

	BeforeEach(func() {
		jsonClient = &fakes.JSONClient{}
		client = &policy_client.Client{
			JsonClient: jsonClient,
		}
	})

	Describe("GetPolicies", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}) error {
				respBytes := []byte(`{ "policies": [ {"source": { "id": "some-app-guid", "tag": "BEEF" }, "destination": { "id": "some-other-app-guid", "protocol": "tcp", "port": 8090 } } ] }`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})
		It("does the right json http client request", func() {
			policies, err := client.GetPolicies()
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _ := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("GET"))
			Expect(route).To(Equal("/networking/v0/internal/policies"))
			Expect(reqData).To(BeNil())

			Expect(policies).To(Equal([]models.Policy{
				{
					Source: models.Source{
						ID:  "some-app-guid",
						Tag: "BEEF",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			},
			))
		})

		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.GetPolicies()
				Expect(err).To(MatchError("banana"))
			})
		})
	})

	Describe("AddPolicies", func() {
		BeforeEach(func() {
			jsonClient.DoStub = func(method, route string, reqData, respData interface{}) error {
				respBytes := []byte(`{}`)
				json.Unmarshal(respBytes, respData)
				return nil
			}
		})
		It("does the right json http client request", func() {
			err := client.AddPolicies([]models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonClient.DoCallCount()).To(Equal(1))
			method, route, reqData, _ := jsonClient.DoArgsForCall(0)
			Expect(method).To(Equal("POST"))
			Expect(route).To(Equal("/networking/v0/external/policies"))
			Expect(reqData).To(Equal([]models.Policy{
				{
					Source: models.Source{
						ID: "some-app-guid",
					},
					Destination: models.Destination{
						ID:       "some-other-app-guid",
						Port:     8090,
						Protocol: "tcp",
					},
				},
			},
			))
		})
		Context("when the json client fails", func() {
			BeforeEach(func() {
				jsonClient.DoReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := client.GetPolicies()
				Expect(err).To(MatchError("banana"))
			})
		})
	})
})
