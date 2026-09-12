package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func startServer() {
	if _, err := os.Stat("server.pid"); err == nil {
		fmt.Println("Game Server is running. Check status or stop it before starting a new instance.")
		return
	}

	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}

	cmd := exec.Command("./dummy_server.exe")

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting the server:", err)
		logFile.Close()
		return
	}

	pid := cmd.Process.Pid
	fmt.Printf("Dummy Game Server! (PID: %d)\n", pid)

	err = os.WriteFile("server.pid", []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		fmt.Println("Error writing PID file:", err)
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

func restartServer() {
	fmt.Println("Restarting Game Server...")
	stopServer()
	time.Sleep(1 * time.Second)
	startServer()
}

func showLogs() {
	file, err := os.Open("server.log")
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer file.Close()

	fmt.Println("======== Game Server Logs ======")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading log file:", err)
	}
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
		fmt.Println("Usage: go run main.go [start|stop|status|restart|logs]")
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
	case "restart":
		restartServer()
	case "logs":
		showLogs()
	default:
		fmt.Println("Unknown command:", command, "Usage: go run main.go [start|stop|status|restart|logs]")
	}
}
