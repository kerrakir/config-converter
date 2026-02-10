package generator

import (
	"fmt"
	"strings"

	"converter/model"
)

func GenerateHuawei(cfg *model.Config) string {
	var sb strings.Builder

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
	for _, o := range cfg.OSPF {
		sb.WriteString(fmt.Sprintf("ospf %d\n", o.ProcessID))
		sb.WriteString(fmt.Sprintf(" area %s\n", o.Area))
		sb.WriteString(fmt.Sprintf("  network %s %s\n", o.Network, o.Wildcard))
	}

	// Интерфейсы
	for _, i := range cfg.Interfaces {
		// если это L3 интерфейс по VLAN → Vlanif
		if strings.HasPrefix(strings.ToLower(i.Name), "vlan") || strings.HasPrefix(strings.ToLower(i.Name), "vlanif") {
			id := strings.TrimLeftFunc(i.Name, func(r rune) bool {
				return r < '0' || r > '9'
			})
			sb.WriteString(fmt.Sprintf("interface Vlanif %s\n", id))
		} else {
			sb.WriteString(fmt.Sprintf("interface %s\n", i.Name))
		}

		if i.TrunkVlans != "" {
			sb.WriteString(" port link-type trunk\n")
			sb.WriteString(fmt.Sprintf(" port trunk allow-pass vlan %s\n", i.TrunkVlans))
		}

		// STP
		if cfg.STP.Mode != "" {
			sb.WriteString(fmt.Sprintf("stp mode %s\n", cfg.STP.Mode))
		}

		// Сервисы
		if cfg.Service.SMTP {
			sb.WriteString("smtp server enable\n")
		}
		if cfg.Service.FTP {
			sb.WriteString("ftp server enable\n")
		}

		if i.Description != "" {
			sb.WriteString(fmt.Sprintf(" description %s\n", i.Description))
		}
		if i.Vlan != 0 {
			sb.WriteString(" port link-type access\n")
			sb.WriteString(fmt.Sprintf(" port default vlan %d\n", i.Vlan))
		}
		if i.IP != "" {
			sb.WriteString(fmt.Sprintf(" ip address %s\n", i.IP))
		}
		sb.WriteString("quit\n\n")
	}

	// Статические маршруты
	for _, r := range cfg.Routes {
		sb.WriteString(fmt.Sprintf("ip route-static %s %s %s\n", r.Destination, r.Mask, r.Gateway))
	}

	return sb.String()
}
