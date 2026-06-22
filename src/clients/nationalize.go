package clients

const nationalizeURL = "https://api.nationalize.io"

type NationalizeCountry struct {
	CountryID   string  `json:"country_id"`
	Probability float64 `json:"probability"`
}

type NationalizeResult struct {
	Country []NationalizeCountry `json:"country"`
}

func FetchNationalize(name string) (*NationalizeResult, error) {
	var result NationalizeResult
	if err := fetchJSON("Nationalize", nationalizeURL, name, &result); err != nil {
		return nil, err
	}
	return &result, nil
}