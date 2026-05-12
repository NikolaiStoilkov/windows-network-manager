package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/elazarl/goproxy"
)

type NetworkConfig struct {
	InterfaceIP string
	DNSServer   string // e.g., "8.8.8.8" or your router IP "192.168.33.1"
	ProxyPort   string
	BrowserPath string
	ProfileName string
}

const configFile = "./config.json"

func loadConfigs(path string) ([]NetworkConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %q: %w", path, err)
	}

	var configs []NetworkConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("could not parse config file %q: %w", path, err)
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("config file %q contains no entries", path)
	}

	return configs, nil
}

func startProxy(config NetworkConfig) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	dialer := &net.Dialer{
		LocalAddr: &net.TCPAddr{IP: net.ParseIP(config.InterfaceIP)},
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				LocalAddr: &net.UDPAddr{IP: net.ParseIP(config.InterfaceIP)},
				Timeout:   time.Second * 5,
			}

			// Use the DNS server assigned to this specific network (port 53)
			return d.DialContext(ctx, "udp", config.DNSServer+":53")
		},
	}

	proxy.Tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, _ := net.SplitHostPort(addr)
		ips, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}

		destination := net.JoinHostPort(ips[0].String(), port)

		return dialer.DialContext(ctx, network, destination)
	}

	fmt.Printf("[+] Proxy on :%s bound to %s (DNS: %s)\n", config.ProxyPort, config.InterfaceIP, config.DNSServer)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+config.ProxyPort, proxy))
}

func launchBrowser(config NetworkConfig) {
	args := []string{
		fmt.Sprintf("--proxy-server=http://127.0.0.1:%s", config.ProxyPort),
		fmt.Sprintf("--user-data-dir=C:\\temp\\browser_profile_%s", config.ProfileName),
		"--no-first-run",
		"--new-window",
	}

	cmd := exec.Command(config.BrowserPath, args...)
	if err := cmd.Start(); err != nil {
		fmt.Printf("[-] Failed to launch browser %s: %v\n", config.ProfileName, err)
	}
}

func main() {
	configs, err := loadConfigs(configFile)
	if err != nil {
		log.Fatalf("[-] Failed to load config: %v\n", err)
	}

	fmt.Printf("[*] Loaded %d network config(s) from %s\n", len(configs), configFile)

	for _, conf := range configs {
		go startProxy(conf)
	}

	time.Sleep(2 * time.Second)

	for _, conf := range configs {
		fmt.Printf("Launching %s...\n", conf.ProfileName)
		launchBrowser(conf)
	}

	fmt.Println("\nAll systems running. Press Ctrl+C to exit.")
	select {}
}
