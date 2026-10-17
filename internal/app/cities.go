package app

import (
	"fmt"
	"io"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// writeCities writes the predefined city labels, sorted by label, with their country codes and coordinates.
//
// Parameters:
//   - writer: where the list goes.
//
// Returns:
//   - error: the first write error.
func writeCities(writer io.Writer) error {
	_, err := fmt.Fprintln(writer, "Available city labels (case insensitive): ")
	if err != nil {
		return fmt.Errorf("failed to write the city list: %w", err)
	}

	_, err = fmt.Fprintln(writer, " CC\t\tCityLabel\tLocation")
	if err != nil {
		return fmt.Errorf("failed to write the city list: %w", err)
	}

	for _, city := range speedtest.Cities() {
		_, err = fmt.Fprintf(
			writer,
			"(%v)\t%20s\t[%v, %v]\n",
			city.CC,
			city.Name,
			city.Lat,
			city.Lon,
		)
		if err != nil {
			return fmt.Errorf("failed to write the city list: %w", err)
		}
	}

	return nil
}
