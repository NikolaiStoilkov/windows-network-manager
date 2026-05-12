package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
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
	//mock config
	configs := []NetworkConfig{
		{
			InterfaceIP: "192.168.33.200",
			DNSServer:   "1.1.1.1", // Cloudflare
			ProxyPort:   "8081",
			BrowserPath: `C:\Program Files\Google\Chrome\Application\chrome.exe`,
			ProfileName: "chrome_lan_33",
		},
		{
			InterfaceIP: "192.168.44.250",
			DNSServer:   "8.8.8.8", // Google
			ProxyPort:   "8082",
			BrowserPath: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			ProfileName: "edge_lan_44",
		},
		{
			InterfaceIP: "192.168.35.20",
			DNSServer:   "192.168.35.1", // Local Router DNS
			ProxyPort:   "8083",
			BrowserPath: `C:\Users\YourUser\AppData\Local\Programs\Opera\launcher.exe`,
			ProfileName: "opera_lan_35",
		},
	}

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
