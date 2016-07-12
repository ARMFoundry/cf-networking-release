package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	appDir     string
	config     helpers.Config
	testConfig struct {
		TestUser         string `json:"test_user"`
		TestUserPassword string `json:"test_user_password"`
	}
)

func Auth(username, password string) {
	By("authenticating as " + username)
	cmd := exec.Command("cf", "auth", username, password)
	sess, err := gexec.Start(cmd, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))
}

func AuthAsAdmin() {
	Auth(config.AdminUser, config.AdminPassword)
}

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		config = helpers.LoadConfig()

		configPath := helpers.ConfigPath()
		configBytes, err := ioutil.ReadFile(configPath)
		Expect(err).NotTo(HaveOccurred())

		err = json.Unmarshal(configBytes, &testConfig)
		Expect(err).NotTo(HaveOccurred())

		Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
		AuthAsAdmin()

		appDir = os.Getenv("APP_DIR")
		Expect(appDir).NotTo(BeEmpty())

		// create binary
		os.Setenv("GOOS", "linux")
		os.Setenv("GOARCH", "amd64")
		binaryPath, err := gexec.Build(appDir)
		Expect(err).NotTo(HaveOccurred())
		err = os.Rename(binaryPath, filepath.Join(appDir, "proxy"))
		Expect(err).NotTo(HaveOccurred())

		rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
	})

	AfterSuite(func() {
		// remove binary
		err := os.Remove(filepath.Join(appDir, "proxy"))
		Expect(err).NotTo(HaveOccurred())
	})

	RunSpecs(t, "Acceptance Suite")
}

func pushApp(appName string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir,
		"-f", filepath.Join(appDir, "manifest.yml"),
		"-c", "./proxy",
		"-b", "binary_buildpack",
	).Wait(Timeout_Push)).To(gexec.Exit(0))
}

func scaleApp(appName string, instances int) {
	Expect(cf.Cf(
		"scale", appName,
		"-i", fmt.Sprintf("%d", instances),
	).Wait(Timeout_Short)).To(gexec.Exit(0))

	// wait for ssh to become available on new instances
	time.Sleep(15 * time.Second)
}

const (
	ip4Regex         = `(?:[0-9]{1,3}\.){3}[0-9]{1,3}`
	ipAddrParseRegex = `inet (` + ip4Regex + `)/24 scope global eth0`
)

func getInstanceIP(appName string, instanceIndex int) string {
	sshSession := cf.Cf(
		"ssh", appName,
		"-i", fmt.Sprintf("%d", instanceIndex),
		"--skip-host-validation",
		"-c", "ip addr",
	)
	Expect(sshSession.Wait(2 * Timeout_Short)).To(gexec.Exit(0))

	addrOut := string(sshSession.Out.Contents())
	matches := regexp.MustCompile(ipAddrParseRegex).FindStringSubmatch(addrOut)
	return matches[1]
}

func curlFromApp(appName string, instanceIndex int, endpoint string, expectSuccess bool) string {
	sshSession := cf.Cf(
		"ssh", appName,
		"-i", fmt.Sprintf("%d", instanceIndex),
		"--skip-host-validation",
		"-c", fmt.Sprintf("curl --connect-timeout 5 %s", endpoint),
	)

	if expectSuccess {
		Expect(sshSession.Wait(Timeout_Short)).To(gexec.Exit(0))
	} else {
		Expect(sshSession.Wait(Timeout_Short)).To(gexec.Exit(28))
	}

	return string(sshSession.Out.Contents())
}

func getAppGuid(appName string) string {
	session := cf.Cf("app", appName, "--guid")
	Expect(session.Wait(Timeout_Short)).To(gexec.Exit(0))
	return strings.TrimSpace(string(session.Out.Contents()))
}
