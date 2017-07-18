package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Request struct {
	Sites      []string `json:"sites" binding:"required"`
	SearchText string   `json:"search_text" binding:"required"`
}

type Response struct {
	FoundAtSite string `json:"FoundAtSite"`
}

func getConfigSettings(name string) interface{} {
	// Here we load config and return specific keys
	switch name {
	case "HTTP_TIMEOUT":
		to := os.Getenv("HTTP_TIMEOUT")
		to_int, err := strconv.ParseInt(to, 10, 64)
		if err != nil {
			// if we got an error set timeout to default
			return time.Duration(75) * time.Second
		}
		return time.Duration(to_int) * time.Second
	}
	return nil
}

func main() {
	router := GetEngine()
	router.Run(":8080")
}

func GetEngine() *gin.Engine {
	router := gin.Default()
	router.POST("/checkText", checkHandler)
	router.GET("/checkHealth", HealthCheckHandler)
	return router
}

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

func getContents(addr string) (body string, err error) {
	httpClient := &http.Client{Timeout: getConfigSettings("HTTP_TIMEOUT").(time.Duration)}
	resp, err := httpClient.Get(addr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("Error getting %v -- %v", addr, resp))
	}
	bodySlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	body = string(bodySlice)
	return body, nil
}
