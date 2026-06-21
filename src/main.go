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
	"regexp"
	"unicode/utf8"
)

var nameRegex = regexp.MustCompile(`^[A-Za-zÀ-ÖØ-öø-ÿ]+(?:[ '-][A-Za-zÀ-ÖØ-öø-ÿ]+)*$`)
func isValidName(name string) bool {
	length := utf8.RuneCountInString(name)
	if length < 1 || length > 100 {
		return false
	}
	return nameRegex.MatchString(name)
}
// ---- response envelope helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeSuccess(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, map[string]any{
		"status": "success",
		"data":   data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"status":  "error",
		"message": message,
	})
}

func isConfident(probability float64, sampleSize int) bool {
	const minProbability = 0.7
	const minSampleSize = 100
	return probability >= minProbability && sampleSize >= minSampleSize
}
// ---- handlers ----

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		"message": "Hurray!!! API is up and running",
	})
}

func classifyHandler(w http.ResponseWriter, r *http.Request) {
	allowedParams := map[string]bool{"name": true}
	for key := range r.URL.Query() {
		if !allowedParams[key] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unexpected query parameter: %s", key))
			return
		}
	}

	name := r.URL.Query().Get("name")
	if !isValidName(name) {
		writeError(w, http.StatusUnprocessableEntity, "unable to process name, please use a valid name")
		return
	}

	client := &http.Client{Timeout: 20 * time.Second}

	baseURL, _ := url.Parse("https://api.genderize.io")
	params := url.Values{}
	params.Add("name", name)
	baseURL.RawQuery = params.Encode()

	resp, err := client.Get(baseURL.String())
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			log.Println("upstream timed out:", err)
		} else {
			log.Println("upstream unreachable:", err)
		}
		writeError(w, http.StatusServiceUnavailable, "classify service is currently unavailable")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		writeError(w, http.StatusTooManyRequests, "upstream rate limit reached, try again later")
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("upstream returned unexpected status: %d", resp.StatusCode)
		writeError(w, http.StatusBadGateway, "upstream service error")
		return
	}

	var result struct {
		Name        string  `json:"name"`
		Gender      string  `json:"gender"`
		Count       int     `json:"count"`
		Probability float64 `json:"probability"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("failed to decode upstream response:", err)
		writeError(w, http.StatusInternalServerError, "failed to parse response")
		return
	}

	processedAt := time.Now().UTC()

	if result.Gender == "" {
		writeSuccess(w, http.StatusOK, map[string]any{
			"name":         result.Name,
			"gender":       nil,
			"is_confident": false,
			"processed_at": processedAt,
		})
		return
	}

	writeSuccess(w, http.StatusOK, map[string]any{
		"name":         result.Name,
		"gender":       result.Gender,
		"probability":  result.Probability,
		"sample_size":  result.Count,
		"is_confident": isConfident(result.Probability, result.Count),
		"processed_at": processedAt,
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/classify", classifyHandler)

	srv := &http.Server{
		Addr:        ":8080",
		Handler:     mux,
		ReadTimeout: 5 * time.Second,
		IdleTimeout: 20 * time.Second,
	}

	fmt.Println("Server is running on :8080")
	log.Fatal(srv.ListenAndServe())
}