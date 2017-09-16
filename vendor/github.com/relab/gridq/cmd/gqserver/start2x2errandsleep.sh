#! /bin/bash

set -e

go build

./gqserver -port=8080 -id=0:0 -sleep -erate=30 &
./gqserver -port=8081 -id=0:1 -sleep -erate=30 &
./gqserver -port=8082 -id=1:0 -sleep -erate=30 &
./gqserver -port=8083 -id=1:1 -sleep -erate=30 &

echo "running, enter to stop"

read && killall gqserver 
