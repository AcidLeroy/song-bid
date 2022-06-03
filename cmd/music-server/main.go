package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var spotifyUrl = "https://accounts.spotify.com"
var credentialsFile = ".spotify-token.json"

func main() {
	auth, err := fetchToken()

	if err != nil {
		return
	}

	fmt.Printf("Successfully received token:\n\n%v\n\n", (*auth).String())
	nextTrack, err := getNextTrack()
	if err != nil {
		fmt.Println("Received an error when attempting to play next track: ", err)
	}
	fmt.Printf("The next track is: %+v\n", nextTrack)
	err = PlaySong(auth, nextTrack, "")
	if err != nil {
		fmt.Println("Failed to play next song because: ", err)
	}

	return
}

type Authorization struct {
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	ExpiresIn   *int       `json:"expires_in,omitempty"`
	Timestamp   *time.Time `json:"timestamp,omitempty"`
}

func (a *Authorization) String() string {
	result := ""
	result += "AccessToken: " + a.AccessToken + ",\n" + "TokenType: " + a.TokenType
	if a.ExpiresIn != nil {
		result += ",\nExpiresIn: " + strconv.Itoa(*a.ExpiresIn)
	}
	if a.Timestamp != nil {
		result += ",\nTimestamp: " + (*a.Timestamp).Format(time.UnixDate)
	}
	return result
}

func PlaySong(auth *Authorization, songId, deviceId string) error {
	payload := make(map[string]string)
	payload["context_uri"] = songId
	bytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Received an error when atteping to marshal JSON in PlaySong: ", err)
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(bytes)
	req, err := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/play", strings.NewReader(encoded))
	if err != nil {
		fmt.Println("Error creating PUT request for /me/player/play: ", err)
		return err
	}

	authString := auth.TokenType + " " + auth.AccessToken

	req.Header.Set("Authorization", authString)
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: time.Duration(30 * time.Second)}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Received an error when sending request to play song: ", err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Got an error reading the body: %v\n", err)
		return err
	}
	fmt.Println("Body = ", string(body))
	return nil

}

func getTokenFromFile(filename string) (*Authorization, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Could not read token file %v: %v\n", filename, err)
		return nil, err
	}
	result := &Authorization{}
	err = json.Unmarshal(buf, result)
	if err != nil {
		fmt.Printf("Error unmarshalling spotify token from file, %v: %v\n", filename, err)
		return nil, err
	}

	if result.ExpiresIn == nil {
		fmt.Println("There is no expiration on token.")
		return result, nil
	}

	if result.Timestamp == nil {
		fmt.Println("There is no timestamp, need to fetch a new token.")
		return nil, errors.New("timestamp missing from authorization, need to regenerate a new one.")
	}

	if time.Duration(time.Since(*result.Timestamp).Seconds()) > time.Duration((*result.ExpiresIn)*time.Now().Second()) {
		fmt.Println("Token is expired, need to fetch a new one")
		return nil, errors.New("Spotify token expired")
	}

	return result, nil
}

func writeTokenFile(filename string, token *Authorization) error {
	bytes, err := json.Marshal(token)
	if err != nil {
		fmt.Printf("Could not marshal JSON token: %v\n", err)
		return err
	}
	err = ioutil.WriteFile(filename, bytes, 0644)
	if err != nil {
		fmt.Printf("Could not write file '%v': %v\n", filename, err)
		return err
	}

	return err
}

// fetchToken retrieves the client_credentials to make requests to the Spotify API.
// It returns a map of values where the token can be extracted. See https://developer.spotify.com/documentation/general/guides/authorization/client-credentials/
// for workflow details.
func fetchToken() (*Authorization, error) {
	fmt.Println("fetching file...")
	fmt.Println("Attempting to read credentials from file...")
	cachedResult, err := getTokenFromFile(credentialsFile)

	if err == nil {
		fmt.Println("Successfully pulled credentials from file.")
		return cachedResult, nil
	}

	formData := url.Values{
		"grant_type": {"client_credentials"},
	}

	req, err := http.NewRequest("POST", spotifyUrl+"/api/token", strings.NewReader(formData.Encode()))

	if err != nil {
		fmt.Printf("Got an error making the request to %v: %v", spotifyUrl, err)
		return nil, err
	}
	authString := clientId + ":" + clientSecret

	fmt.Printf("Auth String: '%v'\n", authString)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(authString)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: time.Duration(30 * time.Second)}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("Error making request to spotify: %v\n", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Got an error reading the body: %v\n", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Received an error with status code %v\n", resp.StatusCode)
		fmt.Printf("Error message: %v", string(body))
		return nil, err
	}

	result := &Authorization{}
	_ = json.Unmarshal(body, &result)
	// append the current time to token as well
	result.Timestamp = new(time.Time)
	*result.Timestamp = time.Now()

	err = writeTokenFile(credentialsFile, result)
	if err != nil {
		fmt.Println("Could not write Tokenfile, skipping")
	}

	return result, nil
}

func getNextTrack() (string, error) {
	return "spotify:track:6ADzlFXHPk846zUCEOM2C1", nil
}
