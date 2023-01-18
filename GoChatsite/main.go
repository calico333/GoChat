package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type ChatMessage struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

var (
	rdb *redis.Client
)

var clients = make(map[*websocket.Conn]bool)
var broadcaster = make(chan ChatMessage)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// If a message is sent while a client is closing, ignore the error
func unsafeError(err error) bool {
	return !websocket.IsCloseError(err, websocket.CloseGoingAway) && err != io.EOF
}

func pathToFile(filename string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	dir := path.Dir(currentFile)
	return path.Join(dir, filename)
}

func RedirectToHTTPSRouter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		proto := req.Header.Get("x-forwarded-proto")
		if proto == "http" || proto == "HTTP" {
			http.Redirect(res, req, fmt.Sprintf("https://%s%s", req.Host, req.URL), http.StatusPermanentRedirect)
			return
		}

		next.ServeHTTP(res, req)

	})
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	websock, err := upgrader.Upgrade(w, r, nil)
	check(err)

	defer websock.Close()
	clients[websock] = true

	if rdb.Exists(rdb.Context(), "chat_messages").Val() != 0 {
		sendExistingMessages(websock)
	}

	for {
		var msg ChatMessage
		// Read in a new message as JSON and map it to a Message object
		err := websock.ReadJSON(&msg)
		if err != nil {
			delete(clients, websock)

			break
		}
		// send new message to the channel
		broadcaster <- msg
	}
}

func storeInRedis(msg ChatMessage) {
	json, err := json.Marshal(msg)
	check(err)

	err = rdb.RPush(rdb.Context(), "chat_messages", json).Err()
	check(err)
}

func handleMessages() {
	for {
		// grab any next message from channel
		msg := <-broadcaster

		storeInRedis(msg)

		messageClients(msg)
	}
}

func sendExistingMessages(websock *websocket.Conn) {
	chatMessages, err := rdb.LRange(rdb.Context(), "chat_messages", 0, -1).Result()
	check(err)

	// send previous messages

	for _, chatMessage := range chatMessages {
		var msg ChatMessage
		json.Unmarshal([]byte(chatMessage), &msg)
		messageClient(websock, msg)
	}
}

func messageClients(msg ChatMessage) {
	for client := range clients {
		messageClient(client, msg)
	}
}

func messageClient(client *websocket.Conn, msg ChatMessage) {
	err := client.WriteJSON(msg)

	if err != nil && unsafeError(err) {
		log.Printf("error: %v", err)
		client.Close()
		delete(clients, client)
	}
}

func main() {
	//router := mux.NewRouter()
	//httpsRouter := RedirectToHTTPSRouter(router)

	err := godotenv.Load()
	check(err)

	port := os.Getenv("PORT")

	//redisURL := os.Getenv("REDIS_URL")

	opt, err := redis.ParseURL("redis://default:indy2016@localhost:6379/0")
	check(err)

	rdb = redis.NewClient(opt)

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/websocket", handleConnections)
	go handleMessages()

	log.Print("Server started at localhost on port 8080.")

	err = http.ListenAndServeTLS(":"+port, pathToFile("/Certs/server.crt"), pathToFile("/Certs/server.key"), nil)
	check(err)

	//http.ListenAndServeTLS(":8080", pathToFile("server.crt"), pathToFile("server.key"), httpsRouter)

	//http.ListenAndServe(":8080", httpsRouter)
}
