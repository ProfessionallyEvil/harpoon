package main

import (
  //"bytes"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "net/http"
  "flag"
  "crypto/tls"
  "strconv"
  "strings"
)

func get(url string, headers *map[string]string) (*http.Response, error) {
  client := &http.Client{}
  req, err := http.NewRequest("GET", url, nil)
  if (err != nil) {
    return nil, err
  }

  for k, v := range *headers {
    req.Header.Add(k, v)
  }

  res, err := client.Do(req)
  if (err != nil) {
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
  if (err != nil) {
    return nil, err
  }

  for k, v := range *headers {
    req.Header.Add(k, v)
  }
  res, err := client.Do(req)
  if (err != nil) {
    return nil, err
  }
  // make sure the body ReaderCloser gets closed once the func exits
  defer res.Body.Close()

  body, err := ioutil.ReadAll(res.Body)
  if (err != nil) {
    return nil, err
  }
  json.Unmarshal(body, &jsonData)
  return res, err
}

func panicOnErr(err error) {
  if (err != nil) {
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
    if (err != nil) {
      return nil, err
    }
    statuses[(*paths)[p]] = res.StatusCode
  }
  return statuses, nil
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
  //dumpPtr := flag.Bool("dump", false, "Dump all the information possible")
  flag.Parse()

  headers := map[string]string{
    "Accept": "application/json",
  }
  if (*jwtPtr != "") {
    *jwtPtr = strings.ReplaceAll(*jwtPtr, "\"", "")
    //fmt.Println("Bearer " + *jwtPtr)
    headers["Authorization"] = "Bearer " + *jwtPtr
  }

  var jsonData interface{}
  res, err := getJson(*urlPtr + "/version", &jsonData, &headers)
  panicOnErr(err)
  // TODO: pretty print the version info
  if (res.StatusCode == 200) {
    var out string
    jsonMap := jsonData.(map[string]interface{})
    fmtJsonMap(jsonMap, &out)
    fmt.Println("[!] k8s version info:")
    fmt.Println(out)
  }

  parsedPaths := false
  var pathsEnumRes map[string]int
  res, err = getJson(*urlPtr + "/swagger.json", &jsonData, &headers)
  panicOnErr(err)
  if (res.StatusCode == 200) {
    var out string
    jsonMap := jsonData.(map[string]interface{})
    fmtJsonMap(jsonMap, &out)
    fmt.Println("[+] got /swagger.json... attempting to enumerate access")
  } else {
    res, err = getJson(*urlPtr + "/openapi/v2", &jsonData, &headers)
    panicOnErr(err)
    if (res.StatusCode == 200) {
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
      for k, _ := range pathsMap {
        // replace {namespace} with -ns value
        // TODO retool this to go into a slice!
        pathsSlice = append(pathsSlice, strings.Replace(k, "{namespace}", *nsPtr, 1))
      }
      res, err := getPaths(*urlPtr, &pathsSlice, &headers)
      panicOnErr(err)
      for k, v := range res {
        if (v != 403) {
          fmt.Println(k + " -> " + strconv.Itoa(v))
        }
      }
    }
  }

  if (*dumpPtr == true) {
    /* Try to dump the followoing:
     * - /api/v1/namespaces/{namespace}/pods
     * - /api/v1/namespaces/{namespace}/serviceaccounts
     * - /api/v1/namespaces/{namespace}/secrets
     * - /api/v1/namespaces/{namespace}/roles
    */
    base := "/api/v1/namespaces/" + *nsPtr
    paths := [
      base + "/pods",
      base + "/serviceaccounts",
      base + "/secrets",
      base + "/roles"
    ]

    for p := range paths {
      var jsonData interface{}
      res, err := getJson(*urlPtr,
  }
}
