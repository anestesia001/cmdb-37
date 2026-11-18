package main

import (
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    _ "github.com/lib/pq"
)

type Server struct {
    ID          int    `json:"id"`
    Hostname    string `json:"hostname"`
    IPAddress   string `json:"ip_address"`
    OS          string `json:"os"`
    Environment string `json:"environment"`
    Status      string `json:"status"`
    CPU         int    `json:"cpu"`
    RAM         int    `json:"ram"`
    Team        string `json:"team"`
}

var db *sql.DB

func respondJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
    respondJSON(w, status, map[string]string{
        "error": message,
    })
}

func validateServer(s Server) error {
    if strings.TrimSpace(s.Hostname) == "" {
        return errors.New("hostname is required")
    }

    if net.ParseIP(s.IPAddress) == nil {
        return errors.New("invalid ip_address")
    }

    if strings.TrimSpace(s.OS) == "" {
        return errors.New("os is required")
    }

    if s.Environment != "dev" &&
        s.Environment != "stage" &&
        s.Environment != "prod" {
        return errors.New("invalid environment")
    }

    if s.Status != "running" &&
        s.Status != "stopped" &&
        s.Status != "maintenance" {
        return errors.New("invalid status")
    }

    if s.CPU <= 0 || s.RAM <= 0 {
        return errors.New("cpu and ram must be positive")
    }

    if strings.TrimSpace(s.Team) == "" {
        return errors.New("team is required")
    }

    return nil
}

func decodeServer(r *http.Request) (Server, error) {
    var s Server

    r.Body = http.MaxBytesReader(nil, r.Body, 1048576)
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields()

    err := decoder.Decode(&s)
    if err != nil {
        return s, err
    }

    var extra any
    if decoder.Decode(&extra) != nil {
        // A single JSON object is expected.
    } else {
        return s, errors.New("multiple JSON values")
    }

    return s, validateServer(s)
}

func scanServer(scanner interface{ Scan(...any) error }) (Server, error) {
    var s Server

    err := scanner.Scan(
        &s.ID,
        &s.Hostname,
        &s.IPAddress,
        &s.OS,
        &s.Environment,
        &s.Status,
        &s.CPU,
        &s.RAM,
        &s.Team,
    )

    return s, err
}

const serverColumns = `
    id, hostname, ip_address, os,
    environment, status, cpu, ram, team
`

func serversHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        listServers(w, r)
    case http.MethodPost:
        createServer(w, r)
    default:
        w.Header().Set("Allow", "GET, POST")
        respondError(w, http.StatusMethodNotAllowed, "method not allowed")
    }
}

func listServers(w http.ResponseWriter, r *http.Request) {
    query := "SELECT " + serverColumns + " FROM servers WHERE 1=1"
    args := []any{}

    if env := r.URL.Query().Get("env"); env != "" {
        args = append(args, env)
        query += fmt.Sprintf(" AND environment = $%d", len(args))
    }

    if status := r.URL.Query().Get("status"); status != "" {
        args = append(args, status)
        query += fmt.Sprintf(" AND status = $%d", len(args))
    }

    query += " ORDER BY id"

    rows, err := db.QueryContext(r.Context(), query, args...)
    if err != nil {
        log.Printf("list servers: %v", err)
        respondError(w, 500, "database error")
        return
    }
    defer rows.Close()

    servers := []Server{}

    for rows.Next() {
        s, err := scanServer(rows)
        if err != nil {
            log.Printf("scan server: %v", err)
            respondError(w, 500, "database error")
            return
        }
        servers = append(servers, s)
    }

    if err := rows.Err(); err != nil {
        log.Printf("rows error: %v", err)
        respondError(w, 500, "database error")
        return
    }

    respondJSON(w, 200, servers)
}

func createServer(w http.ResponseWriter, r *http.Request) {
    s, err := decodeServer(r)
    if err != nil {
        respondError(w, 400, err.Error())
        return
    }

    query := `
        INSERT INTO servers (
            hostname, ip_address, os, environment,
            status, cpu, ram, team
        )
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        RETURNING id
    `

    err = db.QueryRowContext(
        r.Context(), query,
        s.Hostname, s.IPAddress, s.OS,
        s.Environment, s.Status, s.CPU, s.RAM, s.Team,
    ).Scan(&s.ID)

    if err != nil {
        if strings.Contains(err.Error(), "duplicate key") {
            respondError(w, 409, "hostname or IP already exists")
            return
        }
        log.Printf("create server: %v", err)
        respondError(w, 500, "database error")
        return
    }

    respondJSON(w, 201, s)
}

func serverHandler(w http.ResponseWriter, r *http.Request) {
    idString := strings.TrimPrefix(r.URL.Path, "/api/servers/")
    id, err := strconv.Atoi(idString)

    if err != nil || id <= 0 {
        respondError(w, 400, "invalid server ID")
        return
    }

    switch r.Method {
    case http.MethodGet:
        getServer(w, r, id)
    case http.MethodPut:
        updateServer(w, r, id)
    case http.MethodDelete:
        deleteServer(w, r, id)
    default:
        w.Header().Set("Allow", "GET, PUT, DELETE")
        respondError(w, 405, "method not allowed")
    }
}

func getServer(w http.ResponseWriter, r *http.Request, id int) {
    query := "SELECT " + serverColumns + " FROM servers WHERE id=$1"

    s, err := scanServer(db.QueryRowContext(r.Context(), query, id))

    if errors.Is(err, sql.ErrNoRows) {
        respondError(w, 404, "server not found")
        return
    }
    if err != nil {
        log.Printf("get server: %v", err)
        respondError(w, 500, "database error")
        return
    }

    respondJSON(w, 200, s)
}

func updateServer(w http.ResponseWriter, r *http.Request, id int) {
    s, err := decodeServer(r)
    if err != nil {
        respondError(w, 400, err.Error())
        return
    }

    query := `
        UPDATE servers SET
            hostname=$1, ip_address=$2, os=$3,
            environment=$4, status=$5,
            cpu=$6, ram=$7, team=$8
        WHERE id=$9
        RETURNING id
    `

    err = db.QueryRowContext(
        r.Context(), query,
        s.Hostname, s.IPAddress, s.OS,
        s.Environment, s.Status, s.CPU, s.RAM, s.Team, id,
    ).Scan(&s.ID)

    if errors.Is(err, sql.ErrNoRows) {
        respondError(w, 404, "server not found")
        return
    }
    if err != nil {
        if strings.Contains(err.Error(), "duplicate key") {
            respondError(w, 409, "hostname or IP already exists")
            return
        }
        log.Printf("update server: %v", err)
        respondError(w, 500, "database error")
        return
    }

    respondJSON(w, 200, s)
}

func deleteServer(w http.ResponseWriter, r *http.Request, id int) {
    result, err := db.ExecContext(
        r.Context(),
        "DELETE FROM servers WHERE id=$1",
        id,
    )

    if err != nil {
        log.Printf("delete server: %v", err)
        respondError(w, 500, "database error")
        return
    }

    count, err := result.RowsAffected()
    if err != nil {
        respondError(w, 500, "database error")
        return
    }

    if count == 0 {
        respondError(w, 404, "server not found")
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func main() {
    host := os.Getenv("POSTGRES_HOST")
    port := os.Getenv("POSTGRES_PORT")
    user := os.Getenv("POSTGRES_USER")
    password := os.Getenv("POSTGRES_PASSWORD")
    database := os.Getenv("POSTGRES_DB")

    dsn := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, database,
    )

    var err error
    db, err = sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)

    for i := 0; i < 30; i++ {
        if err = db.Ping(); err == nil {
            break
        }
        log.Println("Waiting for PostgreSQL...")
        time.Sleep(2 * time.Second)
    }

    if err != nil {
        log.Fatal("Cannot connect to PostgreSQL: ", err)
    }

    mux := http.NewServeMux()

    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            respondError(w, 405, "method not allowed")
            return
        }

        if err := db.PingContext(r.Context()); err != nil {
            respondError(w, 503, "database unavailable")
            return
        }

        respondJSON(w, 200, map[string]string{
            "status": "ok",
        })
    })

    mux.HandleFunc("/api/servers", serversHandler)
    mux.HandleFunc("/api/servers/", serverHandler)

    server := &http.Server{
        Addr:              ":8080",
        Handler:           mux,
        ReadHeaderTimeout: 5 * time.Second,
    }

    log.Println("CMDB API started on :8080")
    log.Fatal(server.ListenAndServe())
}
