package clients

const agifyURL = "https://api.agify.io"

type AgifyResult struct {
	Age   *int `json:"age"`
	Count int  `json:"count"`
}

func FetchAgify(name string) (*AgifyResult, error) {
	var result AgifyResult
	if err := fetchJSON("Agify", agifyURL, name, &result); err != nil {
		return nil, err
	}
	return &result, nil
}