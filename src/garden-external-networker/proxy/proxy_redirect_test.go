package proxy_test

import (
	"errors"
	"garden-external-networker/fakes"
	"garden-external-networker/proxy"
	lib_fakes "lib/fakes"
	"lib/rules"
	"strconv"

	"github.com/containernetworking/plugins/pkg/ns"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate counterfeiter -o ../fakes/netNS.go --fake-name NetNS . netNS
type netNS interface {
	ns.NetNS
}

var _ = Describe("Redirect", func() {
	var (
		proxyRedirect    *proxy.Redirect
		iptablesAdapter  *lib_fakes.IPTablesAdapter
		namespaceAdapter *fakes.NamespaceAdapter
		netNS            *fakes.NetNS

		containerNetNamespace string
		redirectCIDR          string
		proxyPort             int
		proxyUID              int
	)

	BeforeEach(func() {
		iptablesAdapter = &lib_fakes.IPTablesAdapter{}
		namespaceAdapter = &fakes.NamespaceAdapter{}
		netNS = &fakes.NetNS{}
		netNS.DoStub = func(toRun func(ns.NetNS) error) error {
			return toRun(netNS)
		}

		namespaceAdapter.GetNSReturns(netNS, nil)

		containerNetNamespace = "some-network-namespace"
		redirectCIDR = "10.255.0.0/24"
		proxyPort = 1111
		proxyUID = 1

		proxyRedirect = &proxy.Redirect{
			IPTables:         iptablesAdapter,
			NamespaceAdapter: namespaceAdapter,
			RedirectCIDR:     redirectCIDR,
			ProxyPort:        proxyPort,
			ProxyUID:         proxyUID,
		}
	})

	Describe("Apply", func() {
		It("apply iptables rules to redirect traffic to the proxy in the container net namespace", func() {
			err := proxyRedirect.Apply(containerNetNamespace)
			Expect(err).NotTo(HaveOccurred())

			Expect(namespaceAdapter.GetNSCallCount()).To(Equal(1))
			Expect(namespaceAdapter.GetNSArgsForCall(0)).To(Equal(containerNetNamespace))

			Expect(netNS.DoCallCount()).To(Equal(1))

			Expect(iptablesAdapter.BulkAppendCallCount()).To(Equal(1))
			table, name, iptablesRules := iptablesAdapter.BulkAppendArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(name).To(Equal("OUTPUT"))
			Expect(iptablesRules).To(Equal([]rules.IPTablesRule{
				{
					"-d", redirectCIDR,
					"-p", "tcp",
					"-m", "owner", "!", "--uid-owner", string(strconv.Itoa(proxyUID)),
					"-j", "REDIRECT", "--to-port", string(strconv.Itoa(proxyPort)),
				},
			}))
		})

		Context("when bulk appending to OUTPUT fails", func() {
			BeforeEach(func() {
				iptablesAdapter.BulkAppendReturns(errors.New("banana"))
			})

			It("returns an error", func() {
				err := proxyRedirect.Apply(containerNetNamespace)
				Expect(err).To(MatchError("do in container: banana"))
			})
		})

		Context("when the redirect cidr is empty", func() {
			BeforeEach(func() {
				proxyRedirect.RedirectCIDR = ""
			})

			It("no-ops", func() {
				Expect(proxyRedirect.Apply(containerNetNamespace)).To(Succeed())
				Expect(netNS.DoCallCount()).To(Equal(0))
			})
		})
	})
})
