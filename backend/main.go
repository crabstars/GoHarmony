package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type State struct {
	VideoId      string `json:"video_id"`
	VideoRunning bool   `json:"video_running"`
	// seconds
	VideoTimestamp   int64 `json:"video_timestamp"`
	RequestTimestamp int64 `json:"request_timestamp"`
}

var currentState = State{"FnLvyysSCw4", false, 0, time.Now().Unix()}

func main() {
	go IncreaseVideoTimestamp()
	clients := map[string]chan State{}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
		},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.Logger)

	r.Get("/events/{clientId}", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher) // TODO check what is http.Pusher
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		clientId := chi.URLParam(r, "clientId")
		if _, exists := clients[clientId]; !exists {
			clients[clientId] = make(chan State)
		}

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
				}

				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-r.Context().Done():
				// TODO mutex
				delete(clients, clientId)
				return
			}
		}
	})

	r.Get("/current-state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		if err := json.NewEncoder(w).Encode(currentState); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Patch("/change-state/{clientId}", func(w http.ResponseWriter, r *http.Request) {
		var receivedState State
		if err := json.NewDecoder(r.Body).Decode(&receivedState); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		clientId := chi.URLParam(r, "clientId")

		// TODO mutex => only one can change the state at a time
		// TODO combine to one function
		if receivedState.VideoRunning != currentState.VideoRunning {
			currentState.VideoRunning = receivedState.VideoRunning
			newState := currentState
			for key := range clients {
				if clientId == key {
					continue
				}
				select {
				case clients[key] <- newState:
					fmt.Println("sent state for videoRunning")
				default:
					fmt.Println("no sub")
				}
			}

		}

		if receivedState.VideoTimestamp != currentState.VideoTimestamp {
			currentState.VideoTimestamp = receivedState.VideoTimestamp
			newState := currentState
			for key := range clients {
				if clientId == key {
					continue
				}
				select {
				case clients[key] <- newState:
					fmt.Println("sent state for videoTime")
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
		// TODO mutex
		if currentState.VideoRunning {
			currentState.VideoTimestamp += 1
			time.Sleep(1 * time.Second)
		}
	}
}

/*

DB

do we need a database at the start??

No:
	1. idea server tracks user and sends them a message(user and server)
	- intern state which has:
		- a list of users
			- user has a timestamp for video, if state timestamp is different by x time the user timestamp needs to change
		- current video
		- bool if video is playing
		- timestamp from the video
			-> if video is playing the timestamp should increase each second
	- if someone is out of sync or we change timestamp can wait a short time to sync them again


	2. idea user makes all requests
	- can send stop/start to server
	- set videotimestamp and send to server
	- set video
	- get current state from server => videoId, timestamp from video, playing(bool), unix timestamp when the videotimestamp was written in the message
	- FE has to calculate the correct timestamp and change local timestamp if it is more than x seconds difference
	- server has to increase videotimestamp periodcly



- Channel Tabel => for the start we only have one channel or none?


*/
