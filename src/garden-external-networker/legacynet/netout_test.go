package legacynet_test

import (
	"errors"
	"garden-external-networker/fakes"
	"garden-external-networker/legacynet"
	"net"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager/lagertest"

	lib_fakes "lib/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Netout", func() {
	var (
		netOut     *legacynet.NetOut
		chainNamer *fakes.ChainNamer
		ipTables   *lib_fakes.IPTables
		logger     *lagertest.TestLogger
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		chainNamer = &fakes.ChainNamer{}
		ipTables = &lib_fakes.IPTables{}
		netOut = &legacynet.NetOut{
			ChainNamer: chainNamer,
			IPTables:   ipTables,
		}
		chainNamer.NameReturns("some-chain-name")
	})

	Describe("Initialize", func() {
		It("creates the chain with the name from the chain namer", func() {
			err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.NameCallCount()).To(Equal(1))
			prefix, handle := chainNamer.NameArgsForCall(0)
			Expect(prefix).To(Equal("netout"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(ipTables.NewChainCallCount()).To(Equal(1))
			_, chain := ipTables.NewChainArgsForCall(0)
			Expect(chain).To(Equal("some-chain-name"))
		})

		It("inserts a jump rule for the new chain", func() {
			err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.InsertCallCount()).To(Equal(1))
			table, chain, position, extraArgs := ipTables.InsertArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(position).To(Equal(1))
			Expect(extraArgs).To(Equal([]string{"--jump", "some-chain-name"}))
		})

		It("writes the default netout rules", func() {
			err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.AppendUniqueCallCount()).To(Equal(2))
			table, chain, rulespec := ipTables.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))
			Expect(rulespec).To(Equal([]string{"-s", "5.6.7.8",
				"!", "-d", "9.9.0.0/16",
				"-m", "state", "--state", "RELATED,ESTABLISHED",
				"--jump", "RETURN"}))

			table, chain, rulespec = ipTables.AppendUniqueArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))
			Expect(rulespec).To(Equal([]string{"-s", "5.6.7.8",
				"!", "-d", "9.9.0.0/16",
				"--jump", "REJECT",
				"--reject-with", "icmp-port-unreachable"}))
		})

		Context("when creating a new chain fails", func() {
			BeforeEach(func() {
				ipTables.NewChainReturns(errors.New("potata"))
			})
			It("returns the error", func() {
				err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
				Expect(err).To(MatchError("creating chain: potata"))
			})
		})

		Context("when inserting a new rule fails", func() {
			BeforeEach(func() {
				ipTables.InsertReturns(errors.New("potato"))
			})
			It("returns the error", func() {
				err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
				Expect(err).To(MatchError("inserting rule: potato"))
			})
		})

		Context("when writing the netout rule fails", func() {
			BeforeEach(func() {
				ipTables.AppendUniqueReturns(errors.New("potato"))
			})
			It("returns the error", func() {
				err := netOut.Initialize(logger, "some-container-handle", net.ParseIP("5.6.7.8"), "9.9.0.0/16")
				Expect(err).To(MatchError("appending rule: potato"))
			})
		})
	})

	Describe("Cleanup", func() {
		It("deletes the correct jump rule from the forward chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(chainNamer.NameCallCount()).To(Equal(1))
			prefix, handle := chainNamer.NameArgsForCall(0)
			Expect(prefix).To(Equal("netout"))
			Expect(handle).To(Equal("some-container-handle"))

			Expect(ipTables.DeleteCallCount()).To(Equal(1))
			table, chain, extraArgs := ipTables.DeleteArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(extraArgs).To(Equal([]string{"--jump", "some-chain-name"}))
		})

		It("clears the container chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.ClearChainCallCount()).To(Equal(1))
			table, chain := ipTables.ClearChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))
		})

		It("deletes the container chain", func() {
			err := netOut.Cleanup("some-container-handle")
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.DeleteChainCallCount()).To(Equal(1))
			table, chain := ipTables.DeleteChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("some-chain-name"))
		})

		Context("when deleting the jump rule fails", func() {
			BeforeEach(func() {
				ipTables.DeleteReturns(errors.New("yukon potato"))
			})
			It("returns an error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError("delete rule: yukon potato"))
			})
		})

		Context("when clearing the container chain fails", func() {
			BeforeEach(func() {
				ipTables.ClearChainReturns(errors.New("idaho potato"))
			})
			It("returns an error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError("clear chain: idaho potato"))
			})
		})

		Context("when deleting the container chain fails", func() {
			BeforeEach(func() {
				ipTables.DeleteChainReturns(errors.New("purple potato"))
			})
			It("returns an error", func() {
				err := netOut.Cleanup("some-container-handle")
				Expect(err).To(MatchError("delete chain: purple potato"))
			})
		})
	})

	Describe("InsertRule", func() {
		var netOutRule garden.NetOutRule

		Context("when ports and protocol are specified", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
					Ports: []garden.PortRange{
						{Start: 9000, End: 9999},
						{Start: 1111, End: 2222},
					},
				}
			})

			It("prepends allow rules to the container's netout chain", func() {
				err := netOut.InsertRule("some-handle", netOutRule, "1.2.3.4")
				Expect(err).NotTo(HaveOccurred())

				Expect(ipTables.InsertCallCount()).To(Equal(4))
				writtenRules := [][]string{}
				for i := 0; i < 4; i++ {
					table, chain, pos, rulespec := ipTables.InsertArgsForCall(i)
					Expect(table).To(Equal("filter"))
					Expect(chain).To(Equal("some-chain-name"))
					Expect(pos).To(Equal(1))
					writtenRules = append(writtenRules, rulespec)
				}
				Expect(writtenRules).To(ConsistOf(
					[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"},
					[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"},
					[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "9000:9999",
						"--jump", "RETURN"},
					[]string{"--source", "1.2.3.4",
						"-m", "iprange", "-p", "tcp",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"-m", "tcp", "--destination-port", "1111:2222",
						"--jump", "RETURN"},
				))
			})

			Context("when insert rule fails", func() {
				BeforeEach(func() {
					ipTables.InsertReturns(errors.New("potato"))
				})
				It("returns an error", func() {
					err := netOut.InsertRule("some-container-handle", netOutRule, "1.2.3.4")
					Expect(err).To(MatchError("inserting net-out rule: potato"))
				})
			})
		})

		Context("when ports or protocol are not specified", func() {
			BeforeEach(func() {
				netOutRule = garden.NetOutRule{
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
						{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
					},
				}
			})

			It("prepends allow rules to the container's netout chain", func() {
				err := netOut.InsertRule("some-handle", netOutRule, "1.2.3.4")
				Expect(err).NotTo(HaveOccurred())

				Expect(ipTables.InsertCallCount()).To(Equal(2))
				writtenRules := [][]string{}
				for i := 0; i < 2; i++ {
					table, chain, pos, rulespec := ipTables.InsertArgsForCall(i)
					Expect(table).To(Equal("filter"))
					Expect(chain).To(Equal("some-chain-name"))
					Expect(pos).To(Equal(1))
					writtenRules = append(writtenRules, rulespec)
				}
				Expect(writtenRules).To(ConsistOf(
					[]string{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "1.1.1.1-2.2.2.2",
						"--jump", "RETURN"},
					[]string{"--source", "1.2.3.4", "-m", "iprange",
						"--dst-range", "3.3.3.3-4.4.4.4",
						"--jump", "RETURN"},
				))
			})

			Context("when insert rule fails", func() {
				BeforeEach(func() {
					ipTables.InsertReturns(errors.New("potato"))
				})
				It("returns an error", func() {
					err := netOut.InsertRule("some-container-handle", netOutRule, "1.2.3.4")
					Expect(err).To(MatchError("inserting net-out rule: potato"))
				})
			})
		})
	})
})
