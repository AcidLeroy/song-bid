package http_server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	cr "github.com/acidleroy/song-bid/cockroach"
)

func getNextSong() ([]cr.BidRow, error) {
	httpServer := "http://localhost:5050/"
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodPut, httpServer+"player/play", nil)
	if err != nil {
		fmt.Println("Received an error when getting NextSong: ", err)
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Received an error when making the request to getNextSong.", err)
		return nil, err
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println("Received a bad status code: ", response.StatusCode)
		fmt.Println("Contents = ", string(contents))
		return nil, errors.New("Bad status code: " + response.Status)
	}

	return nil
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
