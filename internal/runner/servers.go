// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// selection is what a run's lookups found.
type selection struct {
	// user is the caller's details.
	user *speedtest.User
	// targets are the servers to test.
	targets speedtest.Servers
	// pool is the servers a multi-server test spreads across.
	pool speedtest.Servers
}

// userInfoTimeout bounds the lookup of the caller's details, which the reports wait for.
const userInfoTimeout = 30 * time.Second

// Errors of the server selection.
var (
	// errNoServersInCountries is returned when the country filter leaves no servers.
	errNoServersInCountries = errors.New("no servers in the selected countries")
	// errUnknownCountry is returned when the country that [CountryAuto] stands for is unknown.
	errUnknownCountry = errors.New("cannot determine your country for --filter-cc auto")
)

// ListServers fetches the servers that the list command shows, filtered by the country codes of the settings.
//
// Parameters:
//   - ctx: ends the lookups.
//   - tester: looks up the servers, and the caller's country when [CountryAuto] needs it.
//   - settings: the country codes and the city.
//
// Returns:
//   - speedtest.Servers: the servers, in their listed order.
//   - []string: the country codes used, with [CountryAuto] resolved, or nil without a filter.
//   - error: the error of a lookup, or of a filter that resolves to no country or leaves no servers.
func ListServers(
	ctx context.Context,
	tester Tester,
	settings Settings,
) (speedtest.Servers, []string, error) {
	servers, err := tester.FetchServers(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	if len(settings.CountryCodes) == 0 {
		return servers, nil, nil
	}

	var user *speedtest.User

	if settings.City == "" && slices.Contains(settings.CountryCodes, CountryAuto) {
		user, err = tester.FetchUserInfo(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %w", errUnknownCountry, err)
		}
	}

	codes, err := countryCodes(settings, user)
	if err != nil {
		return nil, nil, err
	}

	filtered, err := filterByCountry(servers, codes)
	if err != nil {
		return nil, codes, err
	}

	return filtered, codes, nil
}

// lookup looks up the caller and the servers at the same time, then selects the servers to test.
//
// Parameters:
//   - ctx: ends the lookups.
//
// Returns:
//   - selection: the caller, the targets, and the pool.
//   - error: the error of a lookup or of the country filter.
func (r *Runner) lookup(ctx context.Context) (selection, error) {
	r.report(LookupStarted{})

	var (
		user     *speedtest.User
		userErr  error
		userDone sync.WaitGroup
	)

	userDone.Go(func() {
		userCtx, cancel := context.WithTimeout(ctx, userInfoTimeout)
		defer cancel()

		user, userErr = r.tester.FetchUserInfo(userCtx)
		if userErr == nil {
			r.report(UserFound{User: user})
		}
	})

	pool, source, err := r.findServers(ctx)

	userDone.Wait()

	if userErr != nil {
		return selection{}, fmt.Errorf("failed to retrieve user information: %w", userErr)
	}

	if err != nil {
		return selection{}, fmt.Errorf("failed to retrieve servers: %w", err)
	}

	if source != SourceListed {
		r.report(ServersFound{Codes: nil, Count: len(pool), Source: source})

		return selection{user: user, targets: pool, pool: pool}, nil
	}

	var codes []string

	if len(r.settings.CountryCodes) > 0 {
		codes, err = countryCodes(r.settings, user)
		if err != nil {
			return selection{}, err
		}

		pool, err = filterByCountry(pool, codes)
		if err != nil {
			return selection{}, err
		}
	}

	// Without server IDs, FindServer picks the server with the lowest latency.
	targets, err := pool.FindServer(nil)
	if err != nil {
		return selection{}, fmt.Errorf("failed to select a server: %w", err)
	}

	r.report(ServersFound{Codes: codes, Count: len(pool), Source: source})

	return selection{user: user, targets: targets, pool: pool}, nil
}

// findServers returns the servers the settings name: a custom server, the servers with the given IDs, or the
// server list.
//
// Parameters:
//   - ctx: ends the lookups.
//
// Returns:
//   - speedtest.Servers: the servers.
//   - ServerSource: where they came from.
//   - error: the lookup's error.
func (r *Runner) findServers(ctx context.Context) (speedtest.Servers, ServerSource, error) {
	switch {
	case r.settings.CustomURL != "":
		server, err := r.tester.CustomServer(r.settings.CustomURL)
		if err != nil {
			return nil, SourceCustom, fmt.Errorf("invalid custom server URL: %w", err)
		}

		return speedtest.Servers{server}, SourceCustom, nil
	case len(r.settings.ServerIDs) > 0:
		servers, err := fetchServersByID(ctx, r.tester, r.settings.ServerIDs)

		return servers, SourceSpecified, err
	default:
		servers, err := r.tester.FetchServers(ctx)
		if err != nil {
			return nil, SourceListed, fmt.Errorf("failed to fetch the server list: %w", err)
		}

		return servers, SourceListed, nil
	}
}

// fetchServersByID looks up each server ID concurrently.
//
// Parameters:
//   - ctx: ends the lookups.
//   - tester: performs the lookups.
//   - ids: the server IDs.
//
// Returns:
//   - speedtest.Servers: the servers in the order of ids.
//   - error: an error wrapping [speedtest.ErrServerNotFound] that lists every unknown ID, joined with any other
//     lookup errors.
func fetchServersByID(ctx context.Context, tester Tester, ids []int) (speedtest.Servers, error) {
	found := make(speedtest.Servers, len(ids))
	errs := make([]error, len(ids))

	var wg sync.WaitGroup

	for index, id := range ids {
		wg.Go(func() {
			found[index], errs[index] = tester.FetchServerByID(ctx, strconv.Itoa(id))
		})
	}

	wg.Wait()

	var (
		missing []string
		failed  []error
	)

	for index, serverID := range ids {
		switch err := errs[index]; {
		case err == nil:
		case errors.Is(err, speedtest.ErrServerNotFound):
			missing = append(missing, strconv.Itoa(serverID))
		default:
			failed = append(failed, fmt.Errorf("server %d: %w", serverID, err))
		}
	}

	if len(missing) > 0 {
		failed = append(
			[]error{
				fmt.Errorf(
					"%w for --server %s",
					speedtest.ErrServerNotFound,
					strings.Join(missing, ", "),
				),
			},
			failed...,
		)
	}

	if len(failed) > 0 {
		return nil, errors.Join(failed...)
	}

	return found, nil
}

// countryCodes returns the country codes of the settings, with [CountryAuto] replaced by the country it stands for.
//
// Parameters:
//   - settings: the country codes and the city.
//   - user: the caller's details, or nil when they were not looked up.
//
// Returns:
//   - []string: the upper-case country codes.
//   - error: an error wrapping errUnknownCountry when [CountryAuto] stands for no known country.
func countryCodes(settings Settings, user *speedtest.User) ([]string, error) {
	codes := make([]string, 0, len(settings.CountryCodes))

	for _, code := range settings.CountryCodes {
		if code == CountryAuto {
			country, err := autoCountry(settings.City, user)
			if err != nil {
				return nil, err
			}

			code = country
		}

		codes = append(codes, code)
	}

	return codes, nil
}

// autoCountry returns the country that [CountryAuto] stands for: the country of the city label when one is set,
// and otherwise the country speedtest.net reports for the caller.
//
// Parameters:
//   - city: the predefined city label, or empty.
//   - user: the caller's details, or nil when they were not looked up.
//
// Returns:
//   - string: the upper-case country code.
//   - error: an error wrapping errUnknownCountry when the city label is unknown or no country is known.
func autoCountry(city string, user *speedtest.User) (string, error) {
	if city != "" {
		location, err := speedtest.GetLocation(city)
		if err != nil {
			return "", fmt.Errorf("%w: --city %q: %w", errUnknownCountry, city, err)
		}

		return strings.ToUpper(location.CC), nil
	}

	if user == nil || user.Country == "" {
		return "", errUnknownCountry
	}

	return strings.ToUpper(user.Country), nil
}

// filterByCountry keeps the servers in the given countries.
//
// Parameters:
//   - servers: the servers to filter.
//   - codes: the upper-case country codes.
//
// Returns:
//   - speedtest.Servers: the servers in those countries, in their original order.
//   - error: an error wrapping errNoServersInCountries when no server matches.
func filterByCountry(servers speedtest.Servers, codes []string) (speedtest.Servers, error) {
	filtered := servers.FilterCountry(codes...)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("%w: %s", errNoServersInCountries, strings.Join(codes, ", "))
	}

	return filtered, nil
}
