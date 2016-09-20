package policy_client_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"lib/fakes"
	"lib/policy_client"
	"lib/testsupport"
	"net/http"
	"strings"

	lfakes "lib/fakes"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("JsonClient", func() {
	var (
		jsonClient       *policy_client.JsonClient
		httpClient       *fakes.HTTPClient
		fakeUnmarshaler  *lfakes.Unmarshaler
		fakeMarshaler    *lfakes.Marshaler
		returnedResponse *http.Response
		logger           *lagertest.TestLogger
		method           string
		route            string
		reqData          map[string]string
		respData         map[string]string
	)

	BeforeEach(func() {
		httpClient = &fakes.HTTPClient{}
		fakeUnmarshaler = &lfakes.Unmarshaler{}
		fakeUnmarshaler.UnmarshalStub = json.Unmarshal
		fakeMarshaler = &lfakes.Marshaler{}
		fakeMarshaler.MarshalStub = json.Marshal
		logger = lagertest.NewTestLogger("test")
		jsonClient = &policy_client.JsonClient{
			Logger:      logger,
			HttpClient:  httpClient,
			Url:         "http://some.url",
			Marshaler:   fakeMarshaler,
			Unmarshaler: fakeUnmarshaler,
		}

		returnedResponse = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(strings.NewReader(`{
				"some-key" : "some-value"
			}`)),
		}
		method = "GET"
		route = "/some/route"
		reqData = map[string]string{"request": "data"}
		respData = map[string]string{}
		httpClient.DoReturns(returnedResponse, nil)
	})
	Describe("Do", func() {
		It("makes a request with the given body", func() {
			err := jsonClient.Do(method, route, reqData, &respData)
			Expect(err).NotTo(HaveOccurred())

			Expect(httpClient.DoCallCount()).To(Equal(1))
			receivedRequest := httpClient.DoArgsForCall(0)
			Expect(receivedRequest.Method).To(Equal("GET"))
			Expect(receivedRequest.URL.Host).To(Equal("some.url"))
			Expect(receivedRequest.URL.Path).To(Equal("/some/route"))
			bodyBytes, err := ioutil.ReadAll(receivedRequest.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(bodyBytes).To(MatchJSON(`{"request":"data"}`))

			Expect(respData).To(Equal(map[string]string{"some-key": "some-value"}))
			Expect(logger).To(gbytes.Say(`http-do.*some-key.*some-value`))
		})

		Context("when marshaling the request data to json fails", func() {
			BeforeEach(func() {
				fakeMarshaler.MarshalStub = nil
				fakeMarshaler.MarshalReturns(nil, errors.New("banana"))
			})
			It("returns an error", func() {
				err := jsonClient.Do(method, route, reqData, &respData)
				Expect(err).To(MatchError("json marshal request body: banana"))
			})
		})

		Context("when forming the http request fails", func() {
			BeforeEach(func() {
				jsonClient.Url = "%%%%"
			})
			It("returns an error", func() {
				err := jsonClient.Do(method, route, reqData, &respData)
				Expect(err).To(MatchError(HavePrefix("http new request: parse")))
			})
		})

		Context("when doing the request fails", func() {
			BeforeEach(func() {
				httpClient.DoReturns(nil, errors.New("banana"))
			})
			It("returns the error", func() {
				err := jsonClient.Do(method, route, reqData, &respData)
				Expect(err).To(MatchError("http client do: banana"))
			})
		})

		Context("when the body read fails", func() {
			BeforeEach(func() {
				returnedResponse = &http.Response{
					StatusCode: 200,
					Body:       &testsupport.BadReader{},
				}
				httpClient.DoReturns(returnedResponse, nil)
			})
			It("returns the error", func() {
				err := jsonClient.Do(method, route, reqData, &respData)
				Expect(err).To(MatchError("body read: banana"))
			})
		})

		Context("when the request returns a non 2xx status code", func() {
			BeforeEach(func() {
				returnedResponse.Body = ioutil.NopCloser(strings.NewReader(`{"error":"some-error"}`))
				returnedResponse.StatusCode = http.StatusBadRequest
				httpClient.DoReturns(returnedResponse, nil)
			})

			It("returns the error and logs the body", func() {
				err := jsonClient.Do(method, route, reqData, &respData)
				Expect(err).To(MatchError("http client do: bad response status 400"))

				Expect(logger).To(gbytes.Say(`http-client.*some-error.*400`))
			})
		})

		Context("when the json unmarshal fails", func() {
			BeforeEach(func() {
				fakeUnmarshaler.UnmarshalStub = func([]byte, interface{}) error {
					return errors.New("grapes")
				}
			})
			It("returns and logs the error", func() {
				err := jsonClient.Do(method, route, reqData, &respData)
				Expect(err).To(MatchError("json unmarshal: grapes"))
			})
		})

	})
})
