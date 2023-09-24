package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type State struct {
	VideoId          string         `json:"video_id"`
	VideoRunning     bool           `json:"video_running"`
	VideoTimestamp   VideoTimestamp `json:"video_timestamp"`
	RequestTimestamp int64          `json:"request_timestamp"`
}

type VideoTimestamp struct {
	Hour   uint8 `json:"hour"`
	Minute uint8 `json:"minute"`
	Second uint8 `json:"second"`
}

func main() {
	currentState := State{"FnLvyysSCw4", false, VideoTimestamp{0, 0, 0}, time.Now().Unix()}
	var clientEvents []chan State
	// TODO mabye update to make(map[chan string]bool) because what if a client dc?
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher) // TODO check what is http.Pusher
		eventChannel := make(chan State)
		clientEvents = append(clientEvents, eventChannel)

		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		defer func() {
			for i, channel := range clientEvents {
				if channel == eventChannel {
					clientEvents = append(clientEvents[:i], clientEvents[i+1:]...)
				}
			}
		}()

		for {
			newState := <-eventChannel
			if err := json.NewEncoder(w).Encode(newState); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			flusher.Flush()
		}
	})

	r.Get("/current-state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		if err := json.NewEncoder(w).Encode(currentState); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Patch("/change-state", func(w http.ResponseWriter, r *http.Request) {
		var receivedState State
		if err := json.NewDecoder(r.Body).Decode(&currentState); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		// TODO mutex => only one can change the state at a time
		if receivedState.VideoRunning != currentState.VideoRunning {
			currentState.VideoRunning = receivedState.VideoRunning
			newState := currentState
			// events <- newState // drop message if we have no subscribed clients => we need to allow multiple client right now only one can sub
			for _, client := range clientEvents {
				select {
				case client <- newState:
					fmt.Println("sent state")
				default:
					fmt.Println("no sub")
				}
			}

		}
		w.Write([]byte("received new state"))
	})

	http.ListenAndServe(":3000", r)
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
