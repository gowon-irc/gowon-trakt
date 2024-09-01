package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	traktAPIHistoryURL  = "https://api.trakt.tv/users/%s/history"
	traktAPIWatchingURL = "https://api.trakt.tv/users/%s/watching"
)

type traktJSON []Entry

func (tj traktJSON) Latest() string {
	return tj[0].String()
}

// Entry represents a single entry, film or episode, in the trakt json
type Entry struct {
	Type    string  `json:"type"`
	Episode Episode `json:"episode,omitempty"`
	Show    Show    `json:"show,omitempty"`
	Movie   Movie   `json:"movie,omitempty"`
}

func (e Entry) String() string {
	if e.Type == "episode" {
		return fmt.Sprintf("%s %s", e.Show, e.Episode)
	}

	if e.Type == "movie" {
		return e.Movie.String()
	}

	return "unknown"
}

// Episode represents a single episode in the trakt json
type Episode struct {
	Season int    `json:"season"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}

func (e Episode) String() string {
	return fmt.Sprintf("%02dx%02d - %s", e.Season, e.Number, e.Title)
}

// Show contains the parent show information for the returned episode
type Show struct {
	Title string `json:"title"`
}

func (s Show) String() string {
	return s.Title
}

// Movie represents a single movie in the trakt json
type Movie struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
}

func (m Movie) String() string {
	return fmt.Sprintf("%s (%d)", m.Title, m.Year)
}

func trakt(user, apiKey string) (msg string, err error) {
	// Get user watching
	url := fmt.Sprintf(traktAPIWatchingURL, user)

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("trakt-api-version", "2")
	req.Header.Add("trakt-api-key", apiKey)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return "", err
		}

		wj := &Entry{}

		err = json.Unmarshal(body, &wj)
		if err != nil {
			return "", err
		}

		out := fmt.Sprintf("%s is watching: %s", user, wj.String())

		return out, nil
	}

	// Get user history
	url = fmt.Sprintf(traktAPIHistoryURL, user)

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("trakt-api-version", "2")
	req.Header.Add("trakt-api-key", apiKey)

	res, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Sprintf("User %s not found", user), nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	hj := &traktJSON{}

	err = json.Unmarshal(body, &hj)
	if err != nil {
		return "", err
	}

	if len(*hj) == 0 {
		return fmt.Sprintf("%s has not watched anything", user), nil
	}

	out := fmt.Sprintf("%s last watched: %s", user, hj.Latest())

	return out, nil
}
