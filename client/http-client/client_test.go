package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	cr "github.com/acidleroy/song-bid/cockroach"
	"github.com/google/uuid"
)

type Any interface{}

type HttpClientMock struct {
	DoFunc func(*http.Request) (*http.Response, error)
}

func (h HttpClientMock) Do(r *http.Request) (*http.Response, error) {
	return h.DoFunc(r)
}

func generateString(t *testing.T, data Any) string {
	d, err := json.Marshal(data)
	if err != nil {
		t.Fatal("Failed to marshal testing data.", data)
	}
	return string(d)
}

func TestPlayNextSong(t *testing.T) {
	id, _ := uuid.NewUUID()

	validBidRow := []cr.BidRow{{BidAmount: 1, SongId: "song-id", BidId: id, SongStatus: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	testTable := []struct {
		MockBody       string
		MockStatusCode int

		ExpectedResult []cr.BidRow
		ExpectedError  error
	}{
		{
			MockBody:       generateString(t, validBidRow),
			MockStatusCode: 200,

			ExpectedResult: validBidRow,
			ExpectedError:  nil,
		},
	}

	mockClient := &HttpClientMock{}
	api := NewApi(mockClient, "http://some-fake-website.com", 0)

	for _, test := range testTable {
		// Setup what the mock function does for each test.
		mockClient.DoFunc = func(r *http.Request) (*http.Response, error) {
			fmt.Println("Mockbody is ", test.MockBody)
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(test.MockBody)),
				StatusCode: test.MockStatusCode,
			}, nil
		}

		ctx := context.Background()
		p, err := api.PlayNextSong(ctx)
		if err != test.ExpectedError {
			t.Fatalf("Expected the error %v, but instead received %v.\n", test.ExpectedError, err)
		}

		if len(p) != len(test.ExpectedResult) {
			t.Fatalf("Expected result to be length %v, but instead received %v.\n", len(test.ExpectedResult), len(p))
		}

	}
}
