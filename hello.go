package main

import (
	"bytes"
	"encoding/json"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// structs/handlers for serperate test servers
type fooHandler struct{}

func (m *fooHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Listening on 1331: foo "))
}

type barHandler struct{}

func (m *barHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Listening on 1332: bar "))
}

type bazHandler struct{}

func (m *bazHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Listening on 1333: baz "))
}

// Get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Get listen port
func getListenAddress() string {
	port := getEnv("PORT", "1338")
	return ":" + port
}

// Log the env variables
func logSetup() {
	a_condtion_url := os.Getenv("A_CONDITION_URL")
	b_condtion_url := os.Getenv("B_CONDITION_URL")
	default_condtion_url := os.Getenv("DEFAULT_CONDITION_URL")

	log.Printf("Server running on: %s\n", getListenAddress())
	log.Printf("Redirect to A condition url: %s\n", a_condtion_url)
	log.Printf("Redirect to B condition url: %s\n", b_condtion_url)
	log.Printf("Redirect to Default condition url: %s\n", default_condtion_url)
}

type requestPayloadStruct struct {
	ProxyCondition string `json:"proxy_condition"`
}

// Get json decoder for a requests body
func requestBodyDecoder(request *http.Request) *json.Decoder {
	// Read body to buffer
	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		panic(err)
	}

	// Because golang is a pain, if you read the body then any susequent calls
	// are unable to read the body again....
	request.Body = io.NopCloser(bytes.NewBuffer(body))

	return json.NewDecoder(io.NopCloser(bytes.NewBuffer(body)))
}

// Parse the requests body
func parseRequestBody(request *http.Request) requestPayloadStruct {
	decoder := requestBodyDecoder(request)

	var requestPayload requestPayloadStruct
	err := decoder.Decode(&requestPayload)

	if err != nil {
		panic(err)
	}

	return requestPayload
}

// Log the typeform payload and redirect url
func logRequestPayload(requestionPayload requestPayloadStruct, proxyUrl string) {
	log.Printf("proxy_condition: %s, proxy_url: %s\n", requestionPayload.ProxyCondition, proxyUrl)
}

// Get the url for a given proxy condition
func getProxyUrl(proxyConditionRaw string) string {
	proxyCondition := strings.ToUpper(proxyConditionRaw)

	a_condtion_url := os.Getenv("A_CONDITION_URL")
	b_condtion_url := os.Getenv("B_CONDITION_URL")
	default_condtion_url := os.Getenv("DEFAULT_CONDITION_URL")

	if proxyCondition == "A" {
		return a_condtion_url
	}

	if proxyCondition == "B" {
		return b_condtion_url
	}

	return default_condtion_url
}

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	requestPayload := parseRequestBody(req)
	url := getProxyUrl(requestPayload.ProxyCondition)
	logRequestPayload(requestPayload, url)

	serveReverseProxy(url, res, req)
}

func main() {

	// load env vars
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}
	// spin up isolated http servers for demo
	go func() {
		http.ListenAndServe(":1331", &fooHandler{})
	}()

	go func() {
		http.ListenAndServe(":1332", &barHandler{})
	}()

	go func() {
		http.ListenAndServe(":1333", &bazHandler{})
	}()

	// Log setup values
	logSetup()

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}
