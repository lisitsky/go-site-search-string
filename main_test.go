package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TimeoutTestCase struct {
	timeout time.Duration
	content string // "site content"
	search  string // string to search in "site content"
}

type TimeoutTestCases map[string]TimeoutTestCase

type testServersMapType map[string]httptest.Server

func makeHandlerWithTimeout(t *testing.T, ttc *TimeoutTestCase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(ttc.timeout)
		io.WriteString(w, ttc.content)
	})
}

func TestGetContents(t *testing.T) {
	testCase := &TimeoutTestCase{
		timeout: time.Duration(time.Millisecond * 100),
		content: "<html><body> The Site aaaa </body></html>",
		search:  "",
	}
	RunGetContentsTestCase(t, testCase)
}

func RunGetContentsTestCase(t *testing.T, testCase *TimeoutTestCase) {
	handler := makeHandlerWithTimeout(t, testCase)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	content, err := getContents(ts.URL)
	assert.NoError(t, err, "Error and content", content)
	assert.Equal(t, testCase.content, content)
}

func TestCheckSite_OneContains(t *testing.T) {
	testCase := &TimeoutTestCase{
		timeout: time.Duration(time.Millisecond * 5),
		content: "This site contains search text",
		search:  "contains",
	}
	ts1 := httptest.NewServer(makeHandlerWithTimeout(t, testCase))
	defer ts1.Close()

	result, err := checkSite(ts1.URL, testCase.search)
	assert.NoError(t, err, "Error and content", result)
	assert.Contains(t, testCase.content, testCase.search)
}

func TestCheckSite_OneDoesNotContain(t *testing.T) {
	testCase := &TimeoutTestCase{
		timeout: time.Duration(time.Millisecond * 5),
		content: "This site contains search text",
		search:  "does not contains",
	}
	ts1 := httptest.NewServer(makeHandlerWithTimeout(t, testCase))
	defer ts1.Close()

	result, err := checkSite(ts1.URL, testCase.search)
	assert.NoError(t, err, "Error and content", result)
	assert.NotContains(t, testCase.content, testCase.search)
}

func TestCheckSite_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "<html><body>Not found</body></html>")
	})

	ts1 := httptest.NewServer(handler)
	defer ts1.Close()

	result, err := checkSite(ts1.URL, "Not found")
	assert.NotEqual(t, nil, err, "Error should not be nil for 404 pages")
	assert.Equal(t, false, result, "If page was not found - return false")
}

func TestCheckSite_ConnectionClosed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// suddenly close connection
		conn.Close()
	})

	ts1 := httptest.NewServer(handler)
	defer ts1.Close()

	result, err := checkSite(ts1.URL, "Not found")
	assert.NotEqual(t, nil, err, "Error should not be nil for closed connections")
	assert.Equal(t, false, result, "If error - return false")
}

func TestCheckSite_InvalidServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		bufrw.WriteString("Now we're speaking raw TCP. Say hi: ")
		bufrw.Flush()
		s, err := bufrw.ReadString('\n')
		if err != nil {
			fmt.Printf("error reading string: %v", err)
			return
		}
		fmt.Fprintf(bufrw, "You said: %q\nBye.\n", s)
		bufrw.Flush()

	})

	ts1 := httptest.NewServer(handler)
	defer ts1.Close()

	result, err := checkSite(ts1.URL, "Not found")
	assert.NotEqual(t, nil, err, "Error should not be nil for invalid server replies")
	assert.Equal(t, false, result, "If error - return false")
}

func TestCheckSites_One_Server_Two_URLS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//fmt.Printf("URL is %v\n", r.URL)
		var content string
		switch r.URL.String() {
		case "/first":
			content = "<html><body>First</body></html>"
		case "/second":
			content = "<html><body>Second</body></html>"
		default:
			content = "<html><body>Default</body></html>"
		}
		io.WriteString(w, content)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	result, err := checkSite(ts.URL+"/first", "First")
	assert.Equal(t, nil, err, "Error should be nil")
	assert.Equal(t, true, result, "The URL should return correct data")
	fmt.Print(result, err)
}

func TestCheckSites_BothDoNotContain(t *testing.T) {
	var v1, v2 bool
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "No contents - 1")
		v1 = true
	})
	ts1 := httptest.NewServer(handler1)
	defer ts1.Close()

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "No contents - 2")
		v2 = true
	})
	ts2 := httptest.NewServer(handler2)
	defer ts2.Close()

	result, err := checkSites([]string{ts1.URL, ts2.URL}, "Absent string")
	assert.Equal(t, nil, err, "Error should be nil")
	assert.Equal(t, "", result, "Should return empty string")
	assert.Equal(t, true, v1, "First server should be visited")
	assert.Equal(t, true, v2, "Second server should be visited")
}

func TestCheckSites_OnlyFirstContains(t *testing.T) {
	var v1 bool
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Present")
		v1 = true
	})
	ts1 := httptest.NewServer(handler1)
	defer ts1.Close()

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "No contents - 2")
	})
	ts2 := httptest.NewServer(handler2)
	defer ts2.Close()

	result, err := checkSites([]string{ts1.URL, ts2.URL}, "Present")
	assert.Equal(t, nil, err, "Error should be nil")
	assert.Equal(t, ts1.URL, result, "Should return first URL")
	assert.Equal(t, true, v1, "First server should be visited")
	//assert.Equal(t, true, v2, "Second server should be visited")
}

func TestCheckSites_BothContain(t *testing.T) {
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Present")
	})
	ts1 := httptest.NewServer(handler1)
	defer ts1.Close()

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Present")
	})
	ts2 := httptest.NewServer(handler2)
	defer ts2.Close()

	result, err := checkSites([]string{ts1.URL, ts2.URL}, "Present")
	assert.Equal(t, nil, err, "Error should be nil")
	assert.Contains(t, []string{ts1.URL, ts2.URL}, result, "Should be first or second ")
	//assert.(t, ts1.URL, result, "Should first or second empty string")
	//assert.NotEqual(t, v1, v2, "Ideally only one server should be touched")
	//assert.Equal(t, true, v1, "First server should be visited")
	//assert.Equal(t, true, v2, "Second server should be visited")
}

func TestCheckSites_TwoSlowHandlers_BothContain(t *testing.T) {
	var v1, v2 bool
	var wg sync.WaitGroup
	wg.Add(2)
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		v1 = true
		time.Sleep(2 * getConfigSettings("HTTP_TIMEOUT").(time.Duration)) // Use double HTTP_TIMEOUT
		io.WriteString(w, "Present")
	})
	ts1 := httptest.NewServer(handler1)
	defer ts1.Close()

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		v2 = true
		time.Sleep(2 * getConfigSettings("HTTP_TIMEOUT").(time.Duration))
		io.WriteString(w, "Present")
	})
	ts2 := httptest.NewServer(handler2)
	defer ts2.Close()

	result, err := checkSites([]string{ts1.URL, ts2.URL}, "Present")
	assert.Equal(t, nil, err, "Error should be nil")
	assert.Contains(t, []string{""}, result, "Should be empty string")
	//assert.(t, ts1.URL, result, "Should first or second empty string")
	wg.Wait()
	assert.Equal(t, true, v1, "First server should be visited")
	assert.Equal(t, true, v2, "Second server should be visited")
}

func TestCheckTextServer(t *testing.T) {
	var v1, v2 bool
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v1 = true
		io.WriteString(w, "Present")
	})
	ts1 := httptest.NewServer(handler1)
	defer ts1.Close()

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v2 = true
		io.WriteString(w, "Present")
	})
	ts2 := httptest.NewServer(handler2)
	defer ts2.Close()

	ts := httptest.NewServer(GetEngine())
	payload := Request{Sites: []string{ts1.URL, ts2.URL}, SearchText: "Present"}
	payloadSb, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Cannot marshal Request %v to json. Error: %v", payload, err)
	}
	//fmt.Printf("Payload %q \n", payload_sb)
	resp, err := http.Post(ts.URL+"/checkText", "application/json",
		bytes.NewBuffer(payloadSb))
	if err != nil {
		t.Fatalf("Cannot make post request. Err is: %v", err)
	}
	defer resp.Body.Close()

	//fmt.Printf("Response: %v\n ", resp)
	//assert.Equal(t, true, v1, "First server should be visited")
	//assert.Equal(t, true, v2, "Second server should be visited")
	bodySb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Error reading body", err)
	}
	//result := string(bodySb)
	//fmt.Printf("Body: %v \n", body)
	var decodedResponse Response
	err = json.Unmarshal(bodySb, &decodedResponse)
	if err != nil {
		t.Fatalf("Cannot decode response <%p>from server. Err: %v", bodySb, err)
	}
	assert.Contains(t, []string{ts1.URL, ts2.URL}, decodedResponse.FoundAtSite,
		"Should be first or second ")
}

func TestHealthCheckHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/checkHealth", nil)
	w := httptest.NewRecorder()
	engine := GetEngine()

	engine.ServeHTTP(w, req)
	resp := w.Result()
	bodySb, _ := ioutil.ReadAll(resp.Body)

	//fmt.Println("Resp status code: ", resp.StatusCode)
	//fmt.Println(resp.Header.Get("Content-Type"))
	//fmt.Println("Body:", string(bodySb))

	var decodedResponse interface{}
	err := json.Unmarshal(bodySb, &decodedResponse)
	if err != nil {
		t.Fatalf("Cannot decode response <%p>from server. Err: %v", bodySb, err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should be HTTP 200 OK")
	assert.Equal(t, map[string]interface{}{"status": "ok"}, decodedResponse,
		"Should return status:ok")
}

func TestCheckSitesHandler_BothDoNotContain(t *testing.T) {
	var v1, v2 bool
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "No contents - 1")
		v1 = true
	})
	ts1 := httptest.NewServer(handler1)
	defer ts1.Close()

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "No contents - 2")
		v2 = true
	})
	ts2 := httptest.NewServer(handler2)
	defer ts2.Close()

	ts := httptest.NewServer(GetEngine())
	defer ts.Close()
	payload := Request{Sites: []string{ts1.URL, ts2.URL}, SearchText: "Present"}
	payloadSb, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Cannot marshal Request %v to json. Error: %v", payload, err)
	}
	//fmt.Printf("Payload %q \n", payloadSb)
	resp, err := http.Post(ts.URL+"/checkText", "application/json",
		bytes.NewBuffer(payloadSb))
	if err != nil {
		t.Fatalf("Cannot make post request. Err is: %v", err)
	}
	defer resp.Body.Close()

	//fmt.Printf("Response: %v\n ", resp)
	assert.Equal(t, true, v1, "First server should be visited")
	assert.Equal(t, true, v2, "Second server should be visited")
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "Status should be 204 No Content")
	bodySb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Error reading body", err)
	}
	assert.Equal(t, 0, len(bodySb), "Body length for 204 should be 0 length")
	assert.Equal(t, true, v1, "First server should be visited")
	assert.Equal(t, true, v2, "Second server should be visited")
}

func TestChecksSitesHandler_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(GetEngine())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/checkText", "application/json",
		bytes.NewBuffer([]byte(`}"Invalid JSON %"`)))
	if err != nil {
		t.Fatalf("Error running test: %v", err)
	}
	//fmt.Printf("Err is %v", err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"Status should be HTTP 400 BadRequest")
	//resp := resp.Result()
	bodySb, _ := ioutil.ReadAll(resp.Body)
	var decodedResponse interface{}
	err = json.Unmarshal(bodySb, &decodedResponse)
	if err != nil {
		t.Fatalf("Cannot decode response <%p> from server. Err: %v", bodySb, err)
	}
	//assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should be HTTP 400 BadRequest")
	assert.Equal(t, map[string]interface{}{"status": "bad request"}, decodedResponse,
		"Should return status:bad request")

}

func TestMainExecution(t *testing.T) {
	go main()
	resp, err := http.Get("http://127.0.0.1:8080/checkHealth") //TODO move port to config. Where to get addr?
	if err != nil {
		t.Fatalf("Cannot make get: %v\n", err)
	}
	bodySb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading body: %v\n", err)
	}
	body := string(bodySb)
	fmt.Printf("Body: %v\n", body)
	var decodedResponse interface{}
	err = json.Unmarshal(bodySb, &decodedResponse)
	if err != nil {
		t.Fatalf("Cannot decode response <%p> from server. Err: %v", bodySb, err)
	}
	assert.Equal(t, map[string]interface{}{"status": "ok"}, decodedResponse,
		"Should return status:ok")
}
