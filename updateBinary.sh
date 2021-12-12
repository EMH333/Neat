#!/bin/bash
cd /var/www/services/neat.ethohampton.com || exit
#pull and build
git pull
go build main.go
#now restart the neat service
sudo systemctl restart EMH-Neat
