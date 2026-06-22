package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/insighta-labs/src/models" 
)

// ErrNotFound is returned when a profile lookup finds nothing.
// Defining it once here lets callers check `errors.Is(err, store.ErrNotFound)`
// instead of comparing error strings.
var ErrNotFound = errors.New("profile not found")

// CreateProfile inserts a new profile and returns it with its generated id/created_at filled in.
func (s *Store) CreateProfile(ctx context.Context, p models.Profile) (*models.Profile, error) {
	query := `
		INSERT INTO profiles (name, gender, gender_probability, sample_size, age, age_group, country_id, country_probability)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	err := s.pool.QueryRow(ctx, query,
		p.Name, p.Gender, p.GenderProbability, p.SampleSize, p.Age, p.AgeGroup, p.CountryID, p.CountryProbability,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert profile: %w", err)
	}

	return &p, nil
}

// GetProfileByName looks up a profile by name (used for the dedupe check).
// Returns ErrNotFound if no row matches — not a generic error.
func (s *Store) GetProfileByName(ctx context.Context, name string) (*models.Profile, error) {
	query := `
		SELECT id, name, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at
		FROM profiles
		WHERE name = $1
	`

	var p models.Profile
	err := s.pool.QueryRow(ctx, query, name).Scan(
		&p.ID, &p.Name, &p.Gender, &p.GenderProbability, &p.SampleSize,
		&p.Age, &p.AgeGroup, &p.CountryID, &p.CountryProbability, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get profile by name: %w", err)
	}

	return &p, nil
}

// GetProfileByID looks up a profile by its UUID.
func (s *Store) GetProfileByID(ctx context.Context, id string) (*models.Profile, error) {
	query := `
		SELECT id, name, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at
		FROM profiles
		WHERE id = $1
	`

	var p models.Profile
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Gender, &p.GenderProbability, &p.SampleSize,
		&p.Age, &p.AgeGroup, &p.CountryID, &p.CountryProbability, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get profile by id: %w", err)
	}

	return &p, nil
}

// ListFilters holds the optional query params for listing profiles.
// Empty string means "no filter on this field."
type ListFilters struct {
	Gender   string
	CountryID string
	AgeGroup string
}

// ListProfiles returns profiles matching the given filters (case-insensitive).
// Filters are combined with AND; an empty filter is skipped entirely.
func (s *Store) ListProfiles(ctx context.Context, filters ListFilters) ([]models.Profile, error) {
	query := `
		SELECT id, name, gender, gender_probability, sample_size, age, age_group, country_id, country_probability, created_at
		FROM profiles
		WHERE ($1 = '' OR LOWER(gender) = LOWER($1))
		  AND ($2 = '' OR LOWER(country_id) = LOWER($2))
		  AND ($3 = '' OR LOWER(age_group) = LOWER($3))
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, filters.Gender, filters.CountryID, filters.AgeGroup)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}
	defer rows.Close()

	var profiles []models.Profile
	for rows.Next() {
		var p models.Profile
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Gender, &p.GenderProbability, &p.SampleSize,
			&p.Age, &p.AgeGroup, &p.CountryID, &p.CountryProbability, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan profile row: %w", err)
		}
		profiles = append(profiles, p)
	}

	// rows.Next() can stop early due to an error mid-iteration, not just "no more rows" —
	// always check rows.Err() after the loop to catch that.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profile rows: %w", err)
	}

	return profiles, nil
}

// DeleteProfile removes a profile by id. Returns ErrNotFound if nothing was deleted.
func (s *Store) DeleteProfile(ctx context.Context, id string) error {
	query := `DELETE FROM profiles WHERE id = $1`

	tag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}