package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var clientID string
var clientSecret string
var redirectURL string
var accessToken string
var baseURL string

// ENV VARIABLES

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file %v", err)
	}

	clientID = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
	redirectURL = os.Getenv("REDIRECT_URI")
	baseURL = os.Getenv("BASE_URL")
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
	fmt.Fprintf(w, "Access token recieved: %v,", accessToken)
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
	}

	fmt.Fprintln(w, data)
}

// UTILITY FUNCTIONS

func login(w http.ResponseWriter, r *http.Request) {
	authURL := "https://accounts.spotify.com/authorize"
	redirectURL := "http://localhost:3000/callback"
	scope := "user-read-private user-read-email"
	state := generateRandomString(16)

	urlObj, err := url.Parse(authURL)
	if err != nil {
		http.Error(w, "Error generating URL", http.StatusInternalServerError)
		return
	}

	queryParams := url.Values{}
	queryParams.Set("response_type", "code")
	queryParams.Set("client_id", clientID)
	queryParams.Set("scope", scope)
	queryParams.Set("redirect_uri", redirectURL)
	queryParams.Set("state", state)

	urlObj.RawQuery = queryParams.Encode()
	http.Redirect(w, r, urlObj.String(), http.StatusFound)
}

func getAccessToken(code string) (string, error) {
	// Exchange code for auth token
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

	data := url.Values{}
	data.Set("code", code)
	data.Set("redirect_uri", redirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to make request for auth token: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+auth)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send the request for auth token %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body of response %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK status: %s, response: %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %v", err)
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access token not found in response")
	}

	return accessToken, nil
}

type Playlist struct {
	Tracks struct {
		Items []struct {
			Track struct {
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"track"`
		} `json:"items"`
	} `json:"tracks"`
}

func getPlaylist(_accessToken string, url string) (string, error) {

	// send a request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create a request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+_accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send the request: %v", err)
	}

	defer resp.Body.Close()

	// read body in bytes
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// parse json
	var playlist Playlist
	err = json.Unmarshal(bodyBytes, &playlist)
	if err != nil {
		return "", fmt.Errorf("failed to parse json: %v", err)
	}

	for _, item := range playlist.Tracks.Items {
		_artists := ""
		_trackName := "- " + item.Track.Name
		for _, artist := range item.Track.Artists {
			_artist := artist.Name + " "
			_artists += _artist
		}
		fmt.Println(_artists + _trackName)
	}

	return "Success!", nil
}

func print(args ...interface{}) {
	fmt.Println(args...)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}

	return result
}
