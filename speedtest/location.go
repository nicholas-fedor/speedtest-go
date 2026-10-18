// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
)

const (
	coordinateCountForLatLon = 2
	maxLatitude              = 90
	maxLongitude             = 180
)

var (
	// ErrNotFoundLocation is returned when a location lookup fails.
	ErrNotFoundLocation = errors.New("not found location")
	// ErrInvalidLocationInput is returned when the location input is invalid.
	ErrInvalidLocationInput = errors.New("invalid location input")
	// ErrInvalidInput is returned when the input is invalid.
	ErrInvalidInput = errors.New("invalid input")
)

// Location represents a geographical location with name, country code, latitude, and longitude.
type Location struct {
	Name string
	CC   string
	Lat  float64
	Lon  float64
}

// locations are the predefined city labels, keyed by lower-case label. Nothing changes them, so they need no lock.
var locations = map[string]Location{
	"brasilia":     {"brasilia", "br", -15.793876, -47.8835327},
	"hongkong":     {"hongkong", "hk", 22.3106806, 114.1700546},
	"tokyo":        {"tokyo", "jp", 35.680938, 139.7674114},
	"london":       {"london", "gb", 51.5072493, -0.1288861},
	"moscow":       {"moscow", "ru", 55.7497248, 37.615989},
	"beijing":      {"beijing", "cn", 39.8721243, 116.4077473},
	"paris":        {"paris", "fr", 48.8626026, 2.3477229},
	"sanfrancisco": {"sanfrancisco", "us", 37.7540028, -122.4429967},
	"newyork":      {"newyork", "us", 40.7200876, -74.0220945},
	"yishun":       {"yishun", "sg", 1.4230218, 103.8404728},
	"delhi":        {"delhi", "in", 28.6251287, 77.1960896},
	"monterrey":    {"monterrey", "mx", 25.6881435, -100.3073485},
	"berlin":       {"berlin", "de", 52.5168128, 13.4009469},
	"maputo":       {"maputo", "mz", -25.9579267, 32.5760444},
	"honolulu":     {"honolulu", "us", 20.8247065, -156.918706},
	"seoul":        {"seoul", "kr", 37.6086268, 126.7179721},
	"osaka":        {"osaka", "jp", 34.6952743, 135.5006967},
	"shanghai":     {"shanghai", "cn", 31.2292105, 121.4661666},
	"urumqi":       {"urumqi", "cn", 43.8256624, 87.6058564},
	"ottawa":       {"ottawa", "ca", 45.4161836, -75.7035467},
	"capetown":     {"capetown", "za", -33.9391993, 18.4316716},
	"sydney":       {"sydney", "au", -33.8966622, 151.1731861},
	"perth":        {"perth", "au", -31.9551812, 115.8591904},
	"warsaw":       {"warsaw", "pl", 52.2396659, 21.0129345},
	"kampala":      {"kampala", "ug", 0.3070027, 32.5675581},
	"bangkok":      {"bangkok", "th", 13.7248936, 100.493026},
}

// Cities returns the predefined city labels that [GetLocation] accepts.
//
// Returns:
//   - []Location: a copy of each predefined location, sorted by label.
func Cities() []Location {
	cities := slices.Collect(maps.Values(locations))
	slices.SortFunc(cities, func(a, b Location) int { return strings.Compare(a.Name, b.Name) })

	return cities
}

// GetLocation returns a predefined city by its label.
//
// Parameters:
//   - locationName: the label, in any case, such as "tokyo".
//
// Returns:
//   - *Location: a copy of the city's location.
//   - error: [ErrNotFoundLocation] for a label that is not predefined.
func GetLocation(locationName string) (*Location, error) {
	loc, ok := locations[strings.ToLower(locationName)]
	if !ok {
		return nil, ErrNotFoundLocation
	}

	return &loc, nil
}

// NewLocation returns a location with a name and coordinates and no country code. It does not add a city label.
//
// Parameters:
//   - locationName: the name.
//   - latitude: the latitude in decimal degrees.
//   - longitude: the longitude in decimal degrees.
//
// Returns:
//   - *Location: the location.
func NewLocation(locationName string, latitude, longitude float64) *Location {
	return &Location{Name: locationName, Lat: latitude, Lon: longitude}
}

// ParseLocation returns a location at the coordinates in a "lat,lon" string, named "Custom-" and the given name.
//
// Parameters:
//   - locationName: the name, or empty for "Custom-Default".
//   - coordinateStr: the latitude and longitude, as [ParseCoordinates] accepts them.
//
// Returns:
//   - *Location: the location.
//   - error: the [ParseCoordinates] error.
func ParseLocation(locationName, coordinateStr string) (*Location, error) {
	lat, lon, err := ParseCoordinates(coordinateStr)
	if err != nil {
		return nil, err
	}

	if len(locationName) == 0 {
		return NewLocation("Custom-Default", lat, lon), nil
	}

	return NewLocation("Custom-"+locationName, lat, lon), nil
}

// ParseCoordinates parses a "lat,lon" pair.
//
// Parameters:
//   - text: the latitude and longitude in decimal degrees, separated by a comma, such as "60,-110".
//
// Returns:
//   - latitude: the latitude, from -90 to 90.
//   - longitude: the longitude, from -180 to 180.
//   - err: an error wrapping [ErrInvalidLocationInput] when text is not two values, or [ErrInvalidInput] when a value
//     is not a number or is out of range.
//
//nolint:nonamedreturns // Same-type returns need names.
func ParseCoordinates(text string) (latitude, longitude float64, err error) {
	coordinates := strings.Split(text, ",")
	if len(coordinates) != coordinateCountForLatLon {
		return 0, 0, fmt.Errorf("%w: %s", ErrInvalidLocationInput, text)
	}

	latitude, err = betweenRange(strings.TrimSpace(coordinates[0]), maxLatitude)
	if err != nil {
		return 0, 0, err
	}

	longitude, err = betweenRange(strings.TrimSpace(coordinates[1]), maxLongitude)
	if err != nil {
		return 0, 0, err
	}

	return latitude, longitude, nil
}

// String returns the string representation of the location.
func (l *Location) String() string {
	return fmt.Sprintf("(%s) [%v, %v]", l.Name, l.Lat, l.Lon)
}

// betweenRange latitude and longitude range check.
func betweenRange(inputStrValue string, interval float64) (float64, error) {
	value, err := strconv.ParseFloat(inputStrValue, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidInput, inputStrValue)
	}

	// NaN fails every comparison, so it would pass the range check below unless rejected first.
	if math.IsNaN(value) || value < -interval || interval < value {
		return 0, fmt.Errorf("invalid input. got: %v, expected between -%v and %v: %w",
			inputStrValue, interval, interval, ErrInvalidInput)
	}

	return value, nil
}
