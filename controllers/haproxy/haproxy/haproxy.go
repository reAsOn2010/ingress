package haproxy

import (
	"github.com/reAsOn2010/ingress/controllers/haproxy/haproxy/rewrite"
)

type IngressConfig struct {
	Backends []*Backend
	Hosts    []*Host
}

type Host struct {
	Name              string
	Locations         []*Location
	SSL               bool
	SSLCertificate    string
	SSLCertificateKey string
	SSLPemChecksum    string
}

type HostByName []*Host

func (c HostByName) Len() int      { return len(c) }
func (c HostByName) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c HostByName) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

type Location struct {
	Path         string
	IsDefBackend bool
	Backend      *Backend
}

type LocationByPath []*Location

func (c LocationByPath) Len() int      { return len(c) }
func (c LocationByPath) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c LocationByPath) Less(i, j int) bool {
	return c[i].Path > c[j].Path
}

type Backend struct {
	Name                string
	Algorithm           string
	SessionAffinity     bool
	CookieStickySession bool
	Endpoints           []Endpoint
	RewriteRules        []*rewrite.Rewrite
}

type BackendByName []*Backend

func (c BackendByName) Len() int      { return len(c) }
func (c BackendByName) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c BackendByName) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

type Endpoint struct {
	Addr string
	Port string
}

type EndpointByAddrPort []Endpoint

func (c EndpointByAddrPort) Len() int      { return len(c) }
func (c EndpointByAddrPort) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c EndpointByAddrPort) Less(i, j int) bool {
	iName := c[i].Addr
	jName := c[j].Addr
	if iName != jName {
		return iName < jName
	}

	iPort := c[i].Port
	jPort := c[j].Port
	return iPort < jPort
}
func NewDefaultEp() Endpoint {
	return Endpoint{Addr: "127.0.0.1", Port: "8081"}
}
