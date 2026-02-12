package generator

import (
	"fmt"
	"strings"

	"converter/model"
)

func GenerateHuawei(cfg *model.Config) string {
	var sb strings.Builder
	sb.WriteString("system-view\n")

	// vlan batch
	if len(cfg.Vlans) > 0 {
		sb.WriteString("vlan batch")
		for _, v := range cfg.Vlans {
			sb.WriteString(fmt.Sprintf(" %d", v.ID))
		}
		sb.WriteString("\n\n")

		// описания VLAN
		for _, v := range cfg.Vlans {
			sb.WriteString(fmt.Sprintf("vlan %d\n", v.ID))
			if v.Name != "" {
				sb.WriteString(fmt.Sprintf(" description %s\n", v.Name))
			}
			sb.WriteString("quit\n\n")
		}
	}

	// OSPF
	ospfByProcessArea := make(map[int]map[string][]model.OSPF)
	var processOrder []int
	areaOrderByProcess := make(map[int][]string)
	for _, o := range cfg.OSPF {
		if _, ok := ospfByProcessArea[o.ProcessID]; !ok {
			ospfByProcessArea[o.ProcessID] = make(map[string][]model.OSPF)
			processOrder = append(processOrder, o.ProcessID)
		}
		if _, ok := ospfByProcessArea[o.ProcessID][o.Area]; !ok {
			areaOrderByProcess[o.ProcessID] = append(areaOrderByProcess[o.ProcessID], o.Area)
		}
		ospfByProcessArea[o.ProcessID][o.Area] = append(ospfByProcessArea[o.ProcessID][o.Area], o)
	}
	for _, pid := range processOrder {
		sb.WriteString(fmt.Sprintf("ospf %d\n", pid))
		if cfg.OSPFRouterID != "" {
			sb.WriteString(fmt.Sprintf(" router-id %s\n", cfg.OSPFRouterID))
		}
		if cfg.OSPFPassiveDefault {
			sb.WriteString(" silent-interface all\n")
			for _, iface := range cfg.OSPFNoPassiveIfaces {
				sb.WriteString(fmt.Sprintf(" undo silent-interface %s\n", toHuaweiOspfIface(iface)))
			}
		}
		for _, area := range areaOrderByProcess[pid] {
			sb.WriteString(fmt.Sprintf(" area %s\n", area))
			for _, o := range ospfByProcessArea[pid][area] {
				sb.WriteString(fmt.Sprintf("  network %s %s\n", o.Network, o.Wildcard))
			}
		}
		sb.WriteString("quit\n\n")
	}

	// Интерфейсы
	for _, i := range cfg.Interfaces {
		// L3 interface → Vlanif
		nameLower := strings.ToLower(i.Name)
		if strings.HasPrefix(nameLower, "vlan") || strings.HasPrefix(nameLower, "vlanif") {
			id := strings.TrimLeftFunc(i.Name, func(r rune) bool { return r < '0' || r > '9' })
			sb.WriteString(fmt.Sprintf("interface Vlanif %s\n", id))
		} else {
			sb.WriteString(fmt.Sprintf("interface %s\n", i.Name))
		}

		if i.Description != "" {
			sb.WriteString(fmt.Sprintf(" description %s\n", i.Description))
		}

		// Access
		if i.Vlan != 0 && isHuaweiSubinterface(i.Name) {
			sb.WriteString(fmt.Sprintf(" vlan-type dot1q %d\n", i.Vlan))
		} else if i.Vlan != 0 {
			sb.WriteString(" port link-type access\n")
			sb.WriteString(fmt.Sprintf(" port default vlan %d\n", i.Vlan))
		}

		// Trunk
		if i.TrunkVlans != "" {
			sb.WriteString(" port link-type trunk\n")
			sb.WriteString(fmt.Sprintf(" port trunk allow-pass vlan %s\n", i.TrunkVlans))
		}

		// IP
		if i.IP != "" {
			sb.WriteString(fmt.Sprintf(" ip address %s\n", i.IP))
		}

		sb.WriteString("quit\n\n")
	}
	// Статические маршруты
	for _, r := range cfg.Routes {
		sb.WriteString(fmt.Sprintf("ip route-static %s %s %s\n", r.Destination, r.Mask, r.Gateway))
	}
	for _, acl := range cfg.ACLs {
		huaweiACLID := mapACLIDToHuawei(acl.ID, acl.Type)
		sb.WriteString(fmt.Sprintf("acl number %d\n", huaweiACLID))
		seq := 5
		for _, rule := range acl.Rules {
			if rule.Raw != "" {
				sb.WriteString(fmt.Sprintf(" # unsupported ACL rule: %s\n", rule.Raw))
				continue
			}
			action := rule.Action
			if action == "" {
				action = "permit"
			}
			ruleSeq := rule.Sequence
			if ruleSeq == 0 {
				ruleSeq = seq
			}
			if isExtendedACLRule(rule, acl.Type) {
				proto := rule.Protocol
				if proto == "" {
					proto = "ip"
				}
				line := fmt.Sprintf(" rule %d %s %s source %s", ruleSeq, action, proto, formatHuaweiAddress(rule.Source, rule.Wildcard))
				if rule.SrcPort != "" {
					line += " source-port " + rule.SrcPort
				}
				line += " destination " + formatHuaweiAddress(rule.Destination, rule.DstWildcard)
				if rule.DstPort != "" {
					line += " destination-port " + rule.DstPort
				}
				sb.WriteString(line + "\n")
			} else {
				sb.WriteString(fmt.Sprintf(" rule %d %s source %s\n", ruleSeq, action, formatHuaweiAddress(rule.Source, rule.Wildcard)))
			}
			seq += 5
		}
		sb.WriteString("quit\n\n")
	}
	if len(cfg.NATRule) > 0 {
		for _, r := range cfg.NATRule {
			aclType := findACLTypeForHuawei(cfg, r.ACLID)
			hwACL := mapACLIDToHuawei(r.ACLID, aclType)
			sb.WriteString(fmt.Sprintf("interface %s\n", r.Outside))
			sb.WriteString(fmt.Sprintf(" nat outbound %d\n", hwACL))
			sb.WriteString("quit\n")
		}
	} else {
		for _, n := range cfg.NAT {
			sb.WriteString(fmt.Sprintf("nat address-group 1 %s %s\n", n.Inside, n.Outside))
		}
	}
	if cfg.STP.Mode != "" {
		sb.WriteString(fmt.Sprintf("stp mode %s\n", mapCiscoSTPToHuawei(cfg.STP.Mode)))
	}
	if cfg.Service.SMTP {
		sb.WriteString("smtp server enable\n")
	}
	if cfg.Service.FTP {
		sb.WriteString("ftp server enable\n")
	}
	sb.WriteString("return\n")

	return sb.String()
}

func mapACLIDToHuawei(id int, aclType string) int {
	if aclType == "extended" && ((id >= 100 && id <= 199) || (id >= 2000 && id <= 2699)) {
		if id >= 2000 {
			return id + 1000
		}
		return id + 2900
	}
	if (aclType == "standard" || aclType == "basic" || aclType == "") && id >= 1 && id <= 1999 {
		return 2000 + id
	}
	return id
}

func formatHuaweiAddress(addr, wildcard string) string {
	if addr == "" || strings.EqualFold(addr, "any") {
		return "any"
	}
	if wildcard == "" || wildcard == "0.0.0.0" {
		return "host " + addr
	}
	return addr + " " + wildcard
}

func findACLTypeForHuawei(cfg *model.Config, id int) string {
	for _, acl := range cfg.ACLs {
		if acl.ID == id {
			return acl.Type
		}
	}
	return ""
}

func mapCiscoSTPToHuawei(mode string) string {
	l := strings.ToLower(strings.TrimSpace(mode))
	switch l {
	case "rapid-pvst", "pvst":
		return "rstp"
	default:
		return l
	}
}

func toHuaweiOspfIface(iface string) string {
	i := strings.TrimSpace(iface)
	low := strings.ToLower(i)
	if strings.HasPrefix(low, "vlanif") {
		return i
	}
	if strings.HasPrefix(low, "vlan") {
		id := strings.TrimPrefix(strings.TrimPrefix(i, "Vlan"), "vlan")
		id = strings.TrimSpace(id)
		if id != "" {
			return "Vlanif" + id
		}
	}
	return i
}

func isHuaweiSubinterface(name string) bool {
	return strings.Contains(name, ".")
}
