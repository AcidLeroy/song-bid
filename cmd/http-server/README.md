# Starting the http-server

1. Start the cockroach database: 
    `cockroach start-single-node --insecure --http-port=26256 --host=localhost`

2. Initialize the database
    `cockroach sql --insecure --file song_bid/cockroach/init_database.sql`

3. Start the server
    `go run main.go`

4. Inspect the database
    `cockroach sql --insecure` --> `\c song_bid` --> `select * from tbl_bid;`