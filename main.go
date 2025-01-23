package main

import (
	"fmt"

	probing "github.com/prometheus-community/pro-bing"
)

func main() {
	greeting("fdsa")
	res := start_ping()

	fmt.Println(res)
}

func start_ping() string {
	ipAddress := "8.8.8.8"

	pinger, err := probing.NewPinger(ipAddress)
	if err != nil {
		panic(err)
	}

	pinger.SetPrivileged(true)

	pinger.Count = 3
	err = pinger.Run()
	if err != nil {
		panic(err)
	}

	stats := pinger.Statistics()

	fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
	fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
		stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
	fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
		stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)

	return "ok"
}

func greeting(name string) {
	fmt.Println("adf", name)
}
