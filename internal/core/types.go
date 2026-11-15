package core

import (
	"context"
	"time"
)

// Endpoint is the loopback REST endpoint of the running core.
type Endpoint struct {
	Addr   string
	Secret string
}

type Kind int

const (
	KindEmbedded Kind = iota
	KindPrivileged
)

// Host is one running core. Reload and SetMode never restart the process.
type Host interface {
	Start(ctx context.Context, cfg []byte) error
	Reload(ctx context.Context, cfg []byte) error
	SetMode(ctx context.Context, mode string) error
	Stop(ctx context.Context) error
	Endpoint() Endpoint
	Kind() Kind
}

type Version struct {
	Meta    bool   `json:"meta"`
	Version string `json:"version"`
}

type ConnectionMeta struct {
	Network         string `json:"network"`
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	DestinationIP   string `json:"destinationIP"`
	SourcePort      string `json:"sourcePort"`
	DestinationPort string `json:"destinationPort"`
	Host            string `json:"host"`
	ProcessPath     string `json:"processPath"`
	Process         string `json:"process"`
}

type Connection struct {
	ID       string         `json:"id"`
	Metadata ConnectionMeta `json:"metadata"`
	Upload   int64          `json:"upload"`
	Download int64          `json:"download"`
	Start    time.Time      `json:"start"`
	Chains   []string       `json:"chains"`
	Rule     string         `json:"rule"`
	Payload  string         `json:"rulePayload"`
}

type Connections struct {
	DownloadTotal int64        `json:"downloadTotal"`
	UploadTotal   int64        `json:"uploadTotal"`
	Connections   []Connection `json:"connections"`
}

type LogEntry struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Rule struct {
	Type    string     `json:"type"`
	Index   int        `json:"index"`
	Payload string     `json:"payload"`
	Proxy   string     `json:"proxy"`
	Size    int        `json:"size"`
	Extra   *RuleExtra `json:"extra,omitempty"`
}

type RuleExtra struct {
	Disabled bool      `json:"disabled"`
	HitCount uint64    `json:"hitCount"`
	HitAt    time.Time `json:"hitAt"`
}

type Proxy struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	Now   string   `json:"now"`
	All   []string `json:"all"`
	UDP   bool     `json:"udp"`
	Alive bool     `json:"alive"`
}

type DNSAnswer struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}

type DNSResponse struct {
	Status int         `json:"Status"`
	Answer []DNSAnswer `json:"Answer"`
}

type Provider struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	VehicleType string    `json:"vehicleType"`
	RuleCount   int       `json:"ruleCount"`
	ProxyCount  int       `json:"proxyCount"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
