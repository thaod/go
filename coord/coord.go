/*
Copyright 2019 Tamás Gulácsi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package coord contains a function to get the coordinates of
// a human-readable address, using GMaps.
package coord

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"

	"golang.org/x/time/rate"
	errors "golang.org/x/xerrors"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const gmapsURL = `https://maps.googleapis.com/maps/api/geocode/json?key={{.APIKey}}&sensors=false&address={{.Address}}`

var (
	ErrNotFound       = errors.New("not found")
	ErrTooManyResults = errors.New("too many results")

	httpClient     = retryablehttp.NewClient()
	gmapsRateLimit = rate.NewLimiter(1, 1)

	// APIKey is the API_KEY served too Google Maps services.
	// It is set by default to the contents of the GOOGLE_MAPS_API_KEY env var.
	APIKey = os.Getenv("GOOGLE_MAPS_API_KEY")
)

type Location struct {
	Address string
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

func Get(ctx context.Context, address string) (Location, error) {
	var loc Location
	select {
	case <-ctx.Done():
		return loc, ctx.Err()
	default:
	}
	aURL := gmapsURL
	aURL = strings.Replace(aURL, "{{.Address}}", url.QueryEscape(address), 1)
	aURL = strings.Replace(aURL, "{{.APIKey}}", url.QueryEscape(APIKey), 1)
	req, err := retryablehttp.NewRequest("GET", aURL, nil)

	if err != nil {
		return loc, errors.Errorf("%s: %w", aURL, err)
	}
	req.Request = req.Request.WithContext(ctx)
	var data mapsResponse
	for i := 0; i < 10; i++ {
		if err = gmapsRateLimit.Wait(ctx); err != nil {
			return loc, err
		}
		resp, err := httpClient.Do(req)
		var sc string
		if resp != nil {
			sc = resp.Status
		}
		httpClient.Logger.(retryablehttp.Logger).Printf("GET %s: %s/%v", aURL, sc, err)
		if err != nil {
			return loc, errors.Errorf("%s: %w", aURL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode > 299 {
			return loc, errors.Errorf("%s: %w", aURL, errors.New(resp.Status))
		}

		if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return loc, errors.Errorf("decode: %w", err)
		}
		httpClient.Logger.(retryablehttp.Logger).Printf("status=%q", data.Status)
		if data.Status != "OVER_QUERY_LIMIT" {
			gmapsRateLimit.SetLimit(gmapsRateLimit.Limit() * 1.1)
			break
		}
		gmapsRateLimit.SetLimit(gmapsRateLimit.Limit() / 2)
	}

	switch data.Status {
	case "OK":
	case "ZERO_RESULTS":
		return loc, ErrNotFound
	default:
		return loc, errors.New(data.Status)
	}
	switch len(data.Results) {
	case 0:
		return loc, ErrNotFound
	case 1:
	default:
		return loc, ErrTooManyResults
	}
	result := data.Results[0]
	loc.Address = result.FormattedAddress
	loc.Lat, loc.Lng = result.Geometry.Location.Lat, result.Geometry.Location.Lng
	return loc, nil
}

type mapsResponse struct {
	Status  string       `json:"status"`
	Results []mapsResult `json:"results"`
}

type mapsResult struct {
	FormattedAddress string       `json:"formatted_address"`
	Geometry         mapsGeometry `json:"geometry"`
}
type mapsGeometry struct {
	Location mapsLocation `json:"location"`
}
type mapsLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
