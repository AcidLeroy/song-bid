package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/acidleroy/song-bid/cockroach"
)

const prefix string = "/api/v1"

type apiHandler struct {
	mux      *http.ServeMux
	database *cockroach.Database
}

func NewApiHandler() *apiHandler {
	return &apiHandler{mux: http.NewServeMux(), database: cockroach.Connect()}

}

func (p *apiHandler) HandleGetBids(w http.ResponseWriter, r *http.Request) {
	log.Println("Get all active bids")
	bids, err := p.database.GetBids()
	if err != nil {
		log.Printf("Failed to get bids: %s", err)
		fmt.Fprint(w, "Internal Server error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	value, err := json.Marshal(bids)
	if err != nil {
		log.Printf("Failed to marshal bids: %v", err)
		fmt.Fprint(w, "Internal Server error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", string(value))
}

func (p *apiHandler) HandlePostBid(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		fmt.Fprintf(w, `Invalid JSON request, expecting: {"BidAmount": int, "SongId": string}`)
		w.WriteHeader(http.StatusBadRequest)
	}

	buf, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("Failed to read body: %s", err)
		fmt.Fprint(w, "Internal Server error")
		w.WriteHeader(http.StatusInternalServerError)
	}

	bid := cockroach.PostBidData{}
	bid.UnmarshalJSON(buf)

	uuid, err := p.database.PostBid(bid)
	if err != nil {
		fmt.Fprintf(w, "There was an error posting the bid.")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		fmt.Fprintf(w, "The uuid is : %s", uuid)
	}

}

func (p *apiHandler) HandleBids(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle the bid!")
	switch r.Method {
	case http.MethodGet:
		p.HandleGetBids(w, r)
	case http.MethodPost:
		p.HandlePostBid(w, r)
	default:
		fmt.Fprintf(w, "Handle other methods\n")
	}
}

func (p *apiHandler) HandlePlayerPlay(w http.ResponseWriter, r *http.Request) {
	log.Println("player/play")

	switch r.Method {
	case http.MethodPut:
		if next, err := p.database.PlayNextSong(); err != nil {
			log.Printf("Error playing next song: %v\n", err)
			fmt.Fprintf(w, "Internal server error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			if len(next) > 0 {
				log.Printf("Playing next song: %v", next[0])
				if result, err := json.Marshal(next); err != nil {
					log.Printf("There was an issue marshalling bids: %v\n", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				} else {
					log.Printf("Returning %d bids to the user.\n", len(next))
					fmt.Fprintf(w, "%s", result)
					return
				}

			} else {
				log.Println("There are no songs to play!")
				fmt.Fprintf(w, "{}")
				return
			}

		}
	default:
		log.Printf("Method %v is not supported by '/play' \n", r.Method)
		http.NotFound(w, r)
		return
	}
}

func (p *apiHandler) HandlePlayerFinalize(w http.ResponseWriter, r *http.Request) {
	log.Println("player/finalize")

	switch r.Method {
	case http.MethodPut:
		if next, err := p.database.FinalizeCurrentSong(); err != nil {
			log.Printf("Error finalizing current song: %v\n", err)
			fmt.Fprintf(w, "Internal server error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			if len(next) > 0 {
				log.Printf("Finalizing this song: %v", next[0])
				if result, err := json.Marshal(next); err != nil {
					log.Printf("There was an issue marshalling bids: %v\n", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				} else {
					log.Printf("Finalized these songs: %s\n", result)
					fmt.Fprintf(w, "%s", result)
					return
				}

			} else {
				log.Println("No songs currently playing")
				fmt.Fprintf(w, "{}")
				return
			}

		}
	default:
		log.Printf("Method %v is not supported by '/player/finalize' \n", r.Method)
		http.NotFound(w, r)
		return
	}
}

func main() {

	api := NewApiHandler()

	defer api.database.Close()

	api.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			log.Printf("Unhandled path: %v\n", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, "Welcome to song-bid v1.0!\n")
	})

	api.mux.HandleFunc(prefix+"/bids", api.HandleBids)
	api.mux.HandleFunc(prefix+"/player/play", api.HandlePlayerPlay)
	api.mux.HandleFunc(prefix+"/player/finalize", api.HandlePlayerFinalize)

	// listen to port
	fmt.Println("Starting the server on 5050.")
	if err := http.ListenAndServe(":5050", api.mux); err != nil {
		log.Printf("Could not start the server, got an error: %v\n", err)
	}
}
