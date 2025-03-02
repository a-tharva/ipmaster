package ipinfo

import (
	"fmt"
	"log"
	"net"
)

type InterfaceDetail struct {
	Name  string
	MTU   int
	Flags net.Flags
	IPs   []string
}

func GetIpDetails() ([]InterfaceDetail, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Error fetching network interfaces: %v", err)
		return nil, fmt.Errorf("failed to fetch interfaces: %w", err)
	}

	var ifaceDetailslist []InterfaceDetail
	for _, iface := range interfaces {
		var ipAddress []string
		addrs, err := iface.Addrs()
		if err != nil {
			log.Printf("Error fetching addresses for %s: %v", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			// var ip net.IP
			// switch v := addr.(type){
			// case *net.IPNet:
			// 	ip = v.IP
			// case *net.IPAddr:
			// 	ip = v.IP
			// }
			if addr != nil {
				ipAddress = append(ipAddress, addr.String())
			}
		}

		ifaceDetailslist = append(ifaceDetailslist, InterfaceDetail{
			Name:  iface.Name,
			MTU:   iface.MTU,
			Flags: iface.Flags,
			IPs:   ipAddress,
		})

	}

	return ifaceDetailslist, nil
}
