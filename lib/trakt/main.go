package trakt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/xanderstrike/goplaxt/lib/store"
	"github.com/xanderstrike/goplaxt/tracing"
	"github.com/xanderstrike/plexhooks"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

const (
	traktApiBasePath = "https://api.trakt.tv"
)

// AuthRequest authorize the connection with Trakt
func AuthRequest(ctx context.Context, root, username, code, refreshToken, grantType string) (map[string]interface{}, error) {
	values := map[string]string{
		"code":          code,
		"refresh_token": refreshToken,
		"client_id":     os.Getenv("TRAKT_ID"),
		"client_secret": os.Getenv("TRAKT_SECRET"),
		"redirect_uri":  fmt.Sprintf("%s/authorize?username=%s", root, url.PathEscape(username)),
		"grant_type":    grantType,
	}
	jsonValue, _ := json.Marshal(values)
	resp, err := otelhttp.Post(ctx, fmt.Sprintf("%s/oauth/token", traktApiBasePath), "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return result, nil
}

// Handle determine if an item is a show or a movie
func Handle(ctx context.Context, pr plexhooks.PlexResponse, user store.User, log *log.Entry) error {
	ctx, span := tracing.Tracer.Start(ctx, "trakt.Handle")
	defer span.End()

	var err error
	if pr.Metadata.LibrarySectionType == "show" {
		err = HandleShow(ctx, pr, user.AccessToken, log)
	} else if pr.Metadata.LibrarySectionType == "movie" {
		err = HandleMovie(ctx, pr, user.AccessToken, log)
	} else {
		log.Errorf("Unsupported media type: %s", pr.Metadata.LibrarySectionType)
		return nil
	}
	return trace.Wrap(err)
}

// HandleShow start the scrobbling for a show
func HandleShow(ctx context.Context, pr plexhooks.PlexResponse, accessToken string, log *log.Entry) error {
	ctx, span := tracing.Tracer.Start(ctx, "trakt.HandleShow")
	defer span.End()
	showInfo, err := findShowInfo(ctx, pr, log)
	if err != nil {
		return trace.Wrap(err)
	}
	episode, err := getExtendedEpisodeInfo(ctx, showInfo, log)
	if err != nil {
		return trace.Wrap(err)
	}
	event, progress := getAction(ctx, pr, episode.Runtime*60*1000)

	scrobbleObject := ShowScrobbleBody{
		Progress: progress,
		Episode:  *episode,
	}

	scrobbleJSON, err := json.Marshal(scrobbleObject)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = scrobbleRequest(ctx, event, scrobbleJSON, accessToken)
	return trace.Wrap(err)
}

// HandleMovie start the scrobbling for a movie
func HandleMovie(ctx context.Context, pr plexhooks.PlexResponse, accessToken string, log *log.Entry) error {
	ctx, span := tracing.Tracer.Start(ctx, "trakt.HandleMovie")
	defer span.End()
	event, progress := getAction(ctx, pr, 0)

	movie, err := findMovie(ctx, pr, log)
	if err != nil {
		return trace.Wrap(err)
	}
	scrobbleObject := MovieScrobbleBody{
		Progress: progress,
		Movie:    *movie,
	}

	scrobbleJSON, err := json.Marshal(scrobbleObject)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = scrobbleRequest(ctx, event, scrobbleJSON, accessToken)
	return trace.Wrap(err)
}

func findShowInfo(ctx context.Context, pr plexhooks.PlexResponse, log *log.Entry) (*ShowInfo, error) {
	ctx, span := tracing.Tracer.Start(ctx, "trakt.findShowInfo")
	defer span.End()
	log = log.WithContext(ctx)
	var showInfo []ShowInfo
	var episodeID string

	log.Println("Finding episode with new Plex TV agent")

	traktService := pr.Metadata.ExternalGuid[0].Id[:4]
	episodeID = pr.Metadata.ExternalGuid[0].Id[7:]

	// The new Plex TV agent use episode ID instead of show ID,
	// so we need to do things a bit differently
	URL := fmt.Sprintf("%s/search/%s/%s?type=episode", traktApiBasePath, traktService, episodeID)

	respBody, err := makeRequest(ctx, URL)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	err = json.Unmarshal(respBody, &showInfo)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if len(showInfo) == 0 {
		return nil, trace.BadParameter("No show found on trakt %s", URL)
	}

	log.Print(fmt.Sprintf("Tracking %s - S%02dE%02d using %s", showInfo[0].Show.Title, showInfo[0].Episode.Season, showInfo[0].Episode.Number, traktService))

	return &showInfo[0], nil
}

func getExtendedEpisodeInfo(ctx context.Context, showInfo *ShowInfo, log *log.Entry) (*Episode, error) {
	ctx, span := tracing.Tracer.Start(ctx, "trakt.getExtendedEpisodeInfo")
	defer span.End()

	log = log.WithContext(ctx).
		WithFields(logrus.Fields{
			"show":    showInfo.Show.Title,
			"season":  showInfo.Episode.Season,
			"episode": showInfo.Episode.Number,
		})

	log.Print("Getting extended episode info")
	url := fmt.Sprintf(
		"%s/shows/%d/seasons/%d/episodes/%d?extended=full",
		traktApiBasePath,
		showInfo.Show.Ids.Trakt,
		showInfo.Episode.Season,
		showInfo.Episode.Number,
	)

	responseBody, err := makeRequest(ctx, url)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	var episode Episode
	err = json.Unmarshal(responseBody, &episode)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &episode, nil

}

func findMovie(ctx context.Context, pr plexhooks.PlexResponse, log *log.Entry) (*Movie, error) {
	ctx, span := tracing.Tracer.Start(ctx, "trakt.findMovie")
	defer span.End()
	log = log.WithContext(ctx).WithFields(logrus.Fields{
		"title": pr.Metadata.Title,
		"year":  pr.Metadata.Year,
	})
	log.Print("Finding movie")
	url := fmt.Sprintf(
		"%s/search/movie?query=%s",
		traktApiBasePath,
		url.PathEscape(pr.Metadata.Title),
	)

	respBody, err := makeRequest(ctx, url)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var results []MovieSearchResult

	err = json.Unmarshal(respBody, &results)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	for _, result := range results {
		if result.Movie.Year == pr.Metadata.Year {
			return &result.Movie, nil
		}
	}
	return nil, trace.Errorf("Could not find movie!")
}

func makeRequest(ctx context.Context, url string) ([]byte, error) {
	client := httpClient()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("trakt-api-version", "2")
	req.Header.Add("trakt-api-key", os.Getenv("TRAKT_ID"))

	resp, err := client.Do(req)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("trakt response error %s", resp.Status)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return respBody, nil
}

func httpClient() *http.Client {
	return &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
}

func scrobbleRequest(ctx context.Context, action string, body []byte, accessToken string) ([]byte, error) {
	ctx, span := tracing.Tracer.Start(ctx, "trakt.scrobbleRequest")
	defer span.End()
	span.SetAttributes(attribute.String("action", action))
	client := httpClient()
	url := fmt.Sprintf("%s/scrobble/%s", traktApiBasePath, action)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("trakt-api-version", "2")
	req.Header.Add("trakt-api-key", os.Getenv("TRAKT_ID"))

	resp, err := client.Do(req)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return respBody, nil
}

func getAction(ctx context.Context, pr plexhooks.PlexResponse, runtime int) (string, int) {
	percentage := calculatePercentage(ctx, pr, runtime)
	switch pr.Event {
	case "media.play":
		return "start", percentage
	case "media.pause":
		return "stop", percentage
	case "media.resume":
		return "start", percentage
	case "media.stop":
		return "stop", percentage
	case "media.scrobble":
		return "stop", 90
	}
	return "", percentage
}

func calculatePercentage(ctx context.Context, pr plexhooks.PlexResponse, runtime int) int {
	duration := math.Max(float64(pr.Metadata.Duration), float64(runtime))
	offset := float64(pr.Metadata.ViewOffset)
	percentage := int(offset / duration * 100)
	log.WithContext(ctx).WithFields(log.Fields{
		"duration": duration,
		"offset":   offset,
	}).Debugf("Calculated percentage: %d", percentage)

	return percentage
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
