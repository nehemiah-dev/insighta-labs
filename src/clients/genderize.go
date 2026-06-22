package clients

const genderizeURL = "https://api.genderize.io"

type GenderizeResult struct {
	Gender      string  `json:"gender"`
	Probability float64 `json:"probability"`
	Count       int     `json:"count"`
}

func FetchGenderize(name string) (*GenderizeResult, error) {
	var result GenderizeResult
	if err := fetchJSON("Genderize", genderizeURL, name, &result); err != nil {
		return nil, err
	}
	return &result, nil
}