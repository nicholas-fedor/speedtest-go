// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Country filter errors.
var (
	// errNoServersInCountries is returned when the country filter leaves no servers.
	errNoServersInCountries = errors.New("no servers in the selected countries")
	// errUnknownCountry is returned when the user's own country cannot be determined for auto.
	errUnknownCountry = errors.New("cannot determine your country for --filter-cc auto")
)

// filterByCountry keeps the servers in the configured countries.
//
// Parameters:
//   - client: the client, which looks up the user's country for auto.
//   - cfg: the settings, with the country codes and the city.
//   - servers: the servers to filter.
//
// Returns:
//   - speedtest.Servers: the servers in those countries, in their original order.
//   - []string: the country codes used, with auto resolved.
//   - error: the error resolving auto, or an error wrapping errNoServersInCountries when no server matches.
func filterByCountry(
	client *speedtest.Speedtest,
	cfg config.Config,
	servers speedtest.Servers,
) (speedtest.Servers, []string, error) {
	codes := make([]string, 0, len(cfg.CountryCodes))

	for _, code := range cfg.CountryCodes {
		if code == config.CountryAuto {
			country, err := autoCountry(client, cfg)
			if err != nil {
				return nil, nil, err
			}

			code = country
		}

		codes = append(codes, code)
	}

	filtered := servers.FilterCountry(codes...)
	if len(filtered) == 0 {
		return nil, codes, fmt.Errorf("%w: %s", errNoServersInCountries, strings.Join(codes, ", "))
	}

	return filtered, codes, nil
}

// autoCountry returns the country auto stands for: the country of the --city label when one is set and has a
// country, and otherwise the country speedtest.net reports for the user.
//
// Parameters:
//   - client: the client, which looks up the user's country.
//   - cfg: the settings, with the city.
//
// Returns:
//   - string: the upper-case country code.
//   - error: an error wrapping errUnknownCountry and the lookup's error when the city label is unknown or the user
//     lookup fails, or errUnknownCountry when no country is known.
func autoCountry(client *speedtest.Speedtest, cfg config.Config) (string, error) {
	if cfg.City != "" {
		location, err := speedtest.GetLocation(cfg.City)
		if err != nil {
			return "", fmt.Errorf("%w: --city %q: %w", errUnknownCountry, cfg.City, err)
		}

		// A label registered without a country, such as one built from coordinates, leaves the user's country.
		if location.CC != "" {
			return strings.ToUpper(location.CC), nil
		}
	}

	user, err := client.FetchUserInfo()
	if err != nil {
		return "", fmt.Errorf("%w: %w", errUnknownCountry, err)
	}

	if user.Country == "" {
		return "", errUnknownCountry
	}

	return strings.ToUpper(user.Country), nil
}
