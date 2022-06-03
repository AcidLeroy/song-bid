package cockroach

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type BidRow struct {
	BidAmount  int
	SongId     string
	BidId      uuid.UUID
	SongStatus int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type PostBidData struct {
	BidAmount int
	SongId    string
}

type Database struct {
	connection *pgx.Conn
	tableName  string
}

func (bid *PostBidData) UnmarshalJSON(b []byte) error {

	// This bit is necessary because you end up with an infinite loop if you don't do this.
	type T2 PostBidData
	var t2 T2

	err := json.Unmarshal(b, &t2)
	if err != nil {
		log.Println("Failed to unmarshal PostBidData JSON string: ", string(b))
		return err
	}
	*bid = PostBidData(t2)
	return nil
}

const databaseName string = "song_bid"

func Connect() *Database {
	// Connect to the "company_db" database.
	connectionString := "postgresql://root@localhost:26257/song_bids?sslmode=disable"

	// Connect to the "song-bid" database
	config, err := pgx.ParseConfig(connectionString)

	if err != nil {
		log.Fatal("error configuring the database: ", err)
	}
	config.Database = databaseName
	conn, err := pgx.ConnectConfig(context.Background(), config)

	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	db := Database{connection: conn, tableName: "tbl_bid"}
	return &db
}

func (db *Database) Close() {
	log.Print("Closing connection")
	defer db.connection.Close(context.Background())
}

func insertRow(ctx context.Context, tx pgx.Tx, data BidRow) error {
	// Insert four rows into the "accounts" table.
	log.Printf("Inserting new row: bidAmount = %d, songId = %s, bidId = %s,  songStatus = %d, createdAt = %s, updatedAt = %s",
		data.BidAmount, data.SongId, data.BidId, data.SongStatus, data.CreatedAt, data.UpdatedAt)
	if _, err := tx.Exec(ctx,
		"INSERT INTO tbl_bid (bid_id, song_id, bid_amount, song_status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		data.BidId, data.SongId, data.BidAmount, data.SongStatus, data.CreatedAt, data.UpdatedAt); err != nil {
		return err
	}
	return nil
}

// PostBid 	creates a new entry in the database for a song that has not yet been played.
func (db *Database) PostBid(data PostBidData) (result *uuid.UUID, err error) {

	bidId := uuid.New()
	row := BidRow{data.BidAmount, data.SongId, bidId, 0, time.Now(), time.Now()}

	err = crdbpgx.ExecuteTx(context.Background(), db.connection, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return insertRow(context.Background(), tx, row)
	})

	if err != nil {
		return nil, err
	}
	return &bidId, nil

}

func (db *Database) GetBids() ([]BidRow, error) {
	rows, err := db.connection.Query(context.Background(), "SELECT * FROM tbl_bid")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var result []BidRow

	for rows.Next() {
		bidRow := BidRow{}

		if err := rows.Scan(&bidRow.BidId, &bidRow.SongId, &bidRow.BidAmount, &bidRow.SongStatus, &bidRow.CreatedAt, &bidRow.UpdatedAt); err != nil {
			log.Fatal(err)
		}
		//log.Println("bid row = ", bidRow)
		result = append(result, bidRow)
	}
	return result, nil
}

func (db *Database) GetHighestBid() (PostBidData, error) {
	// Sum all unplayed bids and get the highest one
	rows, err := db.connection.Query(context.Background(), "select SUM(bid_amount) as bid, song_id from tbl_bid where song_status=0 group by song_id order by bid DESC limit 1")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	bidData := PostBidData{}
	for rows.Next() {
		if err := rows.Scan(&bidData.BidAmount, &bidData.SongId); err != nil {
			log.Fatal(err)
		}
	}
	return bidData, nil
}

// GetBidsGroupBySongId gets all the songs that haven't been played yet, sums their values by songId and returns the result
func (db *Database) GetBidsGroupBySongId() ([]PostBidData, error) {
	// Sum all unplayed bids and get the highest one
	rows, err := db.connection.Query(context.Background(), "select SUM(bid_amount) as bid, song_id from tbl_bid where song_status=0 group by song_id order by bid DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	bidData := PostBidData{}
	results := []PostBidData{}
	for rows.Next() {
		if err := rows.Scan(&bidData.BidAmount, &bidData.SongId); err != nil {
			log.Fatal(err)
		}
		results = append(results, bidData)
	}
	return results, nil
}

// PlayNextSong plays the song that has the aggregate high bid in the queue. It also sets the
// the state of all the songs that represent the highest bid to "song_status = 1".  There should
// only be one song in the bid list that has the status set to 1 because it wouldn't make sense to
// play two songs simultaneously .
//The function  returns all the bids for this particular song. It is sufficient to grab the first song in the list
// to determine what the song id is.
func (db *Database) PlayNextSong() ([]BidRow, error) { // TODO: Currently bused

	// Find the next song, then set all bids for that song to the "Playing", i.e 1
	rows, err := db.connection.Query(context.Background(),
		`UPDATE tbl_bid SET (song_status, updated_at) = (1, $1) FROM (
		SELECT SUM(bid_amount) as bid, song_id from tbl_bid where song_status=0 group by song_id order by bid DESC limit 1) as tmp
		WHERE tbl_bid.song_id = tmp.song_id AND tbl_bid.song_status=0 RETURNING tbl_bid.bid_id, tbl_bid.song_id, tbl_bid.bid_amount, tbl_bid.song_status, tbl_bid.created_at, tbl_bid.updated_at;`, time.Now())

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []BidRow

	for rows.Next() {
		bidRow := BidRow{}

		if err := rows.Scan(&bidRow.BidId, &bidRow.SongId, &bidRow.BidAmount, &bidRow.SongStatus, &bidRow.CreatedAt, &bidRow.UpdatedAt); err != nil {
			log.Fatal(err)
		}
		// log.Println("bid row = ", bidRow)
		result = append(result, bidRow)
	}
	return result, nil
}

func (db *Database) FinalizeCurrentSong() ([]BidRow, error) {
	// Find all songs that have status set to 1, and change it to 2, essentially marking the song as played.
	rows, err := db.connection.Query(context.Background(),
		`UPDATE tbl_bid SET (song_status, updated_at) = (2, $1) FROM (
		SELECT * from tbl_bid where song_status=1 ) as tmp
		WHERE tbl_bid.bid_id = tmp.bid_id AND  RETURNING tbl_bid.bid_id, tbl_bid.song_id, tbl_bid.bid_amount, tbl_bid.song_status, tbl_bid.created_at, tbl_bid.updated_at;`, time.Now())

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []BidRow

	for rows.Next() {
		bidRow := BidRow{}

		if err := rows.Scan(&bidRow.BidId, &bidRow.SongId, &bidRow.BidAmount, &bidRow.SongStatus, &bidRow.CreatedAt, &bidRow.UpdatedAt); err != nil {
			log.Fatal(err)
		}
		// log.Println("bid row = ", bidRow)
		result = append(result, bidRow)
	}
	return result, nil
}

func (db *Database) ClearRows() error {
	log.Println("WARNING: cleared all rows from table.")
	return crdbpgx.ExecuteTx(context.Background(), db.connection, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(context.Background(), "TRUNCATE tbl_bid"); err != nil {
			return err
		}
		return nil
	})
}
