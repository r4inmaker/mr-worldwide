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
var spotifyAccessToken string
var geniusAccessToken string
var geniusID string
var geniusSecret string
var geniusRedirectURL string

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
	geniusID = os.Getenv("GENIUS_CLIENT_ID")
	geniusSecret = os.Getenv("GENIUS_CLIENT_SECRET")
	geniusRedirectURL = os.Getenv("GENIUS_REDIRECT_URL")
}

// SERVER

func main() {
	// Initialize a random seed
	rand.Seed(time.Now().UnixNano())

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/loginSpotify", loginSpotifyHandler)
	mux.HandleFunc("/loginGenius", loginGeniusHandler)
	mux.HandleFunc("/callback", spotifyCallbackHandler)
	mux.HandleFunc("/getPlaylist/{id}", getPlaylistHandler)
	mux.HandleFunc("/getTrack", getTrackHandler)
	mux.HandleFunc("/lyricsCallback", geniusCallbackHandler)

	fmt.Println("Starting server on 3000 ...")
	log.Fatal(http.ListenAndServe(":3000", mux))
}

// ROUTE HANDLERS

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Mista.")
}

func loginSpotifyHandler(w http.ResponseWriter, r *http.Request) {
	loginSpotify(w, r)
}

func loginGeniusHandler(w http.ResponseWriter, r *http.Request) {
	loginGenius(w, r)
}

func spotifyCallbackHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	code := queryParams.Get("code")
	state := queryParams.Get("state")

	// Auth Failure
	if code == "" {
		http.Error(w, "Did not recieve a code from Spotify: "+state, http.StatusBadRequest)
		return
	}

	_accessToken, err := getSpotifyAccessToken(code)
	if err != nil {
		fmt.Printf("Error obtaining Spotify access token: %v", err)
	}

	spotifyAccessToken = _accessToken
	fmt.Fprint(w, "Spotify Access token recieved!")
}

func geniusCallbackHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	code := queryParams.Get("code")

	// Auth Failure
	if code == "" {
		http.Error(w, "did not recieve a code from Genius", http.StatusBadRequest)
		return
	}

	_accesToken, err := getGeniusAccesToken(code)
	if err != nil {
		fmt.Println("Error obtaining Genius access token")
		return
	}

	geniusAccessToken = _accesToken

	fmt.Fprint(w, "Genius Access token recieved")
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

	playlistSlice, err := getPlaylist(spotifyAccessToken, url)
	if err != nil {
		fmt.Fprintf(w, "skill issue: %v", err)
		return
	}

	for _, track := range playlistSlice {
		parts := strings.Split(track, " - ")
		artistName, trackName := parts[0], parts[1]
		fmt.Fprintf(w, "Artist: %v   Track Name: %v\n", artistName, trackName)
	}
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
