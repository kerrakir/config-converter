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

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(line, "vlan "):
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
			currentInterface = &model.Interface{Name: name}

		case strings.HasPrefix(line, "description ") && currentInterface != nil:
			currentInterface.Description = strings.TrimPrefix(line, "description ")

		case strings.HasPrefix(line, "port default vlan ") && currentInterface != nil:
			fmt.Sscanf(line, "port default vlan %d", &currentInterface.Vlan)

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

		case strings.HasPrefix(line, "area "):
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				cfg.OSPF = append(cfg.OSPF, model.OSPF{
					ProcessID: currentOSPF,
					Network:   parts[2],
					Wildcard:  parts[3],
					Area:      parts[1],
				})
			}

		case strings.HasPrefix(line, "nat address-group "):
			parts := strings.Fields(line)
			cfg.NAT = append(cfg.NAT, model.NAT{
				Inside:  parts[2],
				Outside: parts[3],
			})

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
		}
	}

	if currentInterface != nil {
		cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
	}
	if currentVlan != nil {
		cfg.Vlans = append(cfg.Vlans, *currentVlan)
	}

	return cfg, nil
}
