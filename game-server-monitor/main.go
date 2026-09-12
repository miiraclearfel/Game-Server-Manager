package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

type GameServer struct {
	Name   string
	IP     string
	Port   int
	Status string
	Ping   time.Duration
}

func CheckServer(ip string, port int) (string, time.Duration) {
	address := fmt.Sprintf("%s:%d", ip, port)
	timeout := 2 * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return "Offline", 0
	}
	defer conn.Close()
	duration := time.Since(start)
	return "Online", duration
}

func ClearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func main() {

	servers := []GameServer{
		{Name: "Valheim", IP: "127.0.0.1", Port: 2456},
		{Name: "Minecraft", IP: "192.168.0.1", Port: 25565},
		{Name: "Google DNS", IP: "8.8.8.8", Port: 53},
		{Name: "Cloudflare DNS", IP: "1.1.1.1", Port: 53},
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	fmt.Println("Starting Game Server Monitor...")

	for {
		var wg sync.WaitGroup

		for i := range servers {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				status, ping := CheckServer(servers[index].IP, servers[index].Port)
				servers[index].Status = status
				servers[index].Ping = ping
			}(i)
		}

		wg.Wait()

		ClearScreen()
		fmt.Println("===============================")
		fmt.Println("	Game Server Monitor (LIVE)	")
		fmt.Printf("Last Update: %s\n", time.Now().Format("15:04:05"))
		fmt.Println("===============================")

		for _, server := range servers {
			fmt.Printf("Server: %s\n", server.Name)
			fmt.Printf("IP: %s:%d\n", server.IP, server.Port)
			fmt.Printf("Status: %s\n", server.Status)
			if server.Status == "Online" {
				fmt.Printf("Ping: %v ms\n", server.Ping.Milliseconds())
			} else {
				fmt.Printf("Ping: N/A\n")
			}
			fmt.Println("-------------------------------")
		}

		fmt.Println("Press Ctrl+C to stop the monitor...")

		<-ticker.C

	}
}
