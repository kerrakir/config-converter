package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"converter/model"
)

func ParseHuawei(path string) (*model.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &model.Config{DeviceType: "huawei"}
	scanner := bufio.NewScanner(file)
	var currentInterface *model.Interface
	var currentVlan *model.Vlan
	var currentOSPF int
	var currentOSPFArea string
	var currentACLID int

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(line, "vlan ") && !strings.HasPrefix(line, "vlan batch"):
			if currentVlan != nil {
				cfg.Vlans = append(cfg.Vlans, *currentVlan)
			}
			var id int
			fmt.Sscanf(line, "vlan %d", &id)
			currentVlan = &model.Vlan{ID: id}

		case strings.HasPrefix(line, "description ") && currentVlan != nil:
			currentVlan.Name = strings.TrimPrefix(line, "description ")

		case line == "quit" && currentVlan != nil:
			cfg.Vlans = append(cfg.Vlans, *currentVlan)
			currentVlan = nil

		case strings.HasPrefix(line, "interface "):
			if currentInterface != nil {
				cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
			}
			name := strings.TrimPrefix(line, "interface ")
			nameParts := strings.Fields(name)
			if len(nameParts) == 2 && strings.EqualFold(nameParts[0], "vlanif") {
				name = "Vlan" + nameParts[1]
			}
			currentInterface = &model.Interface{Name: name}

		case strings.HasPrefix(line, "description ") && currentInterface != nil:
			currentInterface.Description = strings.TrimPrefix(line, "description ")

		case strings.HasPrefix(line, "port default vlan ") && currentInterface != nil:
			fmt.Sscanf(line, "port default vlan %d", &currentInterface.Vlan)

		case strings.HasPrefix(strings.ToLower(line), "vlan-type dot1q ") && currentInterface != nil:
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Huawei subinterface syntax can be "vlan-type dot1q <vid>" or "... vid <vid>".
				if strings.EqualFold(parts[2], "vid") && len(parts) >= 4 {
					fmt.Sscanf(parts[3], "%d", &currentInterface.Vlan)
				} else {
					fmt.Sscanf(parts[2], "%d", &currentInterface.Vlan)
				}
			}

		case strings.HasPrefix(line, "ip address ") && currentInterface != nil:
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentInterface.IP = parts[2] + " " + parts[3]
			}

		case line == "quit" && currentInterface != nil:
			cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
			currentInterface = nil

		case strings.HasPrefix(line, "port link-type trunk") && currentInterface != nil:

		case strings.HasPrefix(line, "port trunk allow-pass vlan ") && currentInterface != nil:
			currentInterface.TrunkVlans = strings.TrimPrefix(line, "port trunk allow-pass vlan ")

		case strings.HasPrefix(line, "ospf "):
			fmt.Sscanf(line, "ospf %d", &currentOSPF)

		case strings.HasPrefix(line, "router-id ") && currentOSPF != 0:
			cfg.OSPFRouterID = strings.TrimPrefix(line, "router-id ")

		case line == "silent-interface all" && currentOSPF != 0:
			cfg.OSPFPassiveDefault = true

		case strings.HasPrefix(line, "undo silent-interface ") && currentOSPF != 0:
			iface := strings.TrimPrefix(line, "undo silent-interface ")
			cfg.OSPFNoPassiveIfaces = append(cfg.OSPFNoPassiveIfaces, normalizeOspfIfaceFromHuawei(iface))

		case line == "quit" && currentOSPF != 0:
			currentOSPF = 0

		case strings.HasPrefix(line, "area "):
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentOSPFArea = parts[1]
			}

		case strings.HasPrefix(line, "network "):
			parts := strings.Fields(line)
			if len(parts) >= 3 && currentOSPF != 0 && currentOSPFArea != "" {
				cfg.OSPF = append(cfg.OSPF, model.OSPF{
					ProcessID: currentOSPF,
					Network:   parts[1],
					Wildcard:  parts[2],
					Area:      currentOSPFArea,
				})
			}

		case strings.HasPrefix(line, "nat address-group "):
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				cfg.NAT = append(cfg.NAT, model.NAT{
					Inside:  parts[len(parts)-2],
					Outside: parts[len(parts)-1],
				})
			}

		case strings.HasPrefix(line, "nat outbound ") && currentInterface != nil:
			var aclID int
			if _, err := fmt.Sscanf(line, "nat outbound %d", &aclID); err == nil {
				cfg.NATRule = append(cfg.NATRule, model.NATPolicy{
					ACLID:   aclID,
					Outside: currentInterface.Name,
				})
			}

		case line == "smtp server enable":
			cfg.Service.SMTP = true

		case line == "ftp server enable":
			cfg.Service.FTP = true

		case strings.HasPrefix(line, "stp mode "):
			cfg.STP.Mode = strings.TrimPrefix(line, "stp mode ")

		case strings.HasPrefix(line, "ip route-static "):
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				cfg.Routes = append(cfg.Routes, model.Route{
					Destination: parts[2],
					Mask:        parts[3],
					Gateway:     parts[4],
				})
			}

		case strings.HasPrefix(line, "acl number "):
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				var aclID int
				if _, err := fmt.Sscanf(parts[2], "%d", &aclID); err == nil {
					currentACLID = aclID
					getOrCreateACL(cfg, currentACLID, inferHuaweiACLType(aclID))
				}
			}

		case strings.HasPrefix(line, "rule ") && currentACLID != 0:
			if rule, ok := parseHuaweiACLRule(line); ok {
				acl := getOrCreateACL(cfg, currentACLID, inferHuaweiACLType(currentACLID))
				acl.Rules = append(acl.Rules, rule)
			}

		case line == "quit" && currentACLID != 0:
			currentACLID = 0
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if currentInterface != nil {
		cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
	}
	if currentVlan != nil {
		cfg.Vlans = append(cfg.Vlans, *currentVlan)
	}

	return cfg, nil
}

func inferHuaweiACLType(id int) string {
	if id >= 3000 && id <= 3999 {
		return "advanced"
	}
	return "basic"
}

func parseHuaweiACLRule(line string) (model.ACLRule, bool) {
	parts := strings.Fields(line)
	if len(parts) < 3 || strings.ToLower(parts[0]) != "rule" {
		return model.ACLRule{}, false
	}

	rule := model.ACLRule{}
	actionIdx := 2
	if _, err := fmt.Sscanf(parts[1], "%d", &rule.Sequence); err != nil {
		actionIdx = 1
	}
	if len(parts) <= actionIdx {
		return model.ACLRule{}, false
	}

	rule.Action = strings.ToLower(parts[actionIdx])
	idx := actionIdx + 1
	if idx < len(parts) {
		next := strings.ToLower(parts[idx])
		if next == "ip" || next == "tcp" || next == "udp" || next == "icmp" || next == "gre" {
			rule.Protocol = next
			idx++
		}
	}

	for idx < len(parts) {
		key := strings.ToLower(parts[idx])
		switch key {
		case "source":
			addr, wc, used := parseHuaweiAddressSpec(parts[idx+1:])
			if used == 0 {
				rule.Raw = strings.Join(parts[actionIdx:], " ")
				return rule, true
			}
			rule.Source = addr
			rule.Wildcard = wc
			idx += 1 + used
		case "destination":
			addr, wc, used := parseHuaweiAddressSpec(parts[idx+1:])
			if used == 0 {
				rule.Raw = strings.Join(parts[actionIdx:], " ")
				return rule, true
			}
			rule.Destination = addr
			rule.DstWildcard = wc
			idx += 1 + used
		case "source-port":
			if p, used := parsePortSpec(parts[idx+1:]); used > 0 {
				rule.SrcPort = p
				idx += 1 + used
			} else {
				rule.Raw = strings.Join(parts[actionIdx:], " ")
				return rule, true
			}
		case "destination-port":
			if p, used := parsePortSpec(parts[idx+1:]); used > 0 {
				rule.DstPort = p
				idx += 1 + used
			} else {
				rule.Raw = strings.Join(parts[actionIdx:], " ")
				return rule, true
			}
		default:
			rule.Raw = strings.Join(parts[actionIdx:], " ")
			return rule, true
		}
	}
	return rule, true
}

func parseHuaweiAddressSpec(tokens []string) (addr string, wildcard string, used int) {
	if len(tokens) == 0 {
		return "", "", 0
	}
	switch strings.ToLower(tokens[0]) {
	case "any":
		return "any", "", 1
	case "host":
		if len(tokens) >= 2 {
			return tokens[1], "0.0.0.0", 2
		}
		return "", "", 0
	default:
		if len(tokens) >= 2 {
			return tokens[0], tokens[1], 2
		}
		return "", "", 0
	}
}

func normalizeOspfIfaceFromHuawei(iface string) string {
	lower := strings.ToLower(strings.TrimSpace(iface))
	if strings.HasPrefix(lower, "vlanif") {
		id := strings.TrimSpace(iface[len("Vlanif"):])
		if id == "" {
			id = strings.TrimSpace(iface[len("vlanif"):])
		}
		if id != "" {
			return "Vlan" + id
		}
	}
	return iface
}
