package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"./throttler"
)

const version = "1.0.0"

func main() {
	// TODO: Add support for other options like packet reordering, duplication, etc.
	var (
		device           = flag.String("device", "", "Interface (device) to use (defaults to eth0 where applicable)")
		stop             = flag.Bool("stop", false, "Stop packet controls")
		delay            = flag.Int("delay", -1, "Latency to add in ms")
		delayrandom      = flag.Int("delay-jitter", -1, "Jitter to add in ms - used with -delay")
		delaycorrelation = flag.String("delay-correlation", "0", "Delay correlation in percentage - used with -delay")
		targetbw         = flag.Int("targetbw", -1, "Target bandwidth limit in kbit/s (slow-lane)")
		defaultbw        = flag.Int("defaultbw", -1, "Default bandwidth limit in kbit/s (fast-lane)")
		loss             = flag.String("packet-loss", "0", "Packet loss percentage (e.g. 0.1%)")
		duplicate        = flag.String("duplicate-pcnt", "0", "Percentage of packets to duplicate (e.g. 1.0%)")
		corrupt          = flag.String("corrupt-pcnt", "0", "Percentage of packets to corrupt (e.g. 0.1%)")
		reorder          = flag.String("reorder-pcnt", "0", "Percentage of packets to reorder (e.g. 0.1%)")
		targetaddr       = flag.String("target-addr", "", "Target addresses, (e.g. 10.0.0.1 or 10.0.0.0/24 or 10.0.0.1,192.168.0.0/24 or 2001:db8:a::123)")
		targetport       = flag.String("target-port", "", "Target port(s) (e.g. 80 or 1:65535 or 22,80,443,1000:1010)")
		targetproto      = flag.String("target-proto", "tcp,udp,icmp", "Target protocol TCP/UDP (e.g. tcp or tcp,udp or icmp)")
		dryrun           = flag.Bool("dry-run", false, "Specifies whether or not to actually commit the rule changes")
		//icmptype      = flag.String("icmp-type", "", "icmp message type (e.g. reply or reply,request)") //TODO: Maybe later :3
		vers = flag.Bool("version", false, "Print Comcast's version")
	)
	flag.Parse()

	if *vers {
		fmt.Printf("Comcast version %s\n", version)
		return
	}

	targetIPv4, targetIPv6 := parseAddrs(*targetaddr)

	throttler.Run(&throttler.Config{
		Device:            *device,
		Stop:              *stop,
		TargetBandwidth:   *targetbw,
		DefaultBandwidth:  *defaultbw,
		Delay:             *delay, // essentially latency
		DelayRandom:       *delayrandom,
		DelayCorrelation:  parseFloat(*delaycorrelation),
		PacketLoss:        parseFloat(*loss),
		DupePacketPcnt:    parseFloat(*duplicate),
		CorruptPacketPcnt: parseFloat(*corrupt),
		ReorderPacketPcnt: parseFloat(*reorder),
		TargetIps:         targetIPv4,
		TargetIps6:        targetIPv6,
		TargetPorts:       parsePorts(*targetport),
		TargetProtos:      parseProtos(*targetproto),
		DryRun:            *dryrun,
	})
}

func parseFloat(percentage string) float64 {
	val := percentage
	if strings.Contains(percentage, "%") {
		val = percentage[:len(percentage)-1]
	}
	l, err := strconv.ParseFloat(val, 64)
	if err != nil {
		fmt.Println("Incorrectly specified percentage:", percentage)
		os.Exit(1)
	}
	return l
}

func parseAddrs(addrs string) ([]string, []string) {
	adrs := strings.Split(addrs, ",")
	parsedIPv4 := []string{}
	parsedIPv6 := []string{}

	if addrs != "" {
		for _, adr := range adrs {
			ip := net.ParseIP(adr)
			if ip != nil {
				if ip.To4() != nil {
					parsedIPv4 = append(parsedIPv4, adr)
				} else {
					parsedIPv6 = append(parsedIPv6, adr)
				}
			} else { //Not a valid single IP, could it be a CIDR?
				parsedIP, net, err := net.ParseCIDR(adr)
				if err == nil {
					if parsedIP.To4() != nil {
						parsedIPv4 = append(parsedIPv4, net.String())
					} else {
						parsedIPv6 = append(parsedIPv6, net.String())
					}
				} else {
					fmt.Println("Incorrectly specified target IP or CIDR:", adr)
					os.Exit(1)
				}
			}
		}
	}

	return parsedIPv4, parsedIPv6
}

func parsePorts(ports string) []string {
	prts := strings.Split(ports, ",")
	parsed := []string{}

	if ports != "" {
		for _, prt := range prts {
			if strings.Contains(prt, ":") {
				if validRange(prt) {
					parsed = append(parsed, prt)
				} else {
					fmt.Println("Incorrectly specified port range:", prt)
					os.Exit(1)
				}
			} else { //Isn't a range, check if just a single port
				if validPort(prt) {
					parsed = append(parsed, prt)
				} else {
					fmt.Println("Incorrectly specified port:", prt)
					os.Exit(1)
				}
			}
		}
	}

	return parsed
}

func parsePort(port string) int {
	prt, err := strconv.Atoi(port)
	if err != nil {
		return 0
	}

	return prt
}

func validPort(port string) bool {
	prt := parsePort(port)
	return prt > 0 && prt < 65536
}

func validRange(ports string) bool {
	pr := strings.Split(ports, ":")

	if len(pr) == 2 {
		if !validPort(pr[0]) || !validPort(pr[1]) {
			return false
		}

		if portHigher(pr[0], pr[1]) {
			return false
		}
	} else {
		return false
	}

	return true
}

func portHigher(prt1, prt2 string) bool {
	p1 := parsePort(prt1)
	p2 := parsePort(prt2)

	return p1 > p2
}

func parseProtos(protos string) []string {
	ptcs := strings.Split(protos, ",")
	parsed := []string{}

	if protos != "" {
		for _, ptc := range ptcs {
			p := strings.ToLower(ptc)
			if p == "udp" ||
				p == "tcp" ||
				p == "icmp" {
				parsed = append(parsed, p)
			} else {
				fmt.Println("Incorrectly specified protocol:", p)
				os.Exit(1)
			}
		}
	}

	return parsed
}
