package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"converter/model"
)

func ParseCisco(path string) (*model.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &model.Config{DeviceType: "cisco"}

	scanner := bufio.NewScanner(file)
	var currentInterface *model.Interface
	var currentVlan *model.Vlan
	var currentOSPF int
	var natInside []string
	var natOutside []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "!") {
			continue
		}

		switch {
		// VLAN
		case strings.HasPrefix(line, "vlan "):
			if currentVlan != nil {
				cfg.Vlans = append(cfg.Vlans, *currentVlan)
			}
			var id int
			fmt.Sscanf(line, "vlan %d", &id)
			currentVlan = &model.Vlan{ID: id}

		case strings.HasPrefix(line, "name ") && currentVlan != nil:
			currentVlan.Name = strings.TrimPrefix(line, "name ")

		case line == "exit" && currentVlan != nil:
			cfg.Vlans = append(cfg.Vlans, *currentVlan)
			currentVlan = nil

		// Интерфейсы
		case strings.HasPrefix(line, "interface "):
			if currentInterface != nil {
				cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
			}
			name := strings.TrimPrefix(line, "interface ")
			currentInterface = &model.Interface{Name: name}

		case strings.HasPrefix(line, "description ") && currentInterface != nil:
			currentInterface.Description = strings.TrimPrefix(line, "description ")

		case strings.HasPrefix(line, "switchport access vlan ") && currentInterface != nil:
			fmt.Sscanf(line, "switchport access vlan %d", &currentInterface.Vlan)

		case strings.HasPrefix(strings.ToLower(line), "encapsulation dot1q ") && currentInterface != nil:
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &currentInterface.Vlan)
			}

		case strings.HasPrefix(line, "ip address ") && currentInterface != nil:
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentInterface.IP = parts[2] + " " + parts[3]
			}

		case line == "ip nat inside" && currentInterface != nil:
			natInside = append(natInside, currentInterface.Name)

		case line == "ip nat outside" && currentInterface != nil:
			natOutside = append(natOutside, currentInterface.Name)

		case line == "exit" && currentInterface != nil:
			cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
			currentInterface = nil

		case strings.HasPrefix(line, "switchport trunk allowed vlan ") && currentInterface != nil:
			currentInterface.TrunkVlans = strings.TrimPrefix(line, "switchport trunk allowed vlan ")

		case strings.HasPrefix(line, "router ospf "):
			var id int
			fmt.Sscanf(line, "router ospf %d", &id)
			currentOSPF = id

		case strings.HasPrefix(line, "router-id ") && currentOSPF != 0:
			cfg.OSPFRouterID = strings.TrimPrefix(line, "router-id ")

		case line == "passive-interface default" && currentOSPF != 0:
			cfg.OSPFPassiveDefault = true

		case strings.HasPrefix(line, "no passive-interface ") && currentOSPF != 0:
			iface := strings.TrimPrefix(line, "no passive-interface ")
			cfg.OSPFNoPassiveIfaces = append(cfg.OSPFNoPassiveIfaces, iface)

		case line == "exit" && currentOSPF != 0:
			currentOSPF = 0

		case strings.HasPrefix(line, "network "):
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				cfg.OSPF = append(cfg.OSPF, model.OSPF{
					ProcessID: currentOSPF,
					Network:   parts[1],
					Wildcard:  parts[2],
					Area:      parts[4],
				})
			}

		case strings.HasPrefix(line, "spanning-tree mode "):
			cfg.STP.Mode = strings.TrimPrefix(line, "spanning-tree mode ")

		case line == "ip smtp server":
			cfg.Service.SMTP = true

		case line == "ip ftp server enable":
			cfg.Service.FTP = true

		// Маршруты
		case strings.HasPrefix(line, "ip route "):
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				cfg.Routes = append(cfg.Routes, model.Route{
					Destination: parts[2],
					Mask:        parts[3],
					Gateway:     parts[4],
				})
			}

		case strings.HasPrefix(line, "access-list "):
			if aclID, aclType, rule, ok := parseCiscoACLLine(line); ok {
				acl := getOrCreateACL(cfg, aclID, aclType)
				acl.Rules = append(acl.Rules, rule)
			}

		case strings.HasPrefix(line, "ip nat inside source list "):
			var aclID int
			var outside string
			if _, err := fmt.Sscanf(line, "ip nat inside source list %d interface %s overload", &aclID, &outside); err == nil {
				cfg.NATRule = append(cfg.NATRule, model.NATPolicy{
					ACLID:    aclID,
					Outside:  outside,
					Overload: true,
				})
			} else if _, err := fmt.Sscanf(line, "ip nat inside source list %d interface %s", &aclID, &outside); err == nil {
				cfg.NATRule = append(cfg.NATRule, model.NATPolicy{
					ACLID:   aclID,
					Outside: outside,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// финализируем незакрытые блоки
	if currentInterface != nil {
		cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
	}
	if currentVlan != nil {
		cfg.Vlans = append(cfg.Vlans, *currentVlan)
	}
	for _, inIf := range natInside {
		for _, outIf := range natOutside {
			cfg.NAT = append(cfg.NAT, model.NAT{
				Inside:  inIf,
				Outside: outIf,
			})
		}
	}

	return cfg, nil
}

func getOrCreateACL(cfg *model.Config, id int, aclType string) *model.ACL {
	for i := range cfg.ACLs {
		if cfg.ACLs[i].ID == id {
			if cfg.ACLs[i].Type == "" {
				cfg.ACLs[i].Type = aclType
			}
			return &cfg.ACLs[i]
		}
	}
	cfg.ACLs = append(cfg.ACLs, model.ACL{
		ID:   id,
		Type: aclType,
	})
	return &cfg.ACLs[len(cfg.ACLs)-1]
}

func parseCiscoACLLine(line string) (int, string, model.ACLRule, bool) {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return 0, "", model.ACLRule{}, false
	}

	var aclID int
	if _, err := fmt.Sscanf(parts[1], "%d", &aclID); err != nil {
		return 0, "", model.ACLRule{}, false
	}

	aclType := inferCiscoACLType(aclID, parts)
	if aclType == "extended" {
		rule, ok := parseCiscoExtendedACLRule(parts[2:])
		return aclID, aclType, rule, ok
	}
	rule, ok := parseCiscoStandardACLRule(parts[2:])
	return aclID, "standard", rule, ok
}

func inferCiscoACLType(id int, parts []string) string {
	if (id >= 100 && id <= 199) || (id >= 2000 && id <= 2699) {
		return "extended"
	}
	if len(parts) >= 5 {
		proto := strings.ToLower(parts[3])
		switch proto {
		case "ip", "tcp", "udp", "icmp", "gre", "esp", "ahp", "ospf":
			return "extended"
		}
	}
	return "standard"
}

func parseCiscoStandardACLRule(tokens []string) (model.ACLRule, bool) {
	if len(tokens) < 2 {
		return model.ACLRule{}, false
	}
	rule := model.ACLRule{
		Action: strings.ToLower(tokens[0]),
	}
	addr, wildcard, used := parseCiscoAddressSpec(tokens[1:])
	if used == 0 {
		rule.Raw = strings.Join(tokens, " ")
		return rule, true
	}
	rule.Source = addr
	rule.Wildcard = wildcard
	return rule, true
}

func parseCiscoExtendedACLRule(tokens []string) (model.ACLRule, bool) {
	// tokens format: <action> <proto> <src> [src-port] <dst> [dst-port]
	if len(tokens) < 4 {
		return model.ACLRule{}, false
	}
	rule := model.ACLRule{
		Action:   strings.ToLower(tokens[0]),
		Protocol: strings.ToLower(tokens[1]),
	}
	idx := 2

	src, srcWc, used := parseCiscoAddressSpec(tokens[idx:])
	if used == 0 {
		rule.Raw = strings.Join(tokens, " ")
		return rule, true
	}
	rule.Source = src
	rule.Wildcard = srcWc
	idx += used

	if rule.Protocol == "tcp" || rule.Protocol == "udp" {
		if p, used := parsePortSpec(tokens[idx:]); used > 0 {
			rule.SrcPort = p
			idx += used
		}
	}

	dst, dstWc, used := parseCiscoAddressSpec(tokens[idx:])
	if used == 0 {
		rule.Raw = strings.Join(tokens, " ")
		return rule, true
	}
	rule.Destination = dst
	rule.DstWildcard = dstWc
	idx += used

	if rule.Protocol == "tcp" || rule.Protocol == "udp" {
		if p, used := parsePortSpec(tokens[idx:]); used > 0 {
			rule.DstPort = p
			idx += used
		}
	}

	if idx < len(tokens) {
		// Keep extra qualifiers (log, established...) as raw tail for visibility.
		tail := strings.Join(tokens[idx:], " ")
		if tail != "" {
			if rule.Raw == "" {
				rule.Raw = tail
			} else {
				rule.Raw = rule.Raw + " " + tail
			}
		}
	}
	return rule, true
}

func parseCiscoAddressSpec(tokens []string) (addr string, wildcard string, used int) {
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

func parsePortSpec(tokens []string) (string, int) {
	if len(tokens) < 2 {
		return "", 0
	}
	op := strings.ToLower(tokens[0])
	switch op {
	case "eq", "neq", "gt", "lt":
		return op + " " + tokens[1], 2
	case "range":
		if len(tokens) >= 3 {
			return op + " " + tokens[1] + " " + tokens[2], 3
		}
	}
	return "", 0
}
