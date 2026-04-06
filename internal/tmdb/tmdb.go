// Package tmdb provides a lightweight TMDB API client for movie/TV search.
// Author: Done-0
// Created: 2026-03-19
package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"magnet2video/configs"
)

const (
	baseURL        = "https://api.themoviedb.org/3"
	imageBaseURL   = "https://image.tmdb.org/t/p/w500"
	requestTimeout = 10 * time.Second
)

// TMDBClient is a lightweight HTTP client for the TMDB API.
type TMDBClient struct {
	apiKey     string
	httpClient *http.Client
}

// New creates a new TMDBClient from application config.
func New(config *configs.Config) *TMDBClient {
	return &TMDBClient{
		apiKey: config.TMDBConfig.APIKey,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// IsEnabled returns true if a TMDB API key is configured.
func (c *TMDBClient) IsEnabled() bool {
	return c.apiKey != ""
}

// searchResult is the raw TMDB search response item.
type searchResult struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Name          string `json:"name"`
	OriginalTitle string `json:"original_title"`
	OriginalName  string `json:"original_name"`
	ReleaseDate   string `json:"release_date"`
	FirstAirDate  string `json:"first_air_date"`
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
}

type searchResponse struct {
	Results    []searchResult `json:"results"`
	TotalPages int            `json:"total_pages"`
	Page       int            `json:"page"`
}

// movieDetail is the TMDB movie detail response (used to get imdb_id).
type movieDetail struct {
	ImdbID string `json:"imdb_id"`
}

// tvExternalIDs is the TMDB TV external IDs response.
type tvExternalIDs struct {
	ImdbID string `json:"imdb_id"`
}

// SearchResult is the public search result type.
type SearchResult struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	ReleaseDate   string `json:"release_date"`
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
	ImdbID        string `json:"imdb_id"`
	MediaType     string `json:"media_type"` // "movie" or "tv"
}

// SearchResponse is the public search response type.
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	TotalPages int            `json:"total_pages"`
	Page       int            `json:"page"`
}

// SearchMulti searches for movies and TV shows using TMDB multi search.
func (c *TMDBClient) SearchMulti(query string, page int) (*SearchResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}
	if page < 1 {
		page = 1
	}

	reqURL := fmt.Sprintf("%s/search/multi?api_key=%s&query=%s&page=%d&language=zh-CN",
		baseURL, c.apiKey, url.QueryEscape(query), page)

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("TMDB request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned status %d", resp.StatusCode)
	}

	// Multi search returns media_type in each result
	type multiResult struct {
		searchResult
		MediaType string `json:"media_type"`
	}
	type multiResponse struct {
		Results    []multiResult `json:"results"`
		TotalPages int           `json:"total_pages"`
		Page       int           `json:"page"`
	}

	var raw multiResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("TMDB decode failed: %w", err)
	}

	results := make([]SearchResult, 0, len(raw.Results))
	for _, r := range raw.Results {
		// Only include movie and tv results
		if r.MediaType != "movie" && r.MediaType != "tv" {
			continue
		}

		title := r.Title
		originalTitle := r.OriginalTitle
		releaseDate := r.ReleaseDate
		if r.MediaType == "tv" {
			title = r.Name
			originalTitle = r.OriginalName
			releaseDate = r.FirstAirDate
		}

		posterPath := ""
		if r.PosterPath != "" {
			posterPath = imageBaseURL + r.PosterPath
		}

		results = append(results, SearchResult{
			ID:            r.ID,
			Title:         title,
			OriginalTitle: originalTitle,
			ReleaseDate:   releaseDate,
			Overview:      r.Overview,
			PosterPath:    posterPath,
			MediaType:     r.MediaType,
		})
	}

	return &SearchResponse{
		Results:    results,
		TotalPages: raw.TotalPages,
		Page:       raw.Page,
	}, nil
}

// GetImdbID fetches the IMDB ID for a given TMDB ID and media type.
func (c *TMDBClient) GetImdbID(tmdbID int, mediaType string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("TMDB API key not configured")
	}

	if mediaType == "tv" {
		return c.getTVImdbID(tmdbID)
	}
	return c.getMovieImdbID(tmdbID)
}

func (c *TMDBClient) getMovieImdbID(tmdbID int) (string, error) {
	reqURL := fmt.Sprintf("%s/movie/%d?api_key=%s", baseURL, tmdbID, c.apiKey)
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("TMDB request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TMDB returned status %d", resp.StatusCode)
	}

	var detail movieDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return "", fmt.Errorf("TMDB decode failed: %w", err)
	}
	return detail.ImdbID, nil
}

func (c *TMDBClient) getTVImdbID(tmdbID int) (string, error) {
	reqURL := fmt.Sprintf("%s/tv/%d/external_ids?api_key=%s", baseURL, tmdbID, c.apiKey)
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("TMDB request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TMDB returned status %d", resp.StatusCode)
	}

	var ids tvExternalIDs
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return "", fmt.Errorf("TMDB decode failed: %w", err)
	}
	return ids.ImdbID, nil
}
