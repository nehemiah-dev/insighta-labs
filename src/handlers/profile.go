package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"unicode/utf8"

	"github.com/insighta-labs/src/services"
	"github.com/insighta-labs/src/store"
)

type ProfileHandler struct {
	service *services.ProfileService
}

func NewProfileHandler(s *services.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: s}
}

var nameRegex = regexp.MustCompile(`^[A-Za-zÀ-ÖØ-öø-ÿ]+(?:[ '-][A-Za-zÀ-ÖØ-öø-ÿ]+)*$`)

func isValidName(name string) bool {
	length := utf8.RuneCountInString(name)
	if length < 1 || length > 100 {
		return false
	}
	return nameRegex.MatchString(name)
}

type createProfileRequest struct {
	Name string `json:"name"`
}

// POST /api/profiles
func (h *ProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !isValidName(req.Name) {
		writeError(w, http.StatusUnprocessableEntity, "name must contain only letters, spaces, apostrophes, or hyphens")
		return
	}

	profile, err := h.service.CreateProfile(r.Context(), req.Name)

	if errors.Is(err, services.ErrAlreadyExists) {
		writeSuccessWithMessage(w, http.StatusOK, "Profile already exists", profile)
		return
	}

	var upstreamErr *services.UpstreamFailure
	if errors.As(err, &upstreamErr) {
		writeError(w, http.StatusBadGateway, upstreamErr.Error())
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create profile")
		return
	}

	writeSuccess(w, http.StatusCreated, profile)
}

// GET /api/profiles/{id}
func (h *ProfileHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	profile, err := h.service.GetProfile(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch profile")
		return
	}

	writeSuccess(w, http.StatusOK, profile)
}

// listProfileView is the slimmer shape used in the list response —
// same data, fewer fields, per the spec's example.
type listProfileView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Gender    string `json:"gender"`
	Age       int    `json:"age"`
	AgeGroup  string `json:"age_group"`
	CountryID string `json:"country_id"`
}

// GET /api/profiles?gender=&country_id=&age_group=
func (h *ProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	filters := store.ListFilters{
		Gender:    r.URL.Query().Get("gender"),
		CountryID: r.URL.Query().Get("country_id"),
		AgeGroup:  r.URL.Query().Get("age_group"),
	}

	profiles, err := h.service.ListProfiles(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list profiles")
		return
	}

	views := make([]listProfileView, 0, len(profiles))
	for _, p := range profiles {
		views = append(views, listProfileView{
			ID:        p.ID,
			Name:      p.Name,
			Gender:    p.Gender,
			Age:       p.Age,
			AgeGroup:  p.AgeGroup,
			CountryID: p.CountryID,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"count":  len(views),
		"data":   views,
	})
}

// DELETE /api/profiles/{id}
func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.service.DeleteProfile(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete profile")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

