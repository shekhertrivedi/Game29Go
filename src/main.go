package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// 1
type longLatStruct struct {
	Long float64 `json:"longitude"`
	Lat  float64 `json:"latitude"`
}

type Player struct {
	Name      string
	BidPoints int
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan *longLatStruct)

var bidding = make(chan map[string]Player)
var biddingInfo = make(map[string]Player)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func init() {
	biddingInfo["A"] = Player{Name: "A"}
	biddingInfo["B"] = Player{Name: "B"}
	biddingInfo["C"] = Player{Name: "C"}
	biddingInfo["D"] = Player{Name: "D"}

}

func main() {
	// 2
	router := mux.NewRouter()
	router.HandleFunc("/", rootHandler).Methods("GET")
	router.HandleFunc("/longlat", longLatHandler).Methods("POST")
	router.HandleFunc("/ws", wsHandler)
	router.HandleFunc("/postBid/{playerID}/{bidPoints}", biddinghandler)
	go echo()

	log.Fatal(http.ListenAndServe(":8844", router))
}

func biddinghandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	playerID := vars["playerId"]
	bidPoints := vars["bidPoints"]
	fmt.Println(playerID, bidPoints)

	player := biddingInfo[playerID]
	bidPt, err := strconv.Atoi(bidPoints)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Invalid bidpoints"))
	}
	player.BidPoints = bidPt
	biddingInfo[playerID] = player
	writerBiddingStatus(biddingInfo)
}

func writerBiddingStatus(biddingiInfo map[string]Player) {
	bidding <- biddingiInfo
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "home")
}

func writer(coord *longLatStruct) {
	broadcast <- coord
}

func longLatHandler(w http.ResponseWriter, r *http.Request) {
	var coordinates longLatStruct
	if err := json.NewDecoder(r.Body).Decode(&coordinates); err != nil {
		log.Printf("ERROR: %s", err)
		http.Error(w, "Bad request", http.StatusTeapot)
		return
	}
	defer r.Body.Close()
	go writer(&coordinates)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	// register client
	clients[ws] = true
}

// 3
func echo() {
	for {

		select {
		case val := <-broadcast:
			{
				latlong := fmt.Sprintf("%f %f %s", val.Lat, val.Long)
				// send to every client that is currently connected
				for client := range clients {
					err := client.WriteMessage(websocket.TextMessage, []byte(latlong))
					if err != nil {
						log.Printf("Websocket error: %s", err)
						client.Close()
						delete(clients, client)
					}
				}
			}
		case val := <-bidding:
			{
				b, err := json.Marshal(val)
				if err != nil {
					fmt.Println("Error occured while marsharling the bidding info")
				}
				for client := range clients {
					err := client.WriteMessage(websocket.TextMessage, b)
					if err != nil {
						log.Printf("Websocket error: %s", err)
						client.Close()
						delete(clients, client)
					}
				}
			}
		}

	}
}
