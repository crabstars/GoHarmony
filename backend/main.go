package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type State struct {
	VideoLink    string `json:"video_link"`
	VideoRunning bool   `json:"video_running"`
	// seconds
	VideoTimestamp   int64 `json:"video_timestamp"`
	RequestTimestamp int64 `json:"request_timestamp"`
	Etag             int64 `json:"etag"`
}

type SafeState struct {
	mu    sync.Mutex
	state State
}

var currentState = SafeState{
	state: State{"https://www.youtube.com/watch?v=p7DrHGrpqFU", false, 0, time.Now().Unix(), 0},
}
var (
	clients = make(map[string]chan State)
	mu      sync.Mutex
)

func main() {
	go IncreaseVideoTimestamp()

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Content-Type",
		},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.Logger)

	r.Get("/events/{clientId}", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		clientId := chi.URLParam(r, "clientId")
		mu.Lock()
		if _, exists := clients[clientId]; !exists {
			clients[clientId] = make(chan State)
		}
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		for {
			// waitng for any channel to have information, until then we wait
			select {
			case newState := <-clients[clientId]:
				data, err := json.Marshal(newState)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-r.Context().Done():
				mu.Lock()
				delete(clients, clientId)
				mu.Unlock()
				return
			}
		}
	})

	r.Get("/current-state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		if err := json.NewEncoder(w).Encode(currentState.state); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Patch("/change-state/{clientId}", func(w http.ResponseWriter, r *http.Request) {
		var receivedState State
		if err := json.NewDecoder(r.Body).Decode(&receivedState); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		// clientId := chi.URLParam(r, "clientId")

		currentState.mu.Lock()

		defer currentState.mu.Unlock()
		// check FE state version
		if currentState.state.Etag > receivedState.Etag {
			fmt.Println("ignore old state")
			w.Write([]byte("received old state, no update"))
			return
		}

		if currentState.state.VideoTimestamp != receivedState.VideoTimestamp ||
			currentState.state.VideoLink != receivedState.VideoLink || currentState.state.VideoRunning != receivedState.VideoRunning {

			currentState.state.VideoTimestamp = receivedState.VideoTimestamp
			currentState.state.VideoLink = receivedState.VideoLink
			currentState.state.VideoRunning = receivedState.VideoRunning
			currentState.state.Etag += 10000

			for key := range clients {
				// if clientId == key {
				// 	continue
				// }
				select {
				case clients[key] <- currentState.state:
					fmt.Println("changed state")
				default:
					fmt.Println("no sub")
				}
			}
		}
		w.Write([]byte("received new state"))
	})

	http.ListenAndServe(":3000", r)
}

func IncreaseVideoTimestamp() {
	for {
		if currentState.state.VideoRunning {
			currentState.mu.Lock()
			currentState.state.VideoTimestamp += 1
			currentState.mu.Unlock()
			time.Sleep(1 * time.Second)
		}
	}
}
