package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var clientID string
var clientSecret string
var redirectURL string
var tempStorePath string
var accessToken string
var playlistSlice []string

// ENV VARIABLES

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file %v", err)
	}

	clientID = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
	redirectURL = os.Getenv("REDIRECT_URI")
	tempStorePath = os.Getenv("TEMP_STORE_PATH")
}

// SERVER

func main() {
	// Initialize a random seed
	rand.Seed(time.Now().UnixNano())

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/callback", callbackHandler)
	mux.HandleFunc("/getPlaylist/{id}", getPlaylistHandler)
	mux.HandleFunc("/getTrack", getTrackHandler)

	print("Starting server on 3000 ...")
	log.Fatal(http.ListenAndServe(":3000", mux))
}

// ROUTE HANDLERS

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Mista.")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	login(w, r)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	code := queryParams.Get("code")
	state := queryParams.Get("state")

	// Auth Failure
	if code == "" {
		http.Error(w, "Did not recieve a code: "+state, http.StatusBadRequest)
		return
	}

	_accessToken, err := getAccessToken(code)
	if err != nil {
		fmt.Printf("Error obtaining an acess token: %v", err)
	}

	accessToken = _accessToken
	fmt.Fprint(w, "Access token recieved!")
}

func getPlaylistHandler(w http.ResponseWriter, r *http.Request) {

	//get playlist ID
	base_path := "/getPlaylist/"
	playlistID := strings.TrimPrefix(r.URL.Path, base_path)

	if playlistID == "" {
		http.Error(w, "You need to provide a playlist ID", http.StatusBadRequest)
		return
	}

	url := "https://api.spotify.com/v1/playlists/" + playlistID

	data, err := getPlaylist(accessToken, url)
	if err != nil {
		fmt.Fprintf(w, "skill issue: %v", err)
		return
	}

	fmt.Fprintln(w, data)
}

func getTrackHandler(w http.ResponseWriter, r *http.Request) {

	//get track query
	queryParams := r.URL.Query()
	searchQuery := queryParams.Get("search")

	track, err := getTrack(searchQuery)
	if err != nil {
		fmt.Printf("error fetching resource: %v", err)
		return
	}

	fmt.Println(track)
}
