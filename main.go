package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"
)

type MessageChannel struct {
	Notifier chan []byte
	Clients  map[chan []byte]bool
}

var messageChannel MessageChannel

func formatSSE(event string, data string) string {
	eventPayload := "event: " + event + "\n"
	dataLines := strings.Split(data, "\n")
	for _, line := range dataLines {
		eventPayload = eventPayload + "data: " + line + "\n"
	}
	return eventPayload + "\n"
}

func broadcaster(done <-chan interface{}) {
	fmt.Println("Broadcaster Started.")
	for {
		select {
		case <-done:
			return
		case data := <-messageChannel.Notifier:
			for channel := range messageChannel.Clients {
				channel <- data
			}
		}
	}
}

func main() {

	messageChannel = MessageChannel{
		Clients:  make(map[chan []byte]bool),
		Notifier: make(chan []byte),
	}

	done := make(chan interface{})
	defer close(done)

	go broadcaster(done)

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/sse", sseHandler)
	http.HandleFunc("/send", sendMessageHandler)

	log.Printf("Starting web server at localhost:3000")
	log.Fatal("HTTP server error: ", http.ListenAndServe(":3000", nil))

}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	var tmpl, err = template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var data map[string]interface{}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	username := r.URL.Query().Get("username")
	if len(username) < 3 {
		http.Error(w, "Invalid User", http.StatusUnauthorized)
		return
	}

	log.Printf("NEW client connection: %s", username)

	msgChan := make(chan []byte)
	messageChannel.Clients[msgChan] = true

	// close the channel after exit the function
	defer func() {
		close(msgChan)
		log.Printf("CLOSED client connection: %s", username)
	}()

	// trap the request under loop forever
	for {

		select {
		case messageData := <-msgChan:
			log.Printf("SENT Message to: %s", username)
			fmt.Fprintf(w, formatSSE("message", string(messageData)))
			flusher.Flush()

		// connection is closed then defer will be executed
		case <-r.Context().Done():
			delete(messageChannel.Clients, msgChan)
			return
		}
	}
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	message := r.FormValue("message")

	if len(username) < 3 {
		http.Error(w, "Invalid User", http.StatusUnauthorized)
		return
	}
	if len(message) < 1 {
		http.Error(w, "Invalid Message", http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"username":   username,
		"event_type": "NEW_CHAT_MESSAGE",
		"message":    message,
		"timestamp":  time.Now().Unix(),
	})

	if err != nil {
		http.Error(w, "Ups", http.StatusInternalServerError)
		return
	}

	messageChannel.Notifier <- jsonData
	log.Printf("NEW Message from: %s", username)

	w.Write([]byte("ok."))
}
