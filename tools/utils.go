package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
)

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		result += string(charset[rand.Intn(len(charset))])
	}

	return result
}

func loginSpotify(w http.ResponseWriter, r *http.Request) {
	authURL := "https://accounts.spotify.com/authorize"
	redirectURL := "http://localhost:3000/callback"
	scope := "user-read-private user-read-email"
	state := generateRandomString(16)

	urlObj, err := url.Parse(authURL)
	if err != nil {
		http.Error(w, "error generating URL", http.StatusInternalServerError)
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

func loginGenius(w http.ResponseWriter, r *http.Request) {
	authURL := "https://api.genius.com/oauth/authorize"
	scope := "me"
	state := generateRandomString(16)
	response_type := "code"

	urlObj, err := url.Parse(authURL)
	if err != nil {
		http.Error(w, "error Generating URL", http.StatusInternalServerError)
		return
	}

	queryParams := url.Values{}
	queryParams.Set("client_id", geniusID)
	queryParams.Set("redirect_uri", geniusRedirectURL)
	queryParams.Set("scope", scope)
	queryParams.Set("state", state)
	queryParams.Set("response_type", response_type)

	urlObj.RawQuery = queryParams.Encode()
	http.Redirect(w, r, urlObj.String(), http.StatusFound)
}

func getSpotifyAccessToken(code string) (string, error) {
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

type AccessTokenRequest struct {
	Code         string `json:"code"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	ResponseType string `json:"response_type"`
	GrantType    string `json:"grant_type"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func getGeniusAccesToken(code string) (string, error) {
	url := "https://api.genius.com/oauth/token"

	payload := AccessTokenRequest{
		Code:         code,
		ClientID:     geniusID,
		ClientSecret: geniusSecret,
		RedirectURI:  geniusRedirectURL,
		ResponseType: "code",
		GrantType:    "authorization_code",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("genius API error: %s, response: %s", string(bodyBytes), resp.Status)
	}

	var accessTokenResponse AccessTokenResponse
	err = json.Unmarshal(bodyBytes, &accessTokenResponse)
	if err != nil {
		return "", err
	}

	return accessTokenResponse.AccessToken, nil
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

type Results struct {
	Data []struct {
		TrackID int    `json:"id"`
		Title   string `json:"title"`
		Preview string `json:"preview"`
		Artist  struct {
			ArtistName    string `json:"name"`
			ArtistId      int    `json:"id"`
			ArtistPicture string `json:"picture_medium"`
		} `json:"artist"`
		Album struct {
			AlbumPicture string `json:"cover_medium"`
		} `json:"album"`
	} `json:"data"`
}

type TrackInfo struct {
	TrackID       int
	ArtistID      int
	ArtistName    string
	Title         string
	PreviewLink   string
	ArtistPicture string
	AlbumPicture  string
}

func getPlaylist(_accessToken string, url string) ([]string, error) {

	// send a request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{}, fmt.Errorf("failed to create a request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+_accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []string{}, fmt.Errorf("failed to send the request: %v", err)
	}

	defer resp.Body.Close()

	// read body in bytes
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, fmt.Errorf("failed to read response body: %v", err)
	}

	// parse json
	var playlist Playlist
	err = json.Unmarshal(bodyBytes, &playlist)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse json: %v", err)
	}

	var playlistSlice []string

	for _, item := range playlist.Tracks.Items {
		_artists := ""
		_trackName := "- " + item.Track.Name
		for _, artist := range item.Track.Artists {
			_artist := artist.Name + " "
			_artists += _artist
		}
		_trackData := _artists + _trackName
		playlistSlice = append(playlistSlice, _trackData)
	}

	return playlistSlice, nil
}

func getTrack(searchQuery string) (TrackInfo, error) {
	baseUrl := "https://api.deezer.com/search"

	u, err := url.Parse(baseUrl)
	if err != nil {
		fmt.Printf("error parsing URL: %v", err)
		return TrackInfo{}, err
	}

	query := u.Query()
	query.Set("q", searchQuery)
	u.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Printf("error creating request: %v", err)
		return TrackInfo{}, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error sending request: %v", err)
		return TrackInfo{}, err
	}

	defer resp.Body.Close()

	bytesData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error reading response: %v", err)
		return TrackInfo{}, err
	}

	var results Results
	err = json.Unmarshal(bytesData, &results)
	if err != nil {
		fmt.Printf("failed to parse json: %v", err)
		return TrackInfo{}, err
	}

	topResult := results.Data[0]

	return TrackInfo{
		TrackID:       topResult.TrackID,
		ArtistID:      topResult.Artist.ArtistId,
		ArtistName:    topResult.Artist.ArtistName,
		Title:         topResult.Title,
		PreviewLink:   topResult.Preview,
		ArtistPicture: topResult.Artist.ArtistPicture,
		AlbumPicture:  topResult.Album.AlbumPicture,
	}, nil
}

// TODO

// Function that fetches lyrics
// Figure out logic for optimal storage of items
//   > no duplicates in images (use IDs)
//   > quick fetching (use deezer ID to name folders)
// Function that stores track data and lyrics
// Function that reads that data

// Storage structure
//
// . /img
//       /artist > 74309.jpg
//       /album  > 6425418.jpg
//
// ./ track_data > 65546431.txt > track_data + lyrics
