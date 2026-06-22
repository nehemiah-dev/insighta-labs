package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// sharedClient is reused across all three APIs — one connection pool, one timeout policy.
var sharedClient = &http.Client{
	Timeout: 20 * time.Second,
}

// UpstreamError means a specific external API failed or returned unusable data.
// The Service field carries which one (genderize/agify/nationalize) so the
// handler layer can build the "${externalApi} returned an invalid response" message.
type UpstreamError struct {
	Service string
	Err     error
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("%s: %v", e.Service, e.Err)
}

func (e *UpstreamError) Unwrap() error {
	return e.Err
}

// fetchJSON is the shared low-level call: GET {baseURL}?name={name}, decode JSON into out.
// Each of genderize.go / agify.go / nationalize.go calls this with their own URL + target struct.
func fetchJSON(serviceName, baseURL, name string, out any) error {
	u, err := url.Parse(baseURL)
	if err != nil {
		return &UpstreamError{Service: serviceName, Err: err}
	}
	params := url.Values{}
	params.Add("name", name)
	u.RawQuery = params.Encode()

	resp, err := sharedClient.Get(u.String())
	if err != nil {
		return &UpstreamError{Service: serviceName, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &UpstreamError{
			Service: serviceName,
			Err:     fmt.Errorf("unexpected status %d", resp.StatusCode),
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &UpstreamError{Service: serviceName, Err: err}
	}
	return nil
}