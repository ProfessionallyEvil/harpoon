package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type dockerInfo struct {
	docker          bool
	dockerSockPath  []string
	dockerApiVer    string
	dockerVer       string
	dockerUserGroup bool
}

func (di dockerInfo) toString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\thas_docker: %v\n", di.docker))
	sb.WriteString(fmt.Sprintf("\tsocket_paths: %v\n", di.dockerSockPath))
	sb.WriteString(fmt.Sprintf("\tdocker_ver_info: %v\n", di.dockerApiVer))
	sb.WriteString(fmt.Sprintf("\tin_docker_group: %v\n", di.dockerUserGroup))
	return sb.String()
}

type kubernetesInfo struct {
}

func execCmd(cmdStr string) (bytes.Buffer, bytes.Buffer) {
	cmdArr := strings.Split(cmdStr, " ")
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(cmdArr[0], cmdArr[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Run()
	return stdout, stderr
}

func findDockerSocket() []string {
	stdout, _ := execCmd("find / -name docker.sock")
	socks := make([]string, 1)
	for _, e := range strings.Split(stdout.String(), "\n") {
		if e != "" {
			socks = append(socks, e)
		}
	}
	return socks
}

func isDockerUser() bool {
	stdout, _ := execCmd("groups")
	return strings.Contains(stdout.String(), "docker")
}

func findDockerApiVer() string {
	// curl -s -XGET --unix-socket /run/docker.sock http://localhost/version 2>&1
	verInfo, _ := execCmd("docker version")
	return verInfo.String()
}

func main() {
	var help string = `Harpoon

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
Usage: harpoon [options]

Options:
	Todo: write this help...
`

	// set up command line opts
	findCommand := flag.NewFlagSet("fingerprint", flag.ExitOnError)
	dockerInfo := dockerInfo{}
	//kubernetesInfo := kubernetesInfo{}

	if len(os.Args) == 1 {
		fmt.Println(help)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "fingerprint":
		findCommand.Parse(os.Args[2:])
		socks := findDockerSocket()
		if socks != nil {
			dockerInfo.docker = true
			dockerInfo.dockerSockPath = socks
		}
		dockerInfo.dockerUserGroup = isDockerUser()
		r := strings.NewReplacer("\n", "\n\t\t")
		dockerInfo.dockerApiVer = fmt.Sprintf("\n\t\t%s\n", r.Replace(findDockerApiVer()))
	}

	// display found info
	fmt.Printf("[+] Docker info found:\n")
	fmt.Println(dockerInfo.toString())
}
