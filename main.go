package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"regexp"
	"flag"
	"time"
)

type Config struct {
	CheckIPUrl string `json:"url_check_ip"`
	UpdateIPUrl string `json:"url_update_ip"`
	Username string `json:"user"`
	Password string `json:"pass"`
	Hostname string `json:"hostname"`
}

// Report error and crash
func reportErr(e error) {
	fmt.Fprintln(os.Stderr, "Error: ", e)
	os.Exit(1)
}


// Given a path to a config file in json,
// try to parse it into Config type with
// corresponding required values
func getConfig(fpath string) (Config, error) {

	var config Config

	if _, err := os.Stat(fpath); err != nil {
		return config, err
	}

	fconfig, err := os.Open(fpath)
	if err != nil {
		return config, err
	}

	parser := json.NewDecoder(fconfig)
	if err = parser.Decode(&config); err != nil {
		return config, err
	}

	return config, nil
}


// Fetch current IP from specific host
func fetchIP(c Config) (string, error) {
	var ip string
	var err error
	var response *http.Response

	if response, err = http.Get(c.CheckIPUrl); err != nil {
		return ip, err
	}

	// close resposne body
	defer response.Body.Close()

	// read content
	if body, err := ioutil.ReadAll(response.Body); err != nil {
		return ip, fmt.Errorf("fetchIP: Body parsing error %v", err.Error())
	} else {
		ip = string(body)
	}

	// validate and check ip for proper formatting
	ip = strings.TrimSpace(string(ip))
	if ip == "" {
		return ip, errors.New("fetchIP: Request did not return proper IP\n")
	}
	// currently, only expect IPv4
	if valid, _ := regexp.MatchString("^(\\d{1,3}\\.?){3}\\d{1,3}$", ip); !valid {
		return ip, errors.New(fmt.Sprintf("fetchIP: Invalid or unknown IP format: %s\n", ip))
	}

	return ip, nil

}

func updateIP(c Config, currentIP string) error {
	var request *http.Request
	var response *http.Response
	var err error

	if request, err = http.NewRequest("POST", c.UpdateIPUrl, nil); err != nil {
		return fmt.Errorf("updateIP: Failed to generate request with error %v", err.Error())
	}

	query := request.URL.Query()
	// add credentials
	query.Add("hostname", c.Hostname)
	query.Add("myip", currentIP)
	query.Add("user", c.Username)
	query.Add("pass", c.Password)
	// append query string
	request.URL.RawQuery = query.Encode()

	// prepare client with timeout
	timeout := time.Duration(10 * time.Second)
	client := http.Client{ Timeout: timeout }

	// execute
	if response, err = client.Do(request); err != nil {
		return fmt.Errorf("updateIP: Failed to update IP with request error %v", err.Error())
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("updateIP: Failed parsing body with error %v", err.Error())
	}

	defer response.Body.Close()
	
	if response.StatusCode > 204 {
		return fmt.Errorf("updateIP: Updated failed status code %v body %v", response.StatusCode, string(body))
	}

	return nil
}

func main() {
	defaultConfigFile := "./config.json"
	configPath := flag.String("c", defaultConfigFile, "A configuration file in json format")
	flag.Parse()

	// check if we have a configPath
	configFile := strings.TrimSpace(*configPath)
	if configFile == defaultConfigFile {
		fmt.Printf("No config path given, using default: %s\n", defaultConfigFile)
	}

	// placeholder parameters
	var config Config
	var err error
	var currentIP string
	// read configuration
	if config, err = getConfig(configFile); err != nil {
		reportErr(err)
	}

	// fetch current ip
	if currentIP, err = fetchIP(config); err != nil {
		reportErr(err)
	}

	// update ip with provider
	if err = updateIP(config, currentIP); err != nil {
		reportErr(err)
	}
	fmt.Println("IP Updated Successfully")
}