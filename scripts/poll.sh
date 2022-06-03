#!/bin/zsh

accept="Accept: application/json"
contentType="Content-Type: application/json"
auth=""
url="https://api.spotify.com"
query="/v1/search?q=vicente%20amigo&type=artist&limit=1"


result=$(http -b $url$query $accept $contentType $auth)

artistId=$(echo $result | jq -r '.artists .items[] .id  ')

echo "artist ID = $artistId"

query="/v1/artists/$artistId/top-tracks?country=US"
songs=$(http -b $url$query $accept $contentType $auth)
trackIds=$(echo $songs | jq '.tracks[] .uri')
echo $trackIds

