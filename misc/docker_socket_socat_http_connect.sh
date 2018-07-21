#!/bin/bash

# POST /containers/$DOCKER_CONTAINER_ID/attach?stream=1&stdin=1&stdout=1&stderr=1 HTTP/1.1
# Host:
# Connection: Upgrade
# Upgrade: tcp
# EOM

# How to use:
# this is one way that we can establish a shell with a container over HTTP
# simply replace $DOCKER_SOCK_PATH with the path to the docker socket (see fingerprinting/find_docker_sock.sh)
socat - UNIX-CONNECT:$DOCKER_SOCK_PATH
