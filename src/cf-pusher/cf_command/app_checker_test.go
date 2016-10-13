package cf_command_test

import (
	"cf-pusher/cf_command"
	"cf-pusher/fakes"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppChecker", func() {
	var (
		appChecker  *cf_command.AppChecker
		fakeAdapter *fakes.CheckCLIAdapter
	)
	BeforeEach(func() {
		fakeAdapter = &fakes.CheckCLIAdapter{}
		appChecker = &cf_command.AppChecker{
			Adapter: fakeAdapter,
		}
	})
	Describe("CheckApps", func() {
		BeforeEach(func() {
			appChecker.Applications = []cf_command.Application{
				{
					Name:      "some-name-1",
					Directory: "some/dir",
				},
			}
			fakeAdapter.AppGuidReturns("some-guid-1", nil)
			str := `{ "guid": "some-guid-1", "name": "scale-tick-1", "running_instances": 2, "instances": 2, "state": "STARTED"}`
			fakeAdapter.CheckAppReturns([]byte(str), nil)

		})
		It("when the app is in state running", func() {
			err := appChecker.CheckApps()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeAdapter.AppGuidCallCount()).To(Equal(1))
			Expect(fakeAdapter.AppGuidArgsForCall(0)).To(Equal("some-name-1"))

			Expect(fakeAdapter.CheckAppCallCount()).To(Equal(1))
			Expect(fakeAdapter.CheckAppArgsForCall(0)).To(Equal("some-guid-1"))
		})

		Context("when check app guid fails", func() {
			BeforeEach(func() {
				fakeAdapter.AppGuidReturns("", errors.New("potato"))
			})
			It("returns a meaningful error", func() {
				err := appChecker.CheckApps()
				Expect(err).To(MatchError("checking app guid some-name-1: potato"))
			})
		})
		Context("when check app fails", func() {
			BeforeEach(func() {
				fakeAdapter.CheckAppReturns(nil, errors.New("potato"))
			})
			It("returns a meaningful error", func() {
				err := appChecker.CheckApps()
				Expect(err).To(MatchError("checking app some-name-1: potato"))
			})
		})
		Context("when the json is malformed", func() {
			BeforeEach(func() {
				str := `{ "guid": "some-guid-1", "name": "scale-tick-1"`
				fakeAdapter.CheckAppReturns([]byte(str), nil)
			})
			It("returns a meaningul error", func() {
				err := appChecker.CheckApps()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when an error response is returned", func() {
			BeforeEach(func() {
				str := ` { "code": 100004,
						   "description": "The app could not be found: guid",
						   "error_code": "CF-AppNotFound"
						}`
				fakeAdapter.CheckAppReturns([]byte(str), nil)
			})
			It("returns a meaningul error", func() {
				err := appChecker.CheckApps()
				Expect(err).To(MatchError("checking app some-name-1: no instances are running"))
			})
		})

		Context("when response is unexpected json or no instances are running", func() {
			BeforeEach(func() {
				str := `{}`
				fakeAdapter.CheckAppReturns([]byte(str), nil)
			})
			It("returns a meaningul error", func() {
				err := appChecker.CheckApps()
				Expect(err).To(MatchError("checking app some-name-1: no instances are running"))
			})
		})

		Context("when one app is not running", func() {
			BeforeEach(func() {
				str := `{ "guid": "some-guid-1", "name": "scale-tick-1", "running_instances": 1, "instances": 2, "state": "STARTED"}`
				fakeAdapter.CheckAppReturns([]byte(str), nil)
			})
			It("returns a meaningul error", func() {
				err := appChecker.CheckApps()
				Expect(err).To(MatchError("checking app some-name-1: not all instances are running"))
			})

		})
	})
})
