package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	addr        = ":8080"
	shortURLLen = 6
	baseURL     = "http://localhost:8080/"
	dbFile      = "urls.db"
)

type requestBody struct {
	URL string `json:"url"`
}

type responseBody struct {
	ShortURL string `json:"short_url"`
}

var (
	db    *sql.DB
	chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}

	createTable := `CREATE TABLE IF NOT EXISTS urls (
		code TEXT PRIMARY KEY,
		original_url TEXT NOT NULL
	);`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

func generateShortCode() string {
	code := make([]rune, shortURLLen)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

func insertURL(code, url string) error {
	_, err := db.Exec("INSERT INTO urls (code, original_url) VALUES (?, ?)", code, url)
	return err
}

func getOriginalURL(code string) (string, error) {
	var url string
	err := db.QueryRow("SELECT original_url FROM urls WHERE code = ?", code).Scan(&url)
	return url, err
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req requestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !strings.HasPrefix(req.URL, "http") {
		http.Error(w, "Invalid request body or URL", http.StatusBadRequest)
		return
	}

	var code string
	for {
		code = generateShortCode()
		err := insertURL(code, req.URL)
		if err == nil {
			break
		}
	}

	resp := responseBody{ShortURL: baseURL + code}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")

	originalURL, err := getOriginalURL(code)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/", redirectHandler)

	fmt.Printf("Server running at %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}