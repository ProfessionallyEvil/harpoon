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
  jsonData *interface{}, headers *map[string]string,
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
  paths *map[string]interface{},
  headers *map[string]string,
) (map[string]int ,error) {
  statuses := make(map[string]int)
  for k, _ := range *paths {
    res, err := get(url + k, headers)
    if (err != nil) {
      return nil, err
    }
    statuses[k] = res.StatusCode
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
  flag.Parse()

  headers := map[string]string{
    "Accept": "application/json",
  }
  if (*jwtPtr != "") {
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
      //var out string
      jsonMap := jsonData.(map[string]interface{})
      pathsMap := jsonMap["paths"].(map[string]interface{})
      fmt.Println("[!] API Paths")

      res, err := getPaths(*urlPtr, &pathsMap, &headers)
      panicOnErr(err)
      for k, v := range res {
        if (v != 403) {
          fmt.Println(k + " -> " + strconv.Itoa(v))
        }
      }
    }
  }
}
