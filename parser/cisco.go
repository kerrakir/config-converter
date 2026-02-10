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

		case strings.HasPrefix(line, "ip address ") && currentInterface != nil:
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentInterface.IP = parts[2] + " " + parts[3]
			}

		case line == "exit" && currentInterface != nil:
			cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
			currentInterface = nil

		case strings.HasPrefix(line, "switchport trunk allowed vlan ") && currentInterface != nil:
			currentInterface.TrunkVlans = strings.TrimPrefix(line, "switchport trunk allowed vlan ")

		case strings.HasPrefix(line, "router ospf "):
			var id int
			fmt.Sscanf(line, "router ospf %d", &id)
			currentOSPF = id

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
		}
	}

	// финализируем незакрытые блоки
	if currentInterface != nil {
		cfg.Interfaces = append(cfg.Interfaces, *currentInterface)
	}
	if currentVlan != nil {
		cfg.Vlans = append(cfg.Vlans, *currentVlan)
	}

	return cfg, nil
}
