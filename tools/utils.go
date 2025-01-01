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

type Results struct {
	Data []struct {
		TrackID int    `json:"id"`
		Title   string `json:"title"`
		Preview string `json:"preview"`
		Artist  struct {
			ArtistName string `json:"name"`
			ArtistId   int    `json:"id"`
			Picture    string `json:"picture_medium"`
		} `json:"artist"`
	} `json:"data"`
}

type TrackInfo struct {
	TrackID        int
	ArtistID       int
	ArtistName     string
	Title          string
	PreviewLink    string
	PreviewPicture string
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
		_trackData := _artists + _trackName
		playlistSlice = append(playlistSlice, _trackData)
	}

	return "Success!", nil
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
		TrackID:        topResult.TrackID,
		ArtistID:       topResult.Artist.ArtistId,
		ArtistName:     topResult.Artist.ArtistName,
		Title:          topResult.Title,
		PreviewLink:    topResult.Preview,
		PreviewPicture: topResult.Artist.Picture,
	}, nil
}

func createFolder(trackID string) error {
	return nil
}
