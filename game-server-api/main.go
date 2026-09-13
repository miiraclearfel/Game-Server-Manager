package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

var db *sql.DB

type Response struct {
	Message string `json:"message"`
	Status  string `json:"status,omitempty"`
	PID     int    `json:"pid,omitempty"`
}

type LogsResponse struct {
	Logs []string `json:"logs"`
}

type HistoryResponse struct {
	History []AuditLog `json:"history"`
}

type AuditLog struct {
	ID        int       `json:"id"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func initDB() {
	connSTR := "host=localhost port=5432 user=postgres password=dragon123 dbname=gameserverdb sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connSTR)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping the database:", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id SERIAL PRIMARY KEY,
		action VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("Failed to create audit_logs table:", err)
	}
}

func recordLog(action string, status string) {
	query := "INSERT INTO audit_logs (action, status, created_at) VALUES ($1, $2, $3)"
	_, err := db.Exec(query, action, status, time.Now())
	if err != nil {
		log.Println("Failed to record log:", err)
	}
}

func sendJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "flyingpanther" {
			sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
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
	data, err := os.ReadFile("server.pid")
	if err != nil {
		recordLog("status_check", "OFFLINE")
		sendJSON(w, http.StatusOK, Response{Message: "Server status", Status: "OFFLINE"})
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
		recordLog("status_check", "ONLINE")
		sendJSON(w, http.StatusOK, Response{Message: "Server status", Status: "ONLINE", PID: pid})
	} else {
		os.Remove("server.pid")
		recordLog("status_check", "OFFLINE")
		sendJSON(w, http.StatusOK, Response{Message: "Server status", Status: "OFFLINE"})
	}
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat("server.pid"); err == nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Server is already running"})
		return
	}

	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to open log file"})
		return
	}

	cmd := exec.Command("./dummy_server.exe")
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to start the server"})
		logFile.Close()
		return
	}

	pid := cmd.Process.Pid
	os.WriteFile("server.pid", []byte(strconv.Itoa(pid)), 0644)

	sendJSON(w, http.StatusOK, Response{Message: "Server started", PID: pid})

}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("server.pid")
	if err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Server is not running"})
		return
	}

	pidStr := strings.TrimSpace(string(data))
	pid, _ := strconv.Atoi(pidStr)

	process, err := os.FindProcess(pid)
	if err != nil || process.Kill() != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to stop the server"})
		return
	}

	os.Remove("server.pid")
	sendJSON(w, http.StatusOK, Response{Message: "Server stopped", Status: "OFFLINE"})
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("server.log")
	if err != nil {
		sendJSON(w, http.StatusOK, LogsResponse{Logs: []string{}})
		return
	}
	defer file.Close()

	var logs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		sendJSON(w, http.StatusInternalServerError, LogsResponse{Logs: []string{}})
		return
	}

	sendJSON(w, http.StatusOK, LogsResponse{Logs: logs})
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, action, status, created_at FROM audit_logs ORDER BY id DESC LIMIT 10")
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to retrieve history"})
		return
	}
	defer rows.Close()

	var history []AuditLog
	for rows.Next() {
		var item AuditLog
		if err := rows.Scan(&item.ID, &item.Action, &item.Status, &item.CreatedAt); err != nil {
			continue
		}
		history = append(history, item)
	}
	if err := rows.Err(); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to retrieve history"})
		return
	}
	sendJSON(w, http.StatusOK, HistoryResponse{History: history})
}

func main() {

	initDB()
	defer db.Close()

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Get("/status", statusHandler)
		r.Get("/logs", logHandler)
		r.Get("/history", historyHandler)

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware)
			r.Post("/start", startHandler)
			r.Post("/stop", stopHandler)
		})
	})

	fmt.Println("Game Server API is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
