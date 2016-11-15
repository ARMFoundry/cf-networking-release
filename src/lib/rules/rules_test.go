package rules_test

import (
	"errors"
	"lib/fakes"
	"lib/rules"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Rules", func() {
	Describe("Enforce", func() {
		var (
			logger   *lagertest.TestLogger
			iptables *fakes.IPTables
			rule     rules.GenericRule
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("test")
			iptables = &fakes.IPTables{}
			rule = rules.GenericRule{
				Properties: []string{"-j", "some-other-chain"},
			}
		})

		It("appends an iptables rule to the chain supplied", func() {
			err := rule.Enforce("some-table", "some-chain", iptables, logger)
			Expect(err).NotTo(HaveOccurred())

			Expect(iptables.AppendUniqueCallCount()).To(Equal(1))
			table, chain, ruleSpec := iptables.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("some-table"))
			Expect(chain).To(Equal("some-chain"))
			Expect(ruleSpec).To(Equal([]string{"-j", "some-other-chain"}))
			Expect(logger).To(gbytes.Say(`enforce-rule.*{"chain":"some-chain","properties":"\[-j some-other-chain\]","table":"some-table"}`))
		})

		Context("when theres an error appending the rule", func() {
			It("logs and returns a useful error", func() {
				iptables.AppendUniqueReturns(errors.New("raspberry"))

				err := rule.Enforce("some-table", "some-chain", iptables, logger)
				Expect(err).To(MatchError("appending rule: raspberry"))
				Expect(logger).To(gbytes.Say("append-rule.*raspberry"))
			})
		})
	})

	Describe("AppendComment", func() {
		var originalRule rules.GenericRule
		BeforeEach(func() {
			originalRule = rules.GenericRule{
				Properties: []string{"some", "rule"},
			}
		})
		It("appends the comment to the iptables rule, replacing spaces with underscores", func() {
			rule := rules.AppendComment(originalRule, `some:comment statement`)
			Expect(rule).To(Equal(rules.GenericRule{
				Properties: []string{
					"some", "rule", "-m", "comment", "--comment", `some:comment_statement`,
				},
			}))
		})
	})
})
