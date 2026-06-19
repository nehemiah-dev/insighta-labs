package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"

	"net/http"
	"net/url"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any {
		"message": "Hurray!!! API is up and running",
	})
}

func classifyHandler(w http.ResponseWriter, r *http.Request) {
	allowedParams := map[string]bool{"name": true}
	for key := range r.URL.Query() {
		if !allowedParams[key] {
			http.Error(w, fmt.Sprintf("unexpected query parameter: %s", key), http.StatusBadRequest)
		}
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing name parameter", http.StatusBadRequest)
		return
	}
	client := &http.Client{
		Timeout: 10 * time.Second,

	}
	baseURL, _ := url.Parse("https://api.genderize.io")
	params := url.Values{}
	params.Add("name", name)
	baseURL.RawQuery = params.Encode()

	resp, err := client.Get(baseURL.String())
	if err != nil {
		var netError net.Error
		if errors.As(err, &netError) && netError.Timeout() {
			fmt.Println("upstream timed out")
		} else {
			fmt.Println("upstream unreachable:", err)
		}
		http.Error(w, "classify service is currently unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	processedAt := time.Now().UTC()
	if resp.StatusCode == 429 {
		http.Error(w, "upstream rate limit reached, try again later", http.StatusTooManyRequests)
		return
	}
	if resp.StatusCode != 200 {
		http.Error(w, "upstream service error", http.StatusInternalServerError)
		return
	}
	var result struct {
		Name string	`json:"name"`
		Gender string `json:"gender"`
		Count int `json:"count"`
		Probability float64 `json:"probability"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		http.Error(w, "failed to parse response", http.StatusInternalServerError)
		return
	}
	if result.Gender == "" {
		writeJSON(w, http.StatusOK, map[string]any{
        "name": result.Name,
        "gender": nil,
        "message": "no prediction available for this name",
        "processed_at": processedAt,
    })
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": result.Name,
		"gender": result.Gender,
		"sample_size": result.Count,
		"probability": result.Probability,
		"processed_at": processedAt,
	})
	
}

func main(){
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/classify", classifyHandler)

	srv := &http.Server{
		Addr: ":8080",
		Handler: mux,
		ReadTimeout: 5 * time.Second,
		IdleTimeout: 20 * time.Second,
	}
	fmt.Println("Server is running on :8080")
	log.Fatal(srv.ListenAndServe())
}