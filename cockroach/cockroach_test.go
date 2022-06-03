package cockroach

import (
	"encoding/json"
	"fmt"
	"testing"
)

//cockroach sql --insecure --host=localhost:26257
func TestConnection(t *testing.T) {
	t.Log("Testing for valid connection")
	db := Connect()
	defer db.Close()
}

func TestPostBid(t *testing.T) {
	db := Connect()

	defer db.Close()
	defer db.ClearRows()

	bidData := PostBidData{BidAmount: 1, SongId: "some-song-id"}
	result, err := db.PostBid(bidData)

	if err != nil {
		t.Log("Received an error: ", err)
		t.FailNow()
	}

	if result == nil {
		t.Log("Result should have had an ID")
		t.FailNow()
	}
}

func TestPostBidDataUnmarshalJson(t *testing.T) {
	t.Log("Testing for valid unmarshalling of JSON for PostBidData")
	var validJson = []byte(`{"BidAmount": 1, "SongId": "some-id"}`)

	postBidData := PostBidData{}

	err := postBidData.UnmarshalJSON(validJson)
	if err != nil {
		t.Logf("Failed to Unmarshal JSON string, %s with error: %s", string(validJson), err)
		t.FailNow()
	}
}

func TestPostBidDataUnmarshalBadJson(t *testing.T) {
	t.Log("Testing for valid unmarshalling of bad JSON for PostBidData")
	var invalidJson = []byte(`{"BidAmount": 1, "SongId": "some-id"`)
	postBidData := PostBidData{}

	err := postBidData.UnmarshalJSON(invalidJson)
	if err == nil {
		t.Logf("Failed to report an error for bad JSON string: %s", string(invalidJson))
		t.FailNow()
	}
}

func TestGetBids(t *testing.T) {
	t.Log("Testing GetBids")
	db := Connect()
	defer db.Close()

	bids, err := db.GetBids()
	if err != nil {
		t.Logf("Failed to get bids: %s", err)
		t.FailNow()
	}

	for _, bid := range bids {
		jsonF, err := json.Marshal(bid)
		if err != nil {
			t.Logf("failed to Marshal bid: %v", err)
			t.FailNow()
		}
		t.Log(string(jsonF))
	}

}

func TestGetBidsGroupBySongId(t *testing.T) {
	t.Log("Testing getting the next song to play")
	db := Connect()
	defer db.Close()
	defer db.ClearRows()

	bids := []PostBidData{
		{BidAmount: 2, SongId: "song-a"},
		{BidAmount: 2, SongId: "song-a"},
		{BidAmount: 5, SongId: "song-b"},
		{BidAmount: 5, SongId: "song-b"},
	}

	for _, bid := range bids {
		_, err := db.PostBid(bid)
		if err != nil {
			t.Logf("Failed to post bid: %v", err)
			t.FailNow()
		}
	}

	results, err := db.GetBidsGroupBySongId()
	if err != nil {
		t.Logf("Received an error from GetBidsGroupBySongId, %v", err)
		t.FailNow()
	}

	if len(results) != 2 {
		t.Logf("Expected the lenght of results to be 2, but instead got %v", len(results))
		t.FailNow()
	}

	if results[0].BidAmount != 10 || results[0].SongId != "song-b" {
		t.Logf("Expected first BidAmount to be 10, received %v, and SongId to be song-b, received %v.\n", results[0].BidAmount, results[0].SongId)
		t.FailNow()
	}

	if results[1].BidAmount != 4 || results[1].SongId != "song-a" {
		t.Logf("Expected secound BidAmount to be 10, received %v, and SongId to be song-b, received %v.\n", results[1].BidAmount, results[1].SongId)
		t.FailNow()
	}

}

func TestGetNextSong(t *testing.T) {

	t.Log("Testing getting the next song to play")
	db := Connect()
	defer db.Close()
	defer db.ClearRows()

	bids := []PostBidData{
		{BidAmount: 2, SongId: "song-a"},
		{BidAmount: 2, SongId: "song-a"},
		{BidAmount: 5, SongId: "song-b"},
		{BidAmount: 5, SongId: "song-b"},
	}

	for _, bid := range bids {
		_, err := db.PostBid(bid)
		if err != nil {
			t.Logf("Failed to post bid: %v", err)
			t.FailNow()
		}
	}
	//select SUM(bid_amount), song_id from tbl_bid where song_status=0 group by song_id;
	//select SUM(bid_amount) as bid, song_id from tbl_bid where song_status=0 group by song_id order by bid DESC limit 1;
	bid, err := db.GetHighestBid()
	if err != nil {
		t.Logf("Got an error when calling GetHighestBid: %v", err)
		t.FailNow()
	}

	if bid.SongId != "song-b" {
		t.Logf("Expected song-b, but got %v", bid.SongId)
		t.FailNow()
	}

	if bid.BidAmount != bids[2].BidAmount*2 {
		t.Logf("Expected the bid amount to be the sum of all the songs curently not played. Instead we got %d.", bid.BidAmount)
		t.FailNow()
	}
}

func TestGetNextSongEmpty(t *testing.T) {
	t.Log("Testing getting the next song to play when there is no song")
	db := Connect()
	defer db.Close()
	defer db.ClearRows()

	bid, err := db.GetHighestBid()
	if err != nil {
		t.Logf("Got an error when calling GetHighestBid: %v", err)
		t.FailNow()
	}

	if bid.BidAmount != 0 || bid.SongId != "" {
		t.Logf("Expected to get an empty bid, but instead got bidAmount = %v and SongId = %v\n", bid.BidAmount, bid.SongId)
		t.FailNow()
	}
}

func TestPlayNextSong(t *testing.T) {
	t.Log("Testing PlayNextSong")
	db := Connect()
	defer db.Close()
	defer db.ClearRows()

	bids := []PostBidData{
		{BidAmount: 2, SongId: "song-a"},
		{BidAmount: 2, SongId: "song-a"},
		{BidAmount: 5, SongId: "song-b"},
		{BidAmount: 5, SongId: "song-b"},
	}

	for _, bid := range bids {
		_, err := db.PostBid(bid)
		if err != nil {
			t.Logf("Failed to post bid: %v", err)
			t.FailNow()
		}
	}

	if rows, err := db.PlayNextSong(); err != nil {
		t.Logf("Received an error when attempting to play next song %v", err)
		t.FailNow()
	} else {
		fmt.Println("rows length = ", len(rows))
		for _, row := range rows {
			if row.SongStatus != 1 {
				t.Logf("Expected song status to be 1, instead received: %d\n", row.SongStatus)
				t.FailNow()
			}

			if row.SongId != "song-b" {
				t.Logf("Expected the song id to be \"song-b\", instead received: \"%v\"", row.SongId)
				t.FailNow()
			}
		}
	}
}

func TestPlayNextSongEmpytList(t *testing.T) {
	t.Log("Testing PlayNextSong")
	db := Connect()
	defer db.Close()
	defer db.ClearRows()

	if rows, err := db.PlayNextSong(); err != nil {
		t.Logf("Received an error when attempting to play next song %v", err)
		t.FailNow()
	} else {
		if len(rows) != 0 {
			t.Logf("Should have received no rows, instead received %v.\n", len(rows))
			t.FailNow()
		}
	}
}

func playNextSongHelper(t *testing.T, db *Database) {
	if _, err := db.PlayNextSong(); err != nil {
		t.Logf("Received an error when attempting to play next song %v", err)
		t.FailNow()
	}
}

func TestFinalizePlayingSongs(t *testing.T) {
	t.Log("Testing PlayNextSong")
	db := Connect()
	defer db.Close()
	defer db.ClearRows()
	bids := []PostBidData{
		{BidAmount: 2, SongId: "song-a"},
		{BidAmount: 2, SongId: "song-a"},
		{BidAmount: 5, SongId: "song-b"},
		{BidAmount: 5, SongId: "song-b"},
	}

	for _, bid := range bids {
		_, err := db.PostBid(bid)
		if err != nil {
			t.Logf("Failed to post bid: %v", err)
			t.FailNow()
		}
	}

	if _, err := db.PlayNextSong(); err != nil {
		t.Logf("Received an error when attempting to play next song %v", err)
		t.FailNow()
	}

	rows, err := db.FinalizeCurrentSong()
	if err != nil {
		t.Logf("Could not finalize current song, got an error: %v\n", err)
		t.FailNow()
	}

	fmt.Println("Number of rows: ", len(rows))
	if len(rows) != 2 {
		t.Logf("Expected to have only updated 2 rows, but instead updated %v.\n", len(rows))
		t.FailNow()
	}

	for _, row := range rows {
		if row.SongStatus != 2 {
			t.Logf("Expected song status to be 2, instead got %v.\n", row.SongStatus)
		}
	}

}
