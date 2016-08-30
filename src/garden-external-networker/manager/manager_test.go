package manager_test

import (
	"errors"
	"net"

	"code.cloudfoundry.org/lager/lagertest"

	"garden-external-networker/fakes"
	"garden-external-networker/manager"

	lib_fakes "lib/fakes"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		mgr                     *manager.Manager
		cniController           *fakes.CNIController
		mounter                 *fakes.Mounter
		encodedGardenProperties string
		expectedExtraProperties map[string]string
		portAllocator           *fakes.PortAllocator
		ipTables                *lib_fakes.IPTables
		logger                  *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		mounter = &fakes.Mounter{}
		cniController = &fakes.CNIController{}
		ipTables = &lib_fakes.IPTables{}
		portAllocator = &fakes.PortAllocator{}

		cniController.UpReturns(&types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("169.254.1.2"),
					Mask: net.IPv4Mask(255, 255, 255, 0),
				},
			},
		}, nil)
		mgr = &manager.Manager{
			Logger:         logger,
			CNIController:  cniController,
			Mounter:        mounter,
			BindMountRoot:  "/some/fake/path",
			IPTables:       ipTables,
			OverlayNetwork: "10.255.0.0/16",
			PortAllocator:  portAllocator,
		}
		encodedGardenProperties = `{ "app_id": "some-group-id" }`
		expectedExtraProperties = map[string]string{"app_id": "some-group-id"}
	})

	Describe("Up", func() {
		It("should ensure that the netNS is mounted to the provided path", func() {
			_, err := mgr.Up("some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			source, target := mounter.IdempotentlyMountArgsForCall(0)
			Expect(source).To(Equal("/proc/42/ns/net"))
			Expect(target).To(Equal("/some/fake/path/some-container-handle"))
		})

		It("should return the IP address in the CNI result as a property", func() {
			properties, err := mgr.Up(42, "some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(properties.ContainerIP).To(Equal(net.ParseIP("169.254.1.2")))
			Expect(properties.DeprecatedHostIP).To(Equal(net.ParseIP("255.255.255.255")))
		})

		It("should call CNI Up, passing in the bind-mounted path to the net ns", func() {
			_, err := mgr.Up(42, "some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(cniController.UpCallCount()).To(Equal(1))
			namespacePath, handle, properties := cniController.UpArgsForCall(0)
			Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
			Expect(handle).To(Equal("some-container-handle"))
			Expect(properties).To(Equal(expectedExtraProperties))
		})

		XContext("when the chain name is longer than 28 characters", func() {
			It("truncates the name", func() {
				_, err := mgr.Up(42, "some-very-long-container-handle", encodedGardenProperties)
				Expect(err).NotTo(HaveOccurred())

				Expect(ipTables.NewChainCallCount()).To(Equal(1))
				_, chain := ipTables.NewChainArgsForCall(0)
				Expect(chain).To(Equal("netout--some-very-long-conta"))
			})
		})

		XIt("should create the container's chain by prepending netout to the handle", func() {
			_, err := mgr.Up(42, "container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.NewChainCallCount()).To(Equal(1))
			table, chain := ipTables.NewChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout--container-handle"))

			Expect(ipTables.InsertCallCount()).To(Equal(1))
			table, chain, pos, rulespec := ipTables.InsertArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(pos).To(Equal(1))
			Expect(rulespec).To(Equal([]string{"--jump", "netout--container-handle"}))
		})

		XIt("should write the default NetOut rules", func() {
			_, err := mgr.Up(42, "container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.AppendUniqueCallCount()).To(Equal(2))
			table, chain, rulespec := ipTables.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout--container-handle"))
			Expect(rulespec).To(Equal([]string{"-s", "169.254.1.2",
				"!", "-d", "10.255.0.0/16",
				"-m", "state", "--state", "RELATED,ESTABLISHED",
				"--jump", "RETURN"}))

			table, chain, rulespec = ipTables.AppendUniqueArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout--container-handle"))
			Expect(rulespec).To(Equal([]string{"-s", "169.254.1.2",
				"!", "-d", "10.255.0.0/16",
				"--jump", "REJECT",
				"--reject-with", "icmp-port-unreachable"}))
		})

		Context("when inserting fails", func() {
			BeforeEach(func() {
				ipTables.InsertReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := mgr.Up(42, "container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("initialize net out: inserting rule: banana"))
			})
		})

		Context("when creating the chain fails", func() {
			BeforeEach(func() {
				ipTables.NewChainReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := mgr.Up(42, "container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("initialize net out: creating chain: banana"))
			})
		})

		Context("when appending a rule fails", func() {
			BeforeEach(func() {
				ipTables.AppendUniqueReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := mgr.Up(42, "container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("initialize net out: appending rule: banana"))
			})
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				_, err := mgr.Up(0, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("up missing pid"))

				_, err = mgr.Up(42, "", encodedGardenProperties)
				Expect(err).To(MatchError("up missing container handle"))
			})
		})

		Context("when missing the encoded garden properties", func() {
			It("should not complain", func() {
				_, err := mgr.Up(42, "some-container-handle", "")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the encoded garden properties is an empty hash", func() {
			It("should still call CNI and the netman agent", func() {
				_, err := mgr.Up(42, "some-container-handle", "{}")
				Expect(err).NotTo(HaveOccurred())

				Expect(cniController.UpCallCount()).To(Equal(1))
				Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			})
		})

		Context("when unmarshaling the encoded garden properties fails", func() {
			It("returns the error", func() {
				_, err := mgr.Up(42, "some-container-handle", "%%%%")
				Expect(err).To(MatchError(ContainSubstring("unmarshal garden properties: invalid character")))
			})
		})

		Context("when the mounter fails", func() {
			It("should return the error", func() {
				mounter.IdempotentlyMountReturns(errors.New("boom"))
				_, err := mgr.Up(42, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("failed mounting /proc/42/ns/net to /some/fake/path/some-container-handle: boom"))
			})
		})

		Context("when the cni Up fails", func() {
			It("should return the error", func() {
				cniController.UpReturns(nil, errors.New("bang"))
				_, err := mgr.Up(42, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("cni up failed: bang"))
			})
		})
	})

	Describe("Down", func() {
		It("should ensure that the netNS is unmounted", func() {
			Expect(mgr.Down("some-container-handle", encodedGardenProperties)).To(Succeed())
			Expect(mounter.RemoveMountCallCount()).To(Equal(1))

			Expect(mounter.RemoveMountArgsForCall(0)).To(Equal("/some/fake/path/some-container-handle"))
		})

		It("should call CNI Down, passing in the bind-mounted path to the net ns", func() {
			Expect(mgr.Down("some-container-handle", encodedGardenProperties)).To(Succeed())
			Expect(cniController.DownCallCount()).To(Equal(1))
			namespacePath, handle, spec := cniController.DownArgsForCall(0)
			Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
			Expect(handle).To(Equal("some-container-handle"))
			Expect(spec).To(Equal(expectedExtraProperties))
		})

		Context("when encodedGardenProperties is empty", func() {
			It("should call CNI", func() {
				err := mgr.Down("some-container-handle", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(cniController.DownCallCount()).To(Equal(1))
				Expect(mounter.RemoveMountCallCount()).To(Equal(1))
			})
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				err := mgr.Down("", "")
				Expect(err).To(MatchError("down missing container handle"))
			})
		})

		Context("when the mounter fails", func() {
			It("should return the error", func() {
				mounter.RemoveMountReturns(errors.New("boom"))
				err := mgr.Down("some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("failed removing mount /some/fake/path/some-container-handle: boom"))
			})
		})

		Context("when the cni Down fails", func() {
			It("should return the error", func() {
				cniController.DownReturns(errors.New("bang"))
				err := mgr.Down("some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("cni down failed: bang"))
			})
		})
	})

	Describe("NetOut", func() {
		var netOutProperties string
		BeforeEach(func() {
			netOutProperties = `{
				"container_ip":"1.2.3.4",
				"netout_rule":{
					"protocol":1,
					"networks":[{"start":"1.1.1.1","end":"2.2.2.2"},{"start":"3.3.3.3","end":"4.4.4.4"}],
					"ports":[{"start":9000,"end":9999},{"start":1111,"end":2222}]
				}
			}`
		})
		It("prepends allow rules to the container's netout chain", func() {
			err := mgr.NetOut("some-handle", netOutProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.InsertCallCount()).To(Equal(4))
			writtenRules := [][]string{}
			for i := 0; i < 4; i++ {
				table, chain, pos, rulespec := ipTables.InsertArgsForCall(i)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("netout--some-handle"))
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
		Context("when the handle is over 28 characters", func() {
			It("truncates the handle", func() {
				err := mgr.NetOut("a-very-long-container-handle", netOutProperties)
				Expect(err).NotTo(HaveOccurred())
				Expect(ipTables.InsertCallCount()).To(Equal(4))
				for i := 0; i < 4; i++ {
					_, chain, _, _ := ipTables.InsertArgsForCall(i)
					Expect(chain).To(Equal("netout--a-very-long-containe"))
				}
			})
		})
		Context("when inserting the rule fails", func() {
			BeforeEach(func() {
				ipTables.InsertReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				err := mgr.NetOut("some-handle", netOutProperties)
				Expect(err).To(MatchError("inserting net-out rule: banana"))
			})
		})
		Context("when unmarshaling json fails", func() {
			BeforeEach(func() {
				netOutProperties = `%%%%%%%`
			})
			It("returns the error", func() {
				err := mgr.NetOut("some-handle", netOutProperties)
				Expect(err).To(MatchError(ContainSubstring("unmarshaling net-out properties: invalid character")))
			})

		})
	})

	Describe("NetIn", func() {
		BeforeEach(func() {
			encodedGardenProperties = `{ "host-ip": "1.2.3.4", "host-port": "0", "container-ip": "10.0.0.2", "container-port": "8888", "app_id": "some-group-id" }`
			portAllocator.AllocatePortReturns(1234, nil)
		})

		It("writes a netin iptables rule", func() {
			_, err := mgr.NetIn("some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.AppendUniqueCallCount()).To(Equal(1))
			table, chain, extraArgs := ipTables.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("nat"))
			Expect(chain).To(Equal("netin--some-container-handle"))
			Expect(extraArgs).To(Equal([]string{
				"-d", "1.2.3.4",
				"-p", "tcp",
				"-m", "tcp", "--dport", "1234",
				"--jump", "DNAT",
				"--to-destination", "10.0.0.2:8888",
				"-m", "comment", "--comment", "dst:some-group-id"}))
		})

		Context("when the container handle is longer than 29 characters", func() {
			It("truncates the chain name to no more than 29 characters", func() {
				_, err := mgr.NetIn("some-container-handle-that-is-longer-than-29-characters", encodedGardenProperties)
				Expect(err).NotTo(HaveOccurred())

				Expect(ipTables.AppendUniqueCallCount()).To(Equal(1))
				_, chain, _ := ipTables.AppendUniqueArgsForCall(0)
				Expect(chain).To(Equal("netin--some-container-handle-"))
			})
		})

		BeforeEach(func() {
			encodedGardenProperties = `{ "host-ip": "1.2.3.4", "host-port": "11111", "container-ip": "10.0.0.2", "container-port": "8888", "app_id": "some-group-id" }`
		})

		It("uses the specified port", func() {
			netInProperties, err := mgr.NetIn("some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(netInProperties).To(Equal(&manager.NetInProperties{
				HostIP:        "1.2.3.4",
				HostPort:      1234,
				ContainerIP:   "10.0.0.2",
				ContainerPort: 8888,
				GroupID:       "some-group-id",
			}))
		})

		Context("when no container port is specified", func() {
			BeforeEach(func() {
				encodedGardenProperties = `{ "host-ip": "1.2.3.4", "host-port": "1234", "container-ip": "10.0.0.2", "container-port": "0", "app_id": "some-group-id" }`
			})

			It("uses the specified external port", func() {
				netInProperties, err := mgr.NetIn("some-container-handle", encodedGardenProperties)
				Expect(err).NotTo(HaveOccurred())

				Expect(netInProperties).To(Equal(&manager.NetInProperties{
					HostIP:        "1.2.3.4",
					HostPort:      1234,
					ContainerIP:   "10.0.0.2",
					ContainerPort: 1234,
					GroupID:       "some-group-id",
				}))
			})
		})
	})
})