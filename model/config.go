package model

type Interface struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Vlan        int    `json:"vlan,omitempty"`
	IP          string `json:"ip,omitempty"`

	TrunkVlans string `json:"trunk_vlans,omitempty"`
}

type OSPF struct {
	ProcessID int    `json:"process_id"`
	Network   string `json:"network"`
	Wildcard  string `json:"wildcard"`
	Area      string `json:"area"`
}

type Vlan struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

type Route struct {
	Destination string `json:"destination"`
	Mask        string `json:"mask"`
	Gateway     string `json:"gateway"`
}

type NAT struct {
	Inside  string `json:"inside"`
	Outside string `json:"outside"`
}

type Config struct {
	DeviceType string      `json:"device_type"`
	Vlans      []Vlan      `json:"vlans,omitempty"`
	Interfaces []Interface `json:"interfaces,omitempty"`
	Routes     []Route     `json:"routes,omitempty"`

	OSPF    []OSPF  `json:"ospf,omitempty"`
	NAT     []NAT   `json:"nat,omitempty"`
	Service Service `json:"service,omitempty"`
	STP     STP     `json:"stp,omitempty"`
}

type Service struct {
	SMTP bool `json:"smtp"`
	FTP  bool `json:"ftp"`
}

type STP struct {
	Mode string `json:"mode"` // pvst, rstp, mstp
}
