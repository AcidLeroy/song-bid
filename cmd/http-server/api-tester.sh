#!/usr/local/bin/zsh

http :5050

http :5050/api/v1/bids SongId="spotify:track:21GdrXAPYwIZPAFx6JaAxh" BidAmount:=1
http :5050/api/v1/bids
http :5050/api/v1/player/play