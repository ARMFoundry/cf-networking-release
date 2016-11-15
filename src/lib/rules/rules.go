package rules

import (
	"fmt"
	"strconv"
	"strings"
)

type IPTablesRule []string

func AppendComment(rule IPTablesRule, comment string) IPTablesRule {
	comment = strings.Replace(comment, " ", "_", -1)
	return IPTablesRule(
		append(rule, "-m", "comment", "--comment", comment),
	)
}

func NewMarkAllowRule(destinationIP, protocol string, port int, tag string, sourceAppGUID, destinationAppGUID string) IPTablesRule {
	return AppendComment(IPTablesRule{
		"-d", destinationIP,
		"-p", protocol,
		"--dport", strconv.Itoa(port),
		"-m", "mark", "--mark", fmt.Sprintf("0x%s", tag),
		"--jump", "ACCEPT",
	}, fmt.Sprintf("src:%s_dst:%s", sourceAppGUID, destinationAppGUID))
}

func NewMarkSetRule(sourceIP, tag, appGUID string) IPTablesRule {
	return AppendComment(IPTablesRule{
		"--source", sourceIP,
		"--jump", "MARK", "--set-xmark", fmt.Sprintf("0x%s", tag),
	}, fmt.Sprintf("src:%s", appGUID))
}

func NewDefaultEgressRule(localSubnet, overlayNetwork string) IPTablesRule {
	return IPTablesRule{
		"--source", localSubnet,
		"!", "-d", overlayNetwork,
		"--jump", "MASQUERADE",
	}
}

func NewLogRule(rule IPTablesRule, name string) IPTablesRule {
	return IPTablesRule(append(
		rule, "-m", "limit", "--limit", "2/min",
		"--jump", "LOG",
		"--log-prefix", name,
	))
}

func NewAcceptExistingLocalRule() IPTablesRule {
	return IPTablesRule{
		"-i", "cni-flannel0",
		"-m", "state", "--state", "ESTABLISHED,RELATED",
		"--jump", "ACCEPT",
	}
}

func NewDefaultDenyLocalRule(localSubnet string) IPTablesRule {
	return IPTablesRule{
		"-i", "cni-flannel0",
		"--source", localSubnet,
		"-d", localSubnet,
		"--jump", "REJECT",
	}
}

func NewAcceptExistingRemoteRule(vni int) IPTablesRule {
	return IPTablesRule{
		"-i", fmt.Sprintf("flannel.%d", vni),
		"-m", "state", "--state", "ESTABLISHED,RELATED",
		"--jump", "ACCEPT",
	}
}

func NewDefaultDenyRemoteRule(vni int) IPTablesRule {
	return IPTablesRule{
		"-i", fmt.Sprintf("flannel.%d", vni),
		"--jump", "REJECT",
	}
}

func NewNetOutRule(containerIP, startIP, endIP string) IPTablesRule {
	return IPTablesRule{
		"--source", containerIP,
		"-m", "iprange",
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"--jump", "RETURN",
	}
}

func NewNetOutWithPortsRule(containerIP, startIP, endIP string, startPort, endPort int, protocol string) IPTablesRule {
	return IPTablesRule{
		"--source", containerIP,
		"-m", "iprange",
		"-p", protocol,
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"-m", protocol,
		"--destination-port", fmt.Sprintf("%d:%d", startPort, endPort),
		"--jump", "RETURN",
	}
}

func NewNetOutLogRule(containerIP, startIP, endIP, chain string) IPTablesRule {
	return IPTablesRule{
		"--source", containerIP,
		"-m", "iprange",
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"-g", chain,
	}
}

func NewNetOutWithPortsLogRule(containerIP, startIP, endIP string, startPort, endPort int, protocol, chain string) IPTablesRule {
	return IPTablesRule{
		"--source", containerIP,
		"-m", "iprange",
		"-p", protocol,
		"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
		"-m", protocol,
		"--destination-port", fmt.Sprintf("%d:%d", startPort, endPort),
		"-g", chain,
	}
}

func NewNetOutDefaultLogRule(prefix string) IPTablesRule {
	return IPTablesRule{
		"-p", "tcp",
		"-m", "conntrack", "--ctstate", "INVALID,NEW,UNTRACKED",
		"-j", "LOG", "--log-prefix", prefix,
	}
}

func NewReturnRule() IPTablesRule {
	return IPTablesRule{
		"--jump", "RETURN",
	}
}

func NewNetOutRelatedEstablishedRule(subnet, overlayNetwork string) IPTablesRule {
	return IPTablesRule{
		"-s", subnet,
		"!", "-d", overlayNetwork,
		"-m", "state", "--state", "RELATED,ESTABLISHED",
		"--jump", "RETURN",
	}
}

func NewNetOutDefaultRejectRule(subnet, overlayNetwork string) IPTablesRule {
	return IPTablesRule{
		"-s", subnet,
		"!", "-d", overlayNetwork,
		"--jump", "REJECT",
		"--reject-with", "icmp-port-unreachable",
	}
}
