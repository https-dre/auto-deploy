#!/bin/bash

set -x

cd $1

git pull

docker stop $2

docker remove $2

docker build -t $2:latest .

docker image prune -f

docker run -d --env-file .env.docker -p $3:$4 --name $2 $2