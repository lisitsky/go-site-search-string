package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Request structure
// Sites - slice of string of sites to visit
// SearchText - text to search on these sites
type Request struct {
	Sites      []string `json:"sites" binding:"required"`
	SearchText string   `json:"search_text" binding:"required"`
}

// Response structure
// FoundAtSite - URL of a site  where SearchText was found
type Response struct {
	FoundAtSite string `json:"FoundAtSite"`
}

type Config struct {
	Http struct {
		Timeout time.Duration
		Listen  string
	}
}

func GetConfig() *Config {
	var c = &Config{}
	httpTimeoutStr := os.Getenv("HTTP_TIMEOUT")
	httpTimeoutInt, err := strconv.ParseInt(httpTimeoutStr, 10, 64)
	if err == nil {
		c.Http.Timeout = time.Duration(httpTimeoutInt) * time.Second
	}

	var httpListen string
	httpListen = os.Getenv("HTTP_LISTEN")
	if httpListen == "" {
		httpListen = ":8080"
	}
	c.Http.Listen = httpListen

	return c
}

func main() {
	var config = GetConfig()
	router := GetEngine()
	router.Run(config.Http.Listen)
}

// GetEngine returns gin.Engine instance with handlers
func GetEngine() *gin.Engine {
	router := gin.Default()
	router.POST("/checkText", checkHandler)
	router.GET("/checkHealth", HealthCheckHandler)
	return router
}

// HealthCheckHandler - handler to be used to check service health
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func checkHandler(c *gin.Context) {
	var payload Request
	if err := c.BindJSON(&payload); err != nil {
		// cannot unpack data to structure
		c.JSON(http.StatusBadRequest, gin.H{"status": "bad request"})
		return
	}
	result, err := checkSites(payload.Sites, payload.SearchText)
	if err != nil || result == "" {
		c.JSON(http.StatusNoContent, gin.H{})
		return
	}
	c.JSON(http.StatusOK, gin.H{"FoundAtSite": result})
}

func checkSites(sites []string, searchText string) (string, error) {
	foundChan := make(chan string)
	allDone := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(len(sites))
	for _, site := range sites {
		// Start checkers
		go func(addr string) {
			defer wg.Done()
			if res, err := checkSite(addr, searchText); err == nil && res {
				foundChan <- addr
			}
		}(site)
	}

	// Wait for all tasks to be done
	go func() {
		wg.Wait()
		allDone <- struct{}{}
	}()

	select {
	case result := <-foundChan:
		return result, nil
	case <-allDone:
		return "", nil
	}
}

// Searches for `text` on site with `addr` address and
// Good old-fashioned synchronous function )
func checkSite(addr string, text string) (found bool, err error) {
	body, err := getContents(addr)
	if err != nil {
		return false, err
	}
	return strings.Contains(body, text), nil
}

// Read body if there's an error reading body, return "", err
func getBodyString(respBody io.Reader) (string, error) {
	bodySlice, err := ioutil.ReadAll(respBody)
	if err != nil {
		return "", err
	}
	return string(bodySlice), nil
}

func getContents(addr string) (body string, err error) {
	var config = GetConfig()
	httpClient := &http.Client{Timeout: config.Http.Timeout}
	resp, err := httpClient.Get(addr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error getting %v -- %v", addr, resp)
	}
	// In case unreadable body just treat it as empty
	body, _ = getBodyString(resp.Body)
	return body, nil
}
