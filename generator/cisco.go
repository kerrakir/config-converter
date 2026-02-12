package generator

import (
	"fmt"
	"strings"

	"converter/model"
)

func GenerateCisco(cfg *model.Config) string {
	var sb strings.Builder
	sb.WriteString("enable\n")
	sb.WriteString("configure terminal\n")

	for _, v := range cfg.Vlans {
		sb.WriteString(fmt.Sprintf("vlan %d\n", v.ID))
		if v.Name != "" {
			sb.WriteString(fmt.Sprintf(" name %s\n", v.Name))
		}
		sb.WriteString(" exit\n")
	}

	ospfByProcess := make(map[int][]model.OSPF)
	var processOrder []int
	for _, o := range cfg.OSPF {
		if _, ok := ospfByProcess[o.ProcessID]; !ok {
			processOrder = append(processOrder, o.ProcessID)
		}
		ospfByProcess[o.ProcessID] = append(ospfByProcess[o.ProcessID], o)
	}
	for _, pid := range processOrder {
		sb.WriteString(fmt.Sprintf("router ospf %d\n", pid))
		if cfg.OSPFRouterID != "" {
			sb.WriteString(fmt.Sprintf(" router-id %s\n", cfg.OSPFRouterID))
		}
		if cfg.OSPFPassiveDefault {
			sb.WriteString(" passive-interface default\n")
			for _, iface := range cfg.OSPFNoPassiveIfaces {
				sb.WriteString(fmt.Sprintf(" no passive-interface %s\n", iface))
			}
		}
		for _, o := range ospfByProcess[pid] {
			sb.WriteString(fmt.Sprintf(" network %s %s area %s\n", o.Network, o.Wildcard, o.Area))
		}
		sb.WriteString(" exit\n")
	}

	if cfg.STP.Mode != "" {
		sb.WriteString(fmt.Sprintf("spanning-tree mode %s\n", cfg.STP.Mode))
	}

	if cfg.Service.SMTP {
		sb.WriteString("ip smtp server\n")
	}
	if cfg.Service.FTP {
		sb.WriteString("ip ftp server enable\n")
	}

	for _, i := range cfg.Interfaces {
		sb.WriteString(fmt.Sprintf("interface %s\n", i.Name))
		if i.TrunkVlans != "" {
			sb.WriteString(" switchport mode trunk\n")
			sb.WriteString(fmt.Sprintf(" switchport trunk allowed vlan %s\n", i.TrunkVlans))
		}
		if i.Description != "" {
			sb.WriteString(fmt.Sprintf(" description %s\n", i.Description))
		}
		if i.Vlan != 0 && isCiscoSubinterface(i.Name) {
			sb.WriteString(fmt.Sprintf(" encapsulation dot1Q %d\n", i.Vlan))
		} else if i.Vlan != 0 {
			sb.WriteString(fmt.Sprintf(" switchport access vlan %d\n", i.Vlan))
		}
		if i.IP != "" {
			sb.WriteString(fmt.Sprintf(" ip address %s\n", i.IP))
		}
		sb.WriteString(" exit\n")
	}

	for _, r := range cfg.Routes {
		sb.WriteString(fmt.Sprintf("ip route %s %s %s\n", r.Destination, r.Mask, r.Gateway))
	}
	for _, acl := range cfg.ACLs {
		ciscoACLID := mapACLIDToCisco(acl.ID, acl.Type)
		for _, rule := range acl.Rules {
			if rule.Raw != "" {
				sb.WriteString(fmt.Sprintf("access-list %d %s\n", ciscoACLID, rule.Raw))
				continue
			}
			action := rule.Action
			if action == "" {
				action = "permit"
			}
			if isExtendedACLRule(rule, acl.Type) {
				proto := rule.Protocol
				if proto == "" {
					proto = "ip"
				}
				line := fmt.Sprintf("access-list %d %s %s %s", ciscoACLID, action, proto, formatCiscoAddress(rule.Source, rule.Wildcard))
				if rule.SrcPort != "" {
					line += " " + rule.SrcPort
				}
				line += " " + formatCiscoAddress(rule.Destination, rule.DstWildcard)
				if rule.DstPort != "" {
					line += " " + rule.DstPort
				}
				sb.WriteString(line + "\n")
			} else {
				sb.WriteString(fmt.Sprintf("access-list %d %s %s\n", ciscoACLID, action, formatCiscoAddress(rule.Source, rule.Wildcard)))
			}
		}
	}
	for _, r := range cfg.NATRule {
		ciscoACL := mapACLIDToCisco(r.ACLID, findACLType(cfg, r.ACLID))
		line := fmt.Sprintf("ip nat inside source list %d interface %s", ciscoACL, r.Outside)
		if r.Overload {
			line += " overload"
		}
		sb.WriteString(line + "\n")
	}
	for _, n := range cfg.NAT {
		sb.WriteString(fmt.Sprintf("interface %s\n", n.Inside))
		sb.WriteString(" ip nat inside\n")
		sb.WriteString(" exit\n")
		sb.WriteString(fmt.Sprintf("interface %s\n", n.Outside))
		sb.WriteString(" ip nat outside\n")
		sb.WriteString(" exit\n")
	}
	sb.WriteString("end\n")

	return sb.String()
}

func mapACLIDToCisco(id int, aclType string) int {
	if aclType == "advanced" && id >= 3000 && id <= 3999 {
		return id - 1000
	}
	if (aclType == "basic" || aclType == "standard" || aclType == "") && id >= 2000 && id <= 2999 {
		return id - 2000
	}
	return id
}

func isExtendedACLRule(rule model.ACLRule, aclType string) bool {
	if aclType == "extended" || aclType == "advanced" {
		return true
	}
	return rule.Protocol != "" || rule.Destination != "" || rule.DstPort != "" || rule.SrcPort != ""
}

func formatCiscoAddress(addr, wildcard string) string {
	if addr == "" || strings.EqualFold(addr, "any") {
		return "any"
	}
	if wildcard == "" || wildcard == "0.0.0.0" {
		return "host " + addr
	}
	return addr + " " + wildcard
}

func findACLType(cfg *model.Config, id int) string {
	for _, acl := range cfg.ACLs {
		if acl.ID == id {
			return acl.Type
		}
	}
	return ""
}

func isCiscoSubinterface(name string) bool {
	return strings.Contains(name, ".")
}
