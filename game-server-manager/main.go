package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func startServer() {
	cmd := exec.Command("./dummy_server.exe")

	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting the server:", err)
		return
	}

	pid := cmd.Process.Pid
	fmt.Printf("Dummy Game Server started! (PID: %d)\n", pid)

	err = os.WriteFile("server.pid", []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		fmt.Println("Error writing PID to file:", err)
	}
}

func stopServer() {
	data, err := os.ReadFile("server.pid")
	if err != nil {
		fmt.Println("Error reading PID file:", err)
		return
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Println("Server is not running or PID file is invalid:", err)
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("Error finding process:", err)
		return
	}

	err = process.Kill()
	if err != nil {
		fmt.Println("Error stopping the server:", err)
		return
	}

	os.Remove("server.pid")
	fmt.Printf("Dummy Game Server (PID: %d) stopped!\n", pid)
}

func statusServer() {
	data, err := os.ReadFile("server.pid")
	if err != nil {
		fmt.Println("Server is not running.")
		return
	}

	pidStr := strings.TrimSpace(string(data))
	pid, _ := strconv.Atoi(pidStr)

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("Server is not running.")
		return
	}

	isRunning := false
	if runtime.GOOS == "windows" {
		isRunning = isWindowsProcessRunning(pid)
	} else {
		err = process.Signal(syscall.Signal(0))
		if err == nil {
			isRunning = true
		}
	}

	if isRunning {
		fmt.Printf("Status: Running (PID: %d)\n", pid)
	} else {
		fmt.Println("Server is not running.")
		os.Remove("server.pid")
	}
}

func isWindowsProcessRunning(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), strconv.Itoa(pid))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [start|stop|status]")
		return
	}

	command := os.Args[1]
	switch command {
	case "start":
		startServer()
	case "stop":
		stopServer()
	case "status":
		statusServer()
	default:
		fmt.Println("Unknown command:", command, "Usage: go run main.go [start|stop|status]")
	}
}
