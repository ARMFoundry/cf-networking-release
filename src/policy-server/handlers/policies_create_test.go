package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	lfakes "lib/fakes"
	"lib/testsupport"
	"net/http"
	"net/http/httptest"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PoliciesCreate", func() {
	var (
		requestJSON       string
		request           *http.Request
		handler           *handlers.PoliciesCreate
		resp              *httptest.ResponseRecorder
		fakeStore         *fakes.Store
		fakeValidator     *fakes.Validator
		fakePolicyGuard   *fakes.PolicyGuard
		logger            *lagertest.TestLogger
		fakeUnmarshaler   *lfakes.Unmarshaler
		fakeMetricsSender *fakes.MetricsSender
		tokenData         uaa_client.CheckTokenResponse
	)

	BeforeEach(func() {
		var err error
		requestJSON = `{"policies": [
			{
				"source": {
					"id": "some-app-guid"
				},
				"destination": {
					"id": "some-other-app-guid",
					"protocol": "tcp",
					"port": 8080
				}
			},
			{
				"source": {
					"id": "another-app-guid"
				},
				"destination": {
					"id": "some-other-app-guid",
					"protocol": "udp",
					"port": 1234
				}
			}
        ]}`
		request, err = http.NewRequest("POST", "/networking/v0/external/policies", bytes.NewBuffer([]byte(requestJSON)))
		Expect(err).NotTo(HaveOccurred())

		fakeStore = &fakes.Store{}
		fakeValidator = &fakes.Validator{}
		fakePolicyGuard = &fakes.PolicyGuard{}
		logger = lagertest.NewTestLogger("test")
		fakeUnmarshaler = &lfakes.Unmarshaler{}
		fakeMetricsSender = &fakes.MetricsSender{}
		fakeUnmarshaler.UnmarshalStub = json.Unmarshal
		handler = &handlers.PoliciesCreate{
			Logger:        logger,
			Store:         fakeStore,
			Unmarshaler:   fakeUnmarshaler,
			Validator:     fakeValidator,
			PolicyGuard:   fakePolicyGuard,
			MetricsSender: fakeMetricsSender,
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.admin"},
			UserName: "some_user",
		}
		fakePolicyGuard.CheckAccessReturns(true, nil)
		resp = httptest.NewRecorder()
	})

	It("persists a new policy rule", func() {
		expectedPolicies := []models.Policy{{
			Source: models.Source{ID: "some-app-guid"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Protocol: "tcp",
				Port:     8080,
			},
		}, {
			Source: models.Source{ID: "another-app-guid"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Protocol: "udp",
				Port:     1234,
			},
		}}

		handler.ServeHTTP(resp, request, tokenData)

		Expect(fakeUnmarshaler.UnmarshalCallCount()).To(Equal(1))
		bodyBytes, _ := fakeUnmarshaler.UnmarshalArgsForCall(0)
		Expect(bodyBytes).To(Equal([]byte(requestJSON)))
		Expect(fakeValidator.ValidatePoliciesCallCount()).To(Equal(1))
		Expect(fakeValidator.ValidatePoliciesArgsForCall(0)).To(Equal(expectedPolicies))
		Expect(fakePolicyGuard.CheckAccessCallCount()).To(Equal(1))
		policies, token := fakePolicyGuard.CheckAccessArgsForCall(0)
		Expect(policies).To(Equal(expectedPolicies))
		Expect(token).To(Equal(tokenData))
		Expect(fakeStore.CreateCallCount()).To(Equal(1))
		Expect(fakeStore.CreateArgsForCall(0)).To(Equal(expectedPolicies))
		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON("{}"))
	})

	It("logs the policy with username and app guid", func() {
		handler.ServeHTTP(resp, request, tokenData)
		Expect(logger).To(gbytes.Say("policy-create.*some-app-guid.*some_user"))
	})

	Context("when the policy guard returns false", func() {
		BeforeEach(func() {
			fakePolicyGuard.CheckAccessReturns(false, nil)
		})

		It("responds with code 403", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusForbidden))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "one or more applications cannot be found or accessed"}`))
		})

		It("logs the failure", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("check-access-failed.*one or more applications cannot be found or accessed"))
		})
	})

	Context("when the validator fails", func() {
		BeforeEach(func() {
			fakeValidator.ValidatePoliciesReturns(errors.New("banana"))
		})

		It("responds with code 400 and a useful error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "banana"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("bad-request.*banana"))
		})
	})

	Context("when the policy guard returns an error", func() {
		BeforeEach(func() {
			fakePolicyGuard.CheckAccessReturns(false, errors.New("banana"))
		})

		It("responds with code 500 and a useful error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "banana"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("check-access-failed.*banana"))
		})

		It("increments the error counter", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
			Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("ExternalPoliciesCreateError"))
		})
	})

	Context("when the store Create call returns an error", func() {
		BeforeEach(func() {
			fakeStore.CreateReturns(errors.New("banana"))
		})

		It("sets a 500 error code, and returns a generic error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "database create failed"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("store-create-failed.*banana"))
		})

		It("increments the error counter", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(fakeMetricsSender.IncrementCounterCallCount()).To(Equal(1))
			Expect(fakeMetricsSender.IncrementCounterArgsForCall(0)).To(Equal("ExternalPoliciesCreateError"))
		})
	})

	Context("when there are errors reading the body bytes", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(&testsupport.BadReader{})
		})

		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "invalid request body"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("body-read-failed.*banana"))
		})
	})

	Context("when there are errors in the request body formatting", func() {
		BeforeEach(func() {
			request.Body = ioutil.NopCloser(bytes.NewReader([]byte(`{"policies":{}}`)))
		})

		It("returns a descriptive error", func() {
			handler.ServeHTTP(resp, request, tokenData)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Body.String()).To(MatchJSON(`{"error": "invalid values passed to API"}`))
		})

		It("logs the full error", func() {
			handler.ServeHTTP(resp, request, tokenData)
			Expect(logger).To(gbytes.Say("unmarshal-failed.*json: cannot unmarshal"))
		})
	})
})
