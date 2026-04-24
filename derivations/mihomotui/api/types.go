package api

type Proxy struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Now     string         `json:"now,omitempty"`
	All     []string       `json:"all,omitempty"`
	UDP     bool           `json:"udp,omitempty"`
	History []DelaySample  `json:"history,omitempty"`
	Extra   map[string]any `json:"extra,omitempty"`
}

type DelaySample struct {
	Time      string `json:"time"`
	Delay     int    `json:"delay"`
	MeanDelay int    `json:"meanDelay,omitempty"`
}

type ProxiesResp struct {
	Proxies map[string]Proxy `json:"proxies"`
}

type DelayResp struct {
	Delay   int    `json:"delay"`
	Message string `json:"message,omitempty"`
}

type GroupDelayResp map[string]int

type Connection struct {
	ID          string   `json:"id"`
	Metadata    ConnMeta `json:"metadata"`
	Upload      int64    `json:"upload"`
	Download    int64    `json:"download"`
	Start       string   `json:"start"`
	Chains      []string `json:"chains"`
	Rule        string   `json:"rule"`
	RulePayload string   `json:"rulePayload"`
}

type ConnMeta struct {
	Network         string `json:"network"`
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	DestinationIP   string `json:"destinationIP"`
	SourcePort      string `json:"sourcePort"`
	DestinationPort string `json:"destinationPort"`
	Host            string `json:"host"`
	ProcessPath     string `json:"processPath"`
}

type ConnectionsSnapshot struct {
	DownloadTotal int64        `json:"downloadTotal"`
	UploadTotal   int64        `json:"uploadTotal"`
	Connections   []Connection `json:"connections"`
}

type LogEntry struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Traffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

type Tun struct {
	Enable bool   `json:"enable"`
	Stack  string `json:"stack,omitempty"`
}

type Config struct {
	Port      int    `json:"port"`
	SocksPort int    `json:"socks-port"`
	MixedPort int    `json:"mixed-port"`
	Mode      string `json:"mode"`
	LogLevel  string `json:"log-level"`
	AllowLan  bool   `json:"allow-lan"`
	Tun       Tun    `json:"tun"`
	IPv6      bool   `json:"ipv6"`
}
