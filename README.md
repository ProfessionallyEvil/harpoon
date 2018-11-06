# Harpoon

```
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
```
