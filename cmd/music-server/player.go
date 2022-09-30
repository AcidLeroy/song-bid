// This example demonstrates how to authenticate with Spotify.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//       - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"

	"github.com/zmb3/spotify/v2"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "http://localhost:8080/callback"

var (
	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadCurrentlyPlaying, spotifyauth.ScopeUserReadPlaybackState, spotifyauth.ScopeUserModifyPlaybackState),
	)
	ch        = make(chan *spotify.Client)
	state     = "abc123"
	tokenFile = ".spotify-token.json"
)

func getTokenFromFile(filename string) (*oauth2.Token, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Could not read token file %v: %v\n", filename, err)
		return nil, err
	}
	result := &oauth2.Token{}
	err = json.Unmarshal(buf, result)
	if err != nil {
		fmt.Printf("Error unmarshalling spotify token from file, %v: %v\n", filename, err)
		return nil, err
	}
	return result, nil
}

func finalize() error {
	httpServer := "http://localhost:5050/"
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodPut, httpServer+"player/finalize", nil)
	if err != nil {
		fmt.Println("Received an error when createing new request to finalize any songs: ", err)
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Received an error when making the request to finalize.", err)
		return err
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println("Received a bad status code: ", response.StatusCode)
		fmt.Println("Contents = ", string(contents))
		return errors.New("Bad status code: " + response.Status)
	}
	return nil
}

func getNextSong() error {
	httpServer := "http://localhost:5050/"
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodPut, httpServer+"player/finalize", nil)
	if err != nil {
		fmt.Println("Received an error when createing new request to finalize any songs: ", err)
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Received an error when making the request to finalize.", err)
		return err
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println("Received a bad status code: ", response.StatusCode)
		fmt.Println("Contents = ", string(contents))
		return errors.New("Bad status code: " + response.Status)
	}
	return nil
}

func writeTokenToFile(filename string, tok *oauth2.Token) error {

	bytes, err := json.Marshal(tok)
	if err != nil {
		fmt.Println("Could not Marshal Oauth2 token: ", err)
		return err
	}

	if err := ioutil.WriteFile(filename, bytes, 0644); err != nil {
		fmt.Printf("Could not write file %v because %v\n", filename, err)
		return err
	}

	return nil
}

func verifyToken(tok *oauth2.Token) bool {
	now := time.Now().Unix()
	if now >= tok.Expiry.Unix() {
		fmt.Println("Token has expired")
		return false
	} else {
		fmt.Println("Token is still valid")
		return true
	}
}

func trackGenerator() func() string {
	tracks := []string{
		"spotify:track:6ADzlFXHPk846zUCEOM2C1",
		"spotify:track:1eVnOimXaPos2ua7Rxb7vY",
		"spotify:track:0nOm9qJ8lChfshtWsMNGX5",
		"spotify:track:21GdrXAPYwIZPAFx6JaAxh",
		"spotify:track:35KJGai6SDpUR65mMJ6lkP",
		"spotify:track:5lpOIzbl08NopnkBYMC2cq",
		"spotify:track:6z4EdxNQRQbah5yMpz2CSL",
		"spotify:track:6LnE7XmNCJrRkUufVwJyLE",
		"spotify:track:7oBAd2YeVYN8i0ShTppfRC",
		"spotify:track:0nHV2PFo3cSocA0Bk1ebIH",
	}
	currentTrack := 0
	nextTrack := func() string {
		track := tracks[currentTrack%len(tracks)]
		currentTrack++
		return track
	}
	return nextTrack

}

func main() {
	// We'll want these variables sooner rather than later
	var client *spotify.Client
	var playerState *spotify.PlayerState

	http.HandleFunc("/callback", completeAuth)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})

	initiateAuth := func() {

		// See if we have a saved token file
		tok, err := getTokenFromFile(tokenFile)
		if err != nil {
			fmt.Println("Could not read token file, must have user authorize app...")
			url := auth.AuthURL(state)
			fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
			// wait for auth to complete
			client = <-ch
			// write the token to the file for next time.
			tok, _ = client.Token()
			writeTokenToFile(tokenFile, tok)
		} else {
			fmt.Println("Successfully loaded token from file.")
			client = spotify.New(auth.Client(context.Background(), tok))
		}

		// For now, we just print what the status is. Ideally, we would refresh the tokens if needed.
		verifyToken(tok)

		// use the client to make calls that require authorization
		user, err := client.CurrentUser(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("You are logged in as:", user.ID)

		playerState, err = client.PlayerState(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found your %s (%s)\n", playerState.Device.Type, playerState.Device.Name)
		nextSong := trackGenerator()
		for {
			playerState, err := client.PlayerState(context.Background())
			if err != nil {
				log.Fatal(err)
			}
			isPlaying := playerState.CurrentlyPlaying.Playing
			if !isPlaying {
				fmt.Println("No song is playing, getting next track from bids database")
				song := spotify.URI(nextSong())
				uris := []spotify.URI{song}
				opts := spotify.PlayOptions{URIs: uris}
				err = client.PlayOpt(context.Background(), &opts)
				if err != nil {
					fmt.Println("There was an error playing the next song: ", err)
				}
			}
			time.Sleep(time.Second * 1)
		}
	}

	go initiateAuth()

	http.ListenAndServe(":8080", nil)

}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	fmt.Println("Token = ", tok)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}
