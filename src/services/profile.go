package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/insighta-labs/src/clients"
	"github.com/insighta-labs/src/models"
	"github.com/insighta-labs/src/store"
)

type ProfileService struct {
	store *store.Store
}

func NewProfileService(s *store.Store) *ProfileService {
	return &ProfileService{store: s}
}

// fetchResults holds the outcome of all three concurrent API calls.
// Each goroutine writes to exactly one field — no two goroutines touch the
// same field, so there's no need for a mutex here despite the concurrency.
type fetchResults struct {
	genderize *clients.GenderizeResult
	agify     *clients.AgifyResult
	nationalize *clients.NationalizeResult

	genderizeErr error
	agifyErr     error
	nationalizeErr error
}

// fetchAll calls genderize, agify, and nationalize concurrently and waits for all three.
func fetchAll(ctx context.Context, name string) fetchResults {
	var wg sync.WaitGroup
	var results fetchResults

	wg.Add(3)

	go func() {
		defer wg.Done()
		results.genderize, results.genderizeErr = clients.FetchGenderize(name)
	}()

	go func() {
		defer wg.Done()
		results.agify, results.agifyErr = clients.FetchAgify(name)
	}()

	go func() {
		defer wg.Done()
		results.nationalize, results.nationalizeErr = clients.FetchNationalize(name)
	}()

	wg.Wait()
	return results
}

// UpstreamFailure is returned when one of the 3 external calls failed or
// returned data we can't use (null gender, null age, no country).
// The handler layer turns this into the spec's 502 message.
type UpstreamFailure struct {
	Service string
}

func (e *UpstreamFailure) Error() string {
	return fmt.Sprintf("%s returned an invalid response", e.Service)
}

var ErrAlreadyExists = errors.New("profile already exists")

// CreateProfile orchestrates the full flow: dedupe check -> fetch 3 APIs
// concurrently -> validate -> classify -> store.
func (s *ProfileService) CreateProfile(ctx context.Context, name string) (*models.Profile, error) {
	// 1. dedupe check first — avoid wasted API calls if we already have this name
	existing, err := s.store.GetProfileByName(ctx, name)
	if err == nil {
		return existing, ErrAlreadyExists
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("checking existing profile: %w", err)
	}

	// 2. fetch all three concurrently
	results := fetchAll(ctx, name)

	if results.genderizeErr != nil {
		return nil, &UpstreamFailure{Service: "Genderize"}
	}
	if results.agifyErr != nil {
		return nil, &UpstreamFailure{Service: "Agify"}
	}
	if results.nationalizeErr != nil {
		return nil, &UpstreamFailure{Service: "Nationalize"}
	}

	// 3. validate the data itself, not just transport errors
	if results.genderize.Gender == "" || results.genderize.Count == 0 {
		return nil, &UpstreamFailure{Service: "Genderize"}
	}
	if results.agify.Age == nil {
		return nil, &UpstreamFailure{Service: "Agify"}
	}
	if len(results.nationalize.Country) == 0 {
		return nil, &UpstreamFailure{Service: "Nationalize"}
	}

	// 4. classify
	age := *results.agify.Age
	topCountry := topCountry(results.nationalize.Country)

	profile := models.Profile{
		Name:               name,
		Gender:             results.genderize.Gender,
		GenderProbability:  results.genderize.Probability,
		SampleSize:         results.genderize.Count,
		Age:                age,
		AgeGroup:           models.AgeGroup(age),
		CountryID:          topCountry.CountryID,
		CountryProbability: topCountry.Probability,
	}

	// 5. store
	created, err := s.store.CreateProfile(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("storing profile: %w", err)
	}

	return created, nil
}

// topCountry returns the country with the highest probability.
func topCountry(countries []clients.NationalizeCountry) clients.NationalizeCountry {
	best := countries[0]
	for _, c := range countries[1:] {
		if c.Probability > best.Probability {
			best = c
		}
	}
	return best
}

func (s *ProfileService) GetProfile(ctx context.Context, id string) (*models.Profile, error) {
	return s.store.GetProfileByID(ctx, id)
}

func (s *ProfileService) ListProfiles(ctx context.Context, filters store.ListFilters) ([]models.Profile, error) {
	return s.store.ListProfiles(ctx, filters)
}

func (s *ProfileService) DeleteProfile(ctx context.Context, id string) error {
	return s.store.DeleteProfile(ctx, id)
}