//go:build linux

package core

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	gatewayNftTable       = "resdmini"
	gatewayNftAllowSet    = "allow4"
	gatewayNftDenySet     = "deny4"
	gatewayNftFile        = "/tmp/resd-mini-transparent.nft"
	gatewayDnsmasqNftName = "resd-mini-nftset.conf"
)

func (s *SystemSetup) isGatewayLikeDistro() bool {
	distro, err := s.getLinuxDistro()
	if err != nil {
		return false
	}
	return distro == "openwrt" || distro == "istoreos"
}

func (s *SystemSetup) getLinuxDistro() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), "\""), nil
		}
	}
	return "", fmt.Errorf("could not determine linux distribution")
}

func (s *SystemSetup) runCommand(args []string, sudo bool) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command provided")
	}

	var cmd *exec.Cmd
	if s.Password != "" && sudo && os.Geteuid() != 0 {
		cmd = exec.Command("sudo", append([]string{"-S"}, args...)...)
		cmd.Stdin = bytes.NewReader([]byte(s.Password + "\n"))
	} else {
		cmd = exec.Command(args[0], args[1:]...)
	}

	output, err := cmd.CombinedOutput()
	return output, err
}

func (s *SystemSetup) setProxy() error {
	if s.isGatewayLikeDistro() {
		s.GatewayTransparent = false
		if err := s.applyGatewayTransparent(); err != nil {
			return err
		}
		s.GatewayTransparent = true
		return nil
	}

	commands := [][]string{
		{"gsettings", "set", "org.gnome.system.proxy", "mode", "manual"},
		{"gsettings", "set", "org.gnome.system.proxy.http", "host", resolvedProxyHost()},
		{"gsettings", "set", "org.gnome.system.proxy.http", "port", globalConfig.Port},
		{"gsettings", "set", "org.gnome.system.proxy.https", "host", resolvedProxyHost()},
		{"gsettings", "set", "org.gnome.system.proxy.https", "port", globalConfig.Port},
	}

	isSuccess := false
	var errs strings.Builder

	for _, cmd := range commands {
		if output, err := s.runCommand(cmd, false); err != nil {
			errs.WriteString(fmt.Sprintf("cmd: %v\noutput: %s\nerr: %s\n", cmd, output, err))
		} else {
			isSuccess = true
		}
	}

	if isSuccess {
		return nil
	}

	return fmt.Errorf("failed to set proxy:\n%s", errs.String())
}

func (s *SystemSetup) unsetProxy() error {
	if s.isGatewayLikeDistro() {
		s.GatewayTransparent = false
		return s.clearGatewayTransparent()
	}

	cmd := []string{"gsettings", "set", "org.gnome.system.proxy", "mode", "none"}
	output, err := s.runCommand(cmd, false)
	if err != nil {
		return fmt.Errorf("failed to unset proxy: %s\noutput: %s", err.Error(), string(output))
	}
	return nil
}

type gatewayRuleSpec struct {
	allowDomains map[string]struct{}
	denyDomains  map[string]struct{}
	allowIPv4    map[string]struct{}
	denyIPv4     map[string]struct{}
}

func newGatewayRuleSpec() gatewayRuleSpec {
	return gatewayRuleSpec{
		allowDomains: make(map[string]struct{}),
		denyDomains:  make(map[string]struct{}),
		allowIPv4:    make(map[string]struct{}),
		denyIPv4:     make(map[string]struct{}),
	}
}

func (s *SystemSetup) applyGatewayTransparent() error {
	if _, err := exec.LookPath("nft"); err != nil {
		return fmt.Errorf("gateway transparent mode requires nft command: %w", err)
	}

	port, err := strconv.Atoi(strings.TrimSpace(globalConfig.Port))
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("invalid service port: %s", globalConfig.Port)
	}

	spec := parseGatewayRule(globalConfig.Rule)

	if len(spec.allowIPv4) == 0 && len(spec.allowDomains) == 0 {
		return fmt.Errorf("gateway whitelist is empty, set specific domain/IP/CIDR rules before enabling transparent mode")
	}

	if err := s.clearGatewayTransparent(); err != nil {
		return err
	}

	useDnsmasqNft := false
	if len(spec.allowDomains) > 0 || len(spec.denyDomains) > 0 {
		useDnsmasqNft = s.dnsmasqSupportsNftset()
		if !useDnsmasqNft {
			// Fallback for systems without dnsmasq nftset support.
			s.resolveDomainTargets(spec)
		}
	}

	if !useDnsmasqNft && len(spec.allowIPv4) == 0 {
		return fmt.Errorf("domain whitelist cannot be resolved to IPv4 addresses, verify DNS or install dnsmasq with nftset support")
	}

	allow := sortedKeys(spec.allowIPv4)
	deny := sortedKeys(spec.denyIPv4)
	ports := "80"
	if s.gatewayEnableHTTPSRedirect() {
		ports = "80, 443"
	}

	script := "table ip " + gatewayNftTable + " {\n" +
		buildNftIPv4Set(gatewayNftAllowSet, allow) +
		buildNftIPv4Set(gatewayNftDenySet, deny) +
		"    chain prerouting {\n" +
		"        type nat hook prerouting priority dstnat; policy accept;\n" +
		"        ip daddr @" + gatewayNftAllowSet + " ip daddr != @" + gatewayNftDenySet + " tcp dport { " + ports + " } counter redirect to :" + strconv.Itoa(port) + "\n" +
		"    }\n" +
		"}\n"

	if err := os.WriteFile(gatewayNftFile, []byte(script), 0644); err != nil {
		_ = s.cleanupDnsmasqNftset()
		return fmt.Errorf("write nft script failed: %w", err)
	}

	output, err := s.runCommand([]string{"nft", "-f", gatewayNftFile}, true)
	if err != nil {
		_ = s.cleanupDnsmasqNftset()
		return fmt.Errorf("apply nft transparent rules failed: %s, output: %s", err.Error(), string(output))
	}

	if useDnsmasqNft {
		if err := s.configureDnsmasqNftset(spec); err != nil {
			_, _ = s.runCommand([]string{"nft", "delete", "table", "ip", gatewayNftTable}, true)
			return err
		}
	}
	return nil
}

func (s *SystemSetup) gatewayEnableHTTPSRedirect() bool {
	raw := strings.TrimSpace(os.Getenv("RESD_GATEWAY_HTTPS"))
	if raw == "" {
		return false
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return parsed
}

func (s *SystemSetup) clearGatewayTransparent() error {
	_ = os.Remove(gatewayNftFile)
	_ = s.cleanupDnsmasqNftset()

	output, err := s.runCommand([]string{"nft", "delete", "table", "ip", gatewayNftTable}, true)
	if err != nil {
		msg := string(output)
		if strings.Contains(msg, "No such file") || strings.Contains(msg, "No such file or directory") {
			return nil
		}
		return fmt.Errorf("clear nft transparent rules failed: %s, output: %s", err.Error(), msg)
	}
	return nil
}

func parseGatewayRule(raw string) gatewayRuleSpec {
	spec := newGatewayRuleSpec()
	for _, line := range strings.Split(raw, "\n") {
		token := strings.TrimSpace(line)
		if token == "" || strings.HasPrefix(token, "#") {
			continue
		}

		isNeg := false
		if strings.HasPrefix(token, "!") {
			isNeg = true
			token = strings.TrimSpace(strings.TrimPrefix(token, "!"))
			if token == "" {
				continue
			}
		}

		// Transparent gateway mode enforces an explicit whitelist.
		if token == "*" {
			continue
		}

		if ip := net.ParseIP(token); ip != nil {
			if ipv4 := ip.To4(); ipv4 != nil {
				addGatewayTarget(spec, isNeg, ipv4.String()+"/32")
			}
			continue
		}

		if _, ipNet, err := net.ParseCIDR(token); err == nil {
			if ipNet.IP.To4() != nil {
				addGatewayTarget(spec, isNeg, ipNet.String())
			}
			continue
		}

		if strings.HasPrefix(token, "*.") {
			token = strings.TrimPrefix(token, "*.")
		}
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			continue
		}

		if isNeg {
			spec.denyDomains[token] = struct{}{}
		} else {
			spec.allowDomains[token] = struct{}{}
		}
	}
	return spec
}

func buildNftIPv4Set(name string, elements []string) string {
	var sb strings.Builder
	sb.WriteString("    set " + name + " {\n")
	sb.WriteString("        type ipv4_addr\n")
	sb.WriteString("        flags interval\n")
	sb.WriteString("        auto-merge\n")
	if len(elements) > 0 {
		sb.WriteString("        elements = { " + strings.Join(elements, ", ") + " }\n")
	}
	sb.WriteString("    }\n")
	return sb.String()
}

func addGatewayTarget(spec gatewayRuleSpec, isNeg bool, value string) {
	if isNeg {
		spec.denyIPv4[value] = struct{}{}
		return
	}
	spec.allowIPv4[value] = struct{}{}
}

func (s *SystemSetup) resolveDomainTargets(spec gatewayRuleSpec) {
	for domain := range spec.allowDomains {
		for _, cidr := range resolveDomainIPv4(domain) {
			spec.allowIPv4[cidr] = struct{}{}
		}
	}
	for domain := range spec.denyDomains {
		for _, cidr := range resolveDomainIPv4(domain) {
			spec.denyIPv4[cidr] = struct{}{}
		}
	}
}

func (s *SystemSetup) dnsmasqSupportsNftset() bool {
	output, err := s.runCommand([]string{"dnsmasq", "-v"}, true)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(output)), "nftset")
}

func (s *SystemSetup) configureDnsmasqNftset(spec gatewayRuleSpec) error {
	cfgPath := s.dnsmasqNftConfigPath()
	cfgDir := filepath.Dir(cfgPath)

	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("create dnsmasq include dir failed: %w", err)
	}

	var lines []string
	for _, domain := range sortedKeys(spec.allowDomains) {
		lines = append(lines, "nftset=/"+domain+"/4#ip#"+gatewayNftTable+"#"+gatewayNftAllowSet)
	}
	for _, domain := range sortedKeys(spec.denyDomains) {
		lines = append(lines, "nftset=/"+domain+"/4#ip#"+gatewayNftTable+"#"+gatewayNftDenySet)
	}

	payload := strings.Join(lines, "\n")
	if payload != "" {
		payload += "\n"
	}
	if err := os.WriteFile(cfgPath, []byte(payload), 0644); err != nil {
		return fmt.Errorf("write dnsmasq nftset config failed: %w", err)
	}

	if err := s.reloadDnsmasq(); err != nil {
		_ = os.Remove(cfgPath)
		return err
	}
	return nil
}

func (s *SystemSetup) cleanupDnsmasqNftset() error {
	paths := []string{
		s.dnsmasqNftConfigPath(),
		filepath.Join("/tmp/dnsmasq.d", gatewayDnsmasqNftName),
	}

	removed := false
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				return err
			}
			removed = true
		}
	}
	if !removed {
		return nil
	}
	if err := s.reloadDnsmasq(); err != nil {
		return err
	}
	return nil
}

func (s *SystemSetup) dnsmasqNftConfigPath() string {
	paths, _ := filepath.Glob("/var/etc/dnsmasq.conf.*")
	if len(paths) > 0 {
		sort.Strings(paths)
		if dir := parseDnsmasqConfDir(paths[0]); dir != "" {
			return filepath.Join(dir, gatewayDnsmasqNftName)
		}
	}
	return filepath.Join("/tmp/dnsmasq.d", gatewayDnsmasqNftName)
}

func parseDnsmasqConfDir(confPath string) string {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "conf-dir=") {
			continue
		}
		dir := strings.TrimSpace(strings.TrimPrefix(line, "conf-dir="))
		if idx := strings.Index(dir, ","); idx >= 0 {
			dir = strings.TrimSpace(dir[:idx])
		}
		if dir != "" {
			return dir
		}
	}
	return ""
}

func (s *SystemSetup) reloadDnsmasq() error {
	if _, err := os.Stat("/etc/init.d/dnsmasq"); err != nil {
		return nil
	}
	// On OpenWrt/iStoreOS, reload may skip conf-dir refresh in some builds.
	output, err := s.runCommand([]string{"/etc/init.d/dnsmasq", "restart"}, true)
	if err == nil {
		return nil
	}
	output2, err2 := s.runCommand([]string{"/etc/init.d/dnsmasq", "reload"}, true)
	if err2 == nil {
		return nil
	}
	return fmt.Errorf("restart dnsmasq failed: %s, output: %s, reload output: %s", err.Error(), string(output), string(output2))
}

func resolveDomainIPv4(domain string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", domain)
	if err != nil {
		return nil
	}

	dedup := make(map[string]struct{})
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			dedup[ipv4.String()+"/32"] = struct{}{}
		}
	}
	return sortedKeys(dedup)
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s *SystemSetup) installCert() (string, error) {
	_, err := s.initCert()
	if err != nil {
		return "", err
	}

	distro, err := s.getLinuxDistro()
	if err != nil {
		return "", fmt.Errorf("detect distro failed: %w", err)
	}

	certName := appOnce.AppName + ".crt"
	var certPath string
	var updateCmd = []string{"update-ca-certificates"}

	switch distro {
	case "deepin":
		certDir := "/usr/share/ca-certificates/" + appOnce.AppName
		certPath = certDir + "/" + certName
		s.runCommand([]string{"mkdir", "-p", certDir}, true)
	case "arch":
		certPath = "/usr/share/ca-certificates/trust-source/" + certName
		updateCmd = []string{"update-ca-trust"}
	case "openwrt", "istoreos":
		// OpenWrt/iStoreOS doesn't ship Debian-style CA update commands by default.
		// In gateway mode, clients should install CA manually from /api/cert.
		return "", nil
	default:
		certPath = "/usr/local/share/ca-certificates/" + certName
	}

	var outs, errs strings.Builder
	isSuccess := false

	if output, err := s.runCommand([]string{"cp", "-f", s.CertFile, certPath}, true); err != nil {
		errs.WriteString(fmt.Sprintf("copy cert failed: %s\n%s\n", err.Error(), output))
	} else {
		isSuccess = true
		outs.Write(output)
	}

	if distro == "deepin" {
		confPath := "/etc/ca-certificates.conf"
		checkCmd := []string{"grep", "-qxF", certName, confPath}
		if _, err := s.runCommand(checkCmd, true); err != nil {
			echoCmd := []string{"bash", "-c", fmt.Sprintf("echo '%s/%s' >> %s", appOnce.AppName, certName, confPath)}
			if output, err := s.runCommand(echoCmd, true); err != nil {
				errs.WriteString(fmt.Sprintf("append conf failed: %s\n%s\n", err.Error(), output))
			} else {
				isSuccess = true
				outs.Write(output)
			}
		}
	}

	if output, err := s.runCommand(updateCmd, true); err != nil {
		errs.WriteString(fmt.Sprintf("update failed: %s\n%s\n", err.Error(), output))
	} else {
		isSuccess = true
		outs.Write(output)
	}

	if isSuccess {
		return "", nil
	}

	return outs.String(), fmt.Errorf("certificate installation failed:\n%s", errs.String())
}
