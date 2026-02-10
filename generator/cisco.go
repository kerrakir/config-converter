package generator

import (
	"fmt"
	"strings"

	"converter/model"
)

func GenerateCisco(cfg *model.Config) string {
	var sb strings.Builder

	for _, v := range cfg.Vlans {
		sb.WriteString(fmt.Sprintf("vlan %d\n", v.ID))
		if v.Name != "" {
			sb.WriteString(fmt.Sprintf(" name %s\n", v.Name))
		}
		sb.WriteString(" exit\n")
	}

	for _, o := range cfg.OSPF {
		sb.WriteString(fmt.Sprintf("router ospf %d\n", o.ProcessID))
		sb.WriteString(fmt.Sprintf(" network %s %s area %s\n", o.Network, o.Wildcard, o.Area))
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
		if i.Vlan != 0 {
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

	return sb.String()
}
