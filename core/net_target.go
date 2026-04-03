package core

import (
	"net"
	"strings"
)

func resolvedProxyHost() string {
	host := "127.0.0.1"
	if globalConfig != nil {
		host = strings.TrimSpace(globalConfig.Host)
	}
	host = strings.Trim(host, "[]")
	if host == "" || host == "0.0.0.0" || host == "::" {
		return "127.0.0.1"
	}
	return host
}

func resolvedProxyAddr() string {
	port := "8899"
	if globalConfig != nil && strings.TrimSpace(globalConfig.Port) != "" {
		port = strings.TrimSpace(globalConfig.Port)
	}
	return net.JoinHostPort(resolvedProxyHost(), port)
}

func panelBaseURL() string {
	return "http://" + resolvedProxyAddr()
}
