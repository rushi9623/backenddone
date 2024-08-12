package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var ctx = context.Background()
var rdb *redis.Client

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var user map[string]string
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	username := user["username"]
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	err := rdb.HSet(ctx, "users", username, 0).Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User registered successfully"))
}

func loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var user map[string]string
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	username := user["username"]
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	points, err := rdb.HGet(ctx, "users", username).Result()
	if err == redis.Nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Login successful, points: " + points))
}

func startGameHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Generate a deck of 5 cards
	deck := []string{"Cat", "Defuse", "Shuffle", "ExplodingKitten", "Cat"}
	// Shuffle the deck (simple example)
	// You might want to implement a proper shuffle algorithm

	err := rdb.HSet(ctx, "games", username, strings.Join(deck, ",")).Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Game started successfully"))
}

func drawCardHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	deckString, err := rdb.HGet(ctx, "games", username).Result()
	if err == redis.Nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deck := strings.Split(deckString, ",")
	if len(deck) == 0 {
		http.Error(w, "No cards left", http.StatusBadRequest)
		return
	}

	card := deck[0]
	deck = deck[1:]
	err = rdb.HSet(ctx, "games", username, strings.Join(deck, ",")).Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Card drawn: " + card))
}

func getLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	leaderboard, err := rdb.HGetAll(ctx, "users").Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON
	response, err := json.Marshal(leaderboard)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func updateLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	var user map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	username, ok := user["username"].(string)
	if !ok || username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	pointsStr, ok := user["points"].(string)
	if !ok {
		http.Error(w, "Points must be an integer", http.StatusBadRequest)
		return
	}

	points, err := strconv.Atoi(pointsStr)
	if err != nil {
		http.Error(w, "Invalid points format", http.StatusBadRequest)
		return
	}

	err = rdb.HSet(ctx, "users", username, points).Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Leaderboard updated successfully"))
}

func main() {
	router := mux.NewRouter()

	// Basic route to check server status
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Server is running"))
	})

	// Define routes
	router.HandleFunc("/register", registerUserHandler).Methods("POST")
	router.HandleFunc("/login", loginUserHandler).Methods("POST")
	router.HandleFunc("/start", startGameHandler).Methods("POST")
	router.HandleFunc("/draw", drawCardHandler).Methods("GET")
	router.HandleFunc("/leaderboard", getLeaderboardHandler).Methods("GET")
	router.HandleFunc("/updateLeaderboard", updateLeaderboardHandler).Methods("POST")

	// Start the server
	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}
