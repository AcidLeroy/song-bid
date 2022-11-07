package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	cr "github.com/acidleroy/song-bid/cockroach"
)

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type apiV1 struct {
	client  HttpClient
	baseUrl string
	timeout time.Duration
}

func NewApi(client HttpClient, baseUrl string, timeout time.Duration) apiV1 {
	return apiV1{client, baseUrl, timeout}

}

// PlayNextSong will fetch a list of bids that represen the next song to play.
// It will return an error if it has any problems fetching the next song.
func (a apiV1) PlayNextSong(ctx context.Context) ([]cr.BidRow, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, a.baseUrl+"player/play", nil)
	if err != nil {
		log.Println("Received an error when getting NextSong: ", err)
		return nil, err
	}

	response, err := a.client.Do(request)
	if err != nil {
		log.Println("Received an error when making the request to PlayNextSong.", err)
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Println("Received a bad status code: ", response.StatusCode)
		return nil, errors.New("Bad status code: " + response.Status)
	}

	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Failed to read body in getNextSong: %s", err)

	}

	bidRows := []cr.BidRow{}

	err = json.Unmarshal(buf, &bidRows)
	if err != nil {
		log.Printf("Failed to Unmarshall Payload in PlayNextSong, %v, buff = %v\n", err, string(buf))
		return nil, err
	}

	if len(bidRows) == 0 {
		log.Printf("There are no more songs in the song queue, nothing to play.\n")
	}

	return bidRows, nil
}

func Finalize() error {
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
