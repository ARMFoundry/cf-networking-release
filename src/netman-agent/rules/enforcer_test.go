package rules_test

import (
	"errors"
	"netman-agent/fakes"
	"netman-agent/rules"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Enforcer", func() {
	Describe("Enforce", func() {
		var (
			fakeRule    *fakes.Rule
			iptables    *fakes.IPTables
			timestamper *fakes.TimeStamper
			logger      *lagertest.TestLogger
			enforcer    *rules.Enforcer
		)

		BeforeEach(func() {
			fakeRule = &fakes.Rule{}
			timestamper = &fakes.TimeStamper{}
			logger = lagertest.NewTestLogger("test")
			iptables = &fakes.IPTables{}

			timestamper.CurrentTimeReturns(42)
			enforcer = rules.NewEnforcer(logger, timestamper, iptables)
		})

		It("enforces all the rules it receives on the correct chain", func() {
			err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRule.EnforceCallCount()).To(Equal(1))
		})

		It("creates a timestamped chain", func() {
			err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
			Expect(err).NotTo(HaveOccurred())

			Expect(iptables.NewChainCallCount()).To(Equal(1))
			tableName, chainName := iptables.NewChainArgsForCall(0)
			Expect(tableName).To(Equal("filter"))
			Expect(chainName).To(Equal("foo42"))
		})

		It("inserts the new chain into the FORWARD chain", func() {
			err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
			Expect(err).NotTo(HaveOccurred())

			Expect(iptables.InsertCallCount()).To(Equal(1))
			tableName, chainName, pos, ruleSpec := iptables.InsertArgsForCall(0)
			Expect(tableName).To(Equal("filter"))
			Expect(chainName).To(Equal("FORWARD"))
			Expect(pos).To(Equal(1))
			Expect(ruleSpec).To(Equal([]string{"-j", "foo42"}))
		})

		Context("when there is an older timestamped chain", func() {
			BeforeEach(func() {
				iptables.ListReturns([]string{
					"-A FORWARD -j foo0000000001",
					"-A FORWARD -j foo9999999999",
				}, nil)
			})
			It("gets deleted", func() {
				err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
				Expect(err).NotTo(HaveOccurred())

				Expect(iptables.DeleteCallCount()).To(Equal(1))
				table, chain, ruleSpec := iptables.DeleteArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("FORWARD"))
				Expect(ruleSpec).To(Equal([]string{"-j", "foo0000000001"}))
				Expect(iptables.ClearChainCallCount()).To(Equal(1))
				table, chain = iptables.ClearChainArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("foo0000000001"))
				Expect(iptables.DeleteChainCallCount()).To(Equal(1))
				table, chain = iptables.DeleteChainArgsForCall(0)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("foo0000000001"))
			})
		})

		Context("when there is an error enforcing a rule", func() {
			BeforeEach(func() {
				fakeRule.EnforceReturns(errors.New("banana"))
			})

			It("returns the error", func() {
				err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
				Expect(err).To(MatchError("banana"))
			})
		})

		Context("when inserting the new chain fails", func() {
			BeforeEach(func() {
				iptables.InsertReturns(errors.New("banana"))
			})

			It("it logs and returns a useful error", func() {
				err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
				Expect(err).To(MatchError("inserting chain: banana"))

				Expect(logger).To(gbytes.Say("insert-chain.*banana"))
			})
		})

		Context("when there are errors cleaning up old rules", func() {
			BeforeEach(func() {
				iptables.ListReturns(nil, errors.New("blueberry"))
			})

			It("it logs and returns a useful error", func() {
				err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
				Expect(err).To(MatchError("listing forward rules: blueberry"))

				Expect(logger).To(gbytes.Say("cleanup-rules.*blueberry"))
			})
		})

		Context("when there are errors cleaning up old chains", func() {
			BeforeEach(func() {
				iptables.DeleteReturns(errors.New("banana"))
				iptables.ListReturns([]string{"-A FORWARD -j foo0000000001"}, nil)
			})

			It("returns a useful error", func() {
				err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
				Expect(err).To(MatchError("cleanup old chain: banana"))
			})
		})

		Context("when creating the new chain fails", func() {
			BeforeEach(func() {
				iptables.NewChainReturns(errors.New("banana"))
			})

			It("it logs and returns a useful error", func() {
				err := enforcer.Enforce("foo", []rules.Rule{fakeRule})
				Expect(err).To(MatchError("creating chain: banana"))

				Expect(logger).To(gbytes.Say("create-chain.*banana"))
			})
		})
	})
})
