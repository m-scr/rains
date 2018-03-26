// Package configs contains templates for the configuration files to be used in testing.
package configs

import (
	"bytes"
	"fmt"
	"text/template"
)

var (
	serverTmpl = `
{
    "RootZonePublicKeyPath":        "{{.RootZonePublicKeyPath}}",
    "ServerAddress":                {
                                        "Type":     "TCP",
                                        "TCPAddr":  {
                                                        "IP":"127.0.0.1",
                                                        "Port":{{.ListenPort}},
                                                        "Zone":""
                                                    }
                                    },
    "MaxConnections":               1000,
    "KeepAlivePeriod":              60,
    "TCPTimeout":                   300,
    "TLSCertificateFile":           "{{.TLSCertificateFile}}",
    "TLSPrivateKeyFile":            "{{.TLSPrivateKeyFile}}",
    "MaxMsgByteLength":             65536,
    "PrioBufferSize":               1000,
    "NormalBufferSize":             100000,
    "PrioWorkerCount":              2,
    "NormalWorkerCount":            10,
    "ActiveTokenCacheSize":         1000,
    "ZoneKeyCacheSize":             1000,
    "ZoneKeyCacheWarnSize":         750,
    "MaxPublicKeysPerZone":         5,
    "PendingKeyCacheSize":          1000,
    "AssertionCacheSize":           10000,
    "PendingQueryCacheSize":        1000,
    "RedirectionCacheSize":         1000,
    "RedirectionCacheWarnSize":     750,
    "CapabilitiesCacheSize":        50,
    "NotificationBufferSize":       20,
    "NotificationWorkerCount":      2,
    "PeerToCapCacheSize":           1000,
    "Capabilities":                 ["urn:x-rains:tlssrv"],
    "InfrastructureKeyCacheSize":   10,
    "ExternalKeyCacheSize":         5,
    "DelegationQueryValidity":      5,
    "NegativeAssertionCacheSize":   500,
    "AddressQueryValidity":         5,
    "QueryValidity":                5,
    "MaxCacheValidity":             {
                                        "AssertionValidity": 720,
                                        "ShardValidity": 720,
                                        "ZoneValidity": 720,
                                        "AddressAssertionValidity": 720,
                                        "AddressZoneValidity": 720
                                    },
    "ReapVerifyTimeout":            1800,
    "ReapEngineTimeout":            1800,
    "ContextAuthority":             ["{{.ContextAuthority}}"],
    "ZoneAuthority":                ["{{.ZoneAuthority}}"]
}
    `
)

type ServerConfigParams struct {
	ListenPort            uint
	RootZonePublicKeyPath string
	TLSCertificateFile    string
	TLSPrivateKeyFile     string
	ContextAuthority      string
	ZoneAuthority         string
}

func (scp *ServerConfigParams) ServerConfig() (string, error) {
	tmpl, err := template.New("serverConfig").Parse(serverTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse config template: %v", err)
	}
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := tmpl.Execute(buf, scp); err != nil {
		return "", fmt.Errorf("failed to execute config template: %v", err)
	}
	return buf.String(), nil
}
