CREATE DATABASE "song_bid"; 
SET DATABASE = "song_bid"; 

CREATE TABLE "tbl_bid" (
    "bid_id" UUID PRIMARY KEY,
    "song_id" STRING(100), 
    "bid_amount" INT, 
    "song_status" INT, 
    "created_at" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);