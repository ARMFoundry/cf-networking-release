package smoke_test

import (
	"cf-pusher/cf_cli_adapter"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"code.cloudfoundry.org/go-db-helpers/testsupport"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const Timeout_Short = 10 * time.Second

var _ = Describe("connectivity between containers on the overlay network", func() {
	Describe("networking policy", func() {
		var (
			appProxy     string
			appRegistry  string
			appTick      string
			appInstances int
			prefix       string
			spaceName    string
			orgName      string
		)

		BeforeEach(func() {
			prefix = config.Prefix
			orgName = config.SmokeOrg

			Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
			spaceName = prefix + "inter-container-connectivity"
			Expect(cf.Cf("create-space", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
			Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))

			Expect(cf.Cf("set-space-role", config.SmokeUser, config.SmokeOrg, spaceName, "SpaceDeveloper").Wait(Timeout_Push)).To(gexec.Exit(0))

			appInstances = config.AppInstances

			appProxy = prefix + "proxy"
			appRegistry = prefix + "registry"
			appTick = prefix + "tick"

		})

		AfterEach(func() {
			appReport(appProxy, Timeout_Short)
			appReport(appRegistry, Timeout_Short)
			appReport(appTick, Timeout_Short)
			Expect(cf.Cf("delete-space", spaceName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
		})

		It("allows the user to configure policies", func(done Done) {
			pushApp(appRegistry, "registry")
			pushApp(appProxy, "proxy")
			pushApp(appTick, "tick", "--no-start")
			setEnv(appTick, "REGISTRY_BASE_URL", fmt.Sprintf("http://%s.%s", appRegistry, config.AppsDomain))
			start(appTick)

			scaleApp(appTick, appInstances)

			ports := []int{8080}
			appsTest := []string{appTick}

			By("checking that all test app instances have registered themselves")
			checkRegistry(appRegistry, 60*time.Second, 500*time.Millisecond, len(appsTest)*appInstances)

			appIPs := getAppIPs(appRegistry)

			By("checking that the connection fails")
			runWithTimeout("check connection failures", 5*time.Minute, func() {
				assertConnectionFails(appProxy, appIPs, ports, 1)
			})

			By("creating policies")
			createAllPolicies(appProxy, appsTest, ports)

			// we should wait for minimum (pollInterval * 2)
			By("waiting for policies to be created on cells")
			time.Sleep(10 * time.Second)

			By(fmt.Sprintf("checking that %s can reach %s", appProxy, appsTest))
			runWithTimeout("check connection success", 5*time.Minute, func() {
				assertConnectionSucceeds(appProxy, appIPs, ports, 1)
			})

			By("deleting policies")
			deleteAllPolicies(appProxy, appsTest, ports)

			By(fmt.Sprintf("checking that %s can NOT reach %s", appProxy, appsTest))
			runWithTimeout("check connection failures, again", 5*time.Minute, func() {
				assertConnectionFails(appProxy, appIPs, ports, 1)
			})

			By("checking that the registry updates when apps are scaled")
			scaleApp(appTick, 1 /* instances */)
			checkRegistry(appRegistry, 60*time.Second, 500*time.Millisecond, len(appsTest))

			scaleApp(appTick, appInstances /* instances */)
			checkRegistry(appRegistry, 60*time.Second, 500*time.Millisecond, len(appsTest)*appInstances)

			close(done)
		}, 30*60 /* <-- overall spec timeout in seconds */)
	})
})

func createAllPolicies(sourceApp string, dstList []string, dstPorts []int) {
	cfCli := &cf_cli_adapter.Adapter{
		CfCliPath: "cf",
	}
	for _, destApp := range dstList {
		for _, port := range dstPorts {
			err := cfCli.AllowAccess(sourceApp, destApp, port, "tcp")
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func deleteAllPolicies(sourceApp string, dstList []string, dstPorts []int) {
	cfCli := &cf_cli_adapter.Adapter{
		CfCliPath: "cf",
	}
	for _, destApp := range dstList {
		for _, port := range dstPorts {
			err := cfCli.RemoveAccess(sourceApp, destApp, port, "tcp")
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func runWithTimeout(operation string, timeout time.Duration, work func()) {
	done := make(chan bool)
	go func() {
		fmt.Printf("starting %s\n", operation)
		work()
		done <- true
	}()

	select {
	case <-done:
		fmt.Printf("completed %s\n", operation)
		return
	case <-time.After(timeout):
		Fail("timeout on " + operation)
	}
}

func getAppIPs(registry string) []string {
	instancesResponse, err := getInstancesFromA8(registry)
	Expect(err).NotTo(HaveOccurred())

	ips := []string{}
	for _, instance := range instancesResponse.Instances {
		ip, _, err := net.SplitHostPort(instance.Endpoint.Value)
		Expect(err).NotTo(HaveOccurred())
		ips = append(ips, ip)
	}
	return ips
}

type RegistryInstancesResponse struct {
	Instances []struct {
		ServiceName string `json:"service_name"`
		Endpoint    struct {
			Value string `json:"value"`
		} `json:"endpoint"`
	} `json:"instances"`
}

func getInstancesFromA8(registry string) (*RegistryInstancesResponse, error) {
	resp, err := httpGetBytes(fmt.Sprintf("http://%s.%s/api/v1/instances", registry, config.AppsDomain))
	if err != nil {
		return nil, err
	}

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var instancesResponse RegistryInstancesResponse
	err = json.Unmarshal(resp.Body, &instancesResponse)
	if err != nil {
		return nil, err
	}
	return &instancesResponse, nil
}

func checkRegistry(registry string, timeout, pollingInterval time.Duration, totalInstances int) {
	registeredApps := func() (int, error) {
		instancesResponse, err := getInstancesFromA8(registry)
		if err != nil {
			return 0, err
		}
		return len(instancesResponse.Instances), nil
	}

	Eventually(registeredApps, timeout, pollingInterval).Should(Equal(totalInstances))
}

func assertConnectionSucceeds(sourceApp string, destApps []string, ports []int, nProxies int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 50 * nProxies,
	}
	parallelRunner.RunOnSliceStrings(destApps, func(appIP string) {
		for _, port := range ports {
			assertSingleConnection(appIP, port, sourceApp, true)
		}
	})
}

func assertConnectionFails(sourceApp string, destApps []string, ports []int, nProxies int) {
	parallelRunner := &testsupport.ParallelRunner{
		NumWorkers: 50 * nProxies,
	}
	parallelRunner.RunOnSliceStrings(destApps, func(appIP string) {
		for _, port := range ports {
			assertSingleConnection(appIP, port, sourceApp, false)
		}
	})
}

func assertSingleConnection(destIP string, port int, sourceAppName string, shouldSucceed bool) {
	if shouldSucceed {
		By(fmt.Sprintf("eventually proxy should reach %s at port %d", destIP, port))
		assertResponseContains(destIP, port, sourceAppName, "application_name")
	} else {
		By(fmt.Sprintf("eventually proxy should NOT reach %s at port %d", destIP, port))
		assertResponseContains(destIP, port, sourceAppName, "request failed")
	}
}

func assertResponseContains(destIP string, port int, sourceAppName string, desiredResponse string) {
	proxyTest := func() (string, error) {
		resp, err := httpGetBytes(fmt.Sprintf("http://%s.%s/proxy/%s:%d", sourceAppName, config.AppsDomain, destIP, port))
		if err != nil {
			return "", err
		}
		return string(resp.Body), nil
	}
	Eventually(proxyTest, 10*time.Second, 500*time.Millisecond).Should(ContainSubstring(desiredResponse))
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout:   4 * time.Second,
			KeepAlive: 0,
		}).Dial,
	},
}

type httpResp struct {
	StatusCode int
	Body       []byte
}

func httpGetBytes(url string) (httpResp, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return httpResp{}, err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return httpResp{}, err
	}

	return httpResp{resp.StatusCode, respBytes}, nil
}
