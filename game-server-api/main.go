package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

type response struct {
	Message string `json:"message"`
	Status  string `json:"status,omitempty"`
	PID     int    `json:"pid,omitempty"`
}

type logsResponse struct {
	Logs []string `json:"logs"`
}

func sendJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func isWindowsProcessRunning(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), strconv.Itoa(pid))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSON(w, http.StatusMethodNotAllowed, response{Message: "Method not allowed"})
		return
	}

	data, err := os.ReadFile("server.pid")
	if err != nil {
		sendJSON(w, http.StatusOK, response{Message: "Server status", Status: "OFFLINE"})
		return
	}

	pidStr := strings.TrimSpace(string(data))
	pid, _ := strconv.Atoi(pidStr)

	isRunning := false
	if runtime.GOOS == "windows" {
		isRunning = isWindowsProcessRunning(pid)
	} else {
		process, err := os.FindProcess(pid)
		if err == nil && process.Signal(syscall.Signal(0)) == nil {
			isRunning = true
		}
	}

	if isRunning {
		sendJSON(w, http.StatusOK, response{Message: "Server status", Status: "ONLINE", PID: pid})
	} else {
		os.Remove("server.pid")
		sendJSON(w, http.StatusOK, response{Message: "Server status", Status: "OFFLINE"})
	}
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, response{Message: "Method not allowed"})
		return
	}

	if _, err := os.Stat("server.pid"); err == nil {
		sendJSON(w, http.StatusBadRequest, response{Message: "Server is already running"})
		return
	}

	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, response{Message: "Failed to open log file"})
		return
	}

	cmd := exec.Command("./dummy_server.exe")
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		sendJSON(w, http.StatusInternalServerError, response{Message: "Failed to start the server"})
		logFile.Close()
		return
	}

	pid := cmd.Process.Pid
	os.WriteFile("server.pid", []byte(strconv.Itoa(pid)), 0644)

	sendJSON(w, http.StatusOK, response{Message: "Server started", PID: pid})

}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, response{Message: "Method not allowed"})
		return
	}

	data, err := os.ReadFile("server.pid")
	if err != nil {
		sendJSON(w, http.StatusBadRequest, response{Message: "Server is not running"})
		return
	}

	pidStr := strings.TrimSpace(string(data))
	pid, _ := strconv.Atoi(pidStr)

	process, err := os.FindProcess(pid)
	if err != nil || process.Kill() != nil {
		sendJSON(w, http.StatusInternalServerError, response{Message: "Failed to stop the server"})
		return
	}

	os.Remove("server.pid")
	sendJSON(w, http.StatusOK, response{Message: "Server stopped", Status: "OFFLINE"})
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSON(w, http.StatusMethodNotAllowed, response{Message: "Method not allowed"})
		return
	}

	file, err := os.Open("server.log")
	if err != nil {
		sendJSON(w, http.StatusOK, logsResponse{Logs: []string{}})
		return
	}
	defer file.Close()

	var logs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		sendJSON(w, http.StatusInternalServerError, response{Message: "Failed to read log file"})
		return
	}

	sendJSON(w, http.StatusOK, logsResponse{Logs: logs})
}

func main() {
	http.HandleFunc("/api/status", statusHandler)
	http.HandleFunc("/api/start", startHandler)
	http.HandleFunc("/api/stop", stopHandler)
	http.HandleFunc("/api/logs", logHandler)

	fmt.Println("Game Server API is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
