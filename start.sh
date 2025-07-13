#!/bin/bash

# Increase system limits
echo "Increasing system limits..."
ulimit -n 1000000
sysctl -w net.ipv4.ip_local_port_range="1024 65535"
sysctl -w net.ipv4.tcp_tw_reuse=1

# Build and run
echo "Building Go binary..."
go build -o ddos-tool main.go

if [ $? -ne 0 ]; then
    echo "Build failed! Exiting."
    exit 1
fi

echo "Starting attack on $1"
./ddos-tool "$1"
