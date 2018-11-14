package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

type dockerInfo struct {
	hasDocker       bool
	dockerSockPath  []string
	dockerApiVer    string
	dockerVer       string
	dockerUserGroup bool
}

func (di dockerInfo) querySocket(socketPath string, method string, path string, payload string) string {
	var message string = method + " " + path + " HTTP/1.1\n\r" + "Host:\n\r" + "\n\r"
	conn, _ := net.Dial("unix", socketPath)
	defer conn.Close()
	_, _ = conn.Write([]byte(message))
	// send message
	_ = conn.(*net.UnixConn).CloseWrite()
	data := make([]byte, 0)
	for {
		buf := make([]byte, 512)
		nr, _ := conn.Read(buf)
		buf = buf[:nr]
		data = append(data, buf...)
	}
	return string(data)
}

func (di dockerInfo) toString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\thas_docker: %v\n", di.hasDocker))
	sb.WriteString(fmt.Sprintf("\tsocket_paths: %v\n", di.dockerSockPath))
	sb.WriteString(fmt.Sprintf("\tdocker_ver_info: %v\n", di.dockerApiVer))
	sb.WriteString(fmt.Sprintf("\tin_docker_group: %v\n", di.dockerUserGroup))
	return sb.String()
}

func (di dockerInfo) findDockerSocket() []string {
	stdout, _ := execCmd("find / -name docker.sock")
	socks := make([]string, 1)
	for _, e := range strings.Split(stdout.String(), "\n") {
		if e != "" {
			socks = append(socks, e)
		}
	}
	return socks
}

func (di dockerInfo) isDockerUser() bool {
	stdout, _ := execCmd("groups")
	return strings.Contains(stdout.String(), "docker")
}

func (di dockerInfo) findDockerApiVer() string {
	// curl -s -XGET --unix-socket /run/docker.sock http://localhost/version 2>&1
	var verInfo bytes.Buffer
	if di.hasDocker {
		verInfo, _ = execCmd("docker version")
	} else if di.dockerSockPath != nil {
		if len(di.dockerSockPath) >= 1 {
			// just take the first value
			var dockerSocket string = di.dockerSockPath[0]
			di.querySocket(dockerSocket, "GET", "http://127.0.0.1/version", "")
		}
	}
	return verInfo.String()
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
		socks := dockerInfo.findDockerSocket()
		if socks != nil {
			dockerInfo.hasDocker = true
			dockerInfo.dockerSockPath = socks
		}
		dockerInfo.dockerUserGroup = dockerInfo.isDockerUser()
		r := strings.NewReplacer("\n", "\n\t\t")
		dockerInfo.dockerApiVer = fmt.Sprintf("\n\t\t%s\n", r.Replace(dockerInfo.findDockerApiVer()))
	}

	// display found info
	fmt.Printf("[+] Docker info found:\n")
	fmt.Println(dockerInfo.toString())
	fmt.Println(dockerInfo.querySocket("/var/run/docker.sock", "GET", "http://127.0.0.1/version", ""))
}
