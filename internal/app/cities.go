package app

import "github.com/nicholas-fedor/speedtest-go/speedtest"

// ShowCities displays the list of predefined cities.
func ShowCities() error {
	speedtest.PrintCityList()

	return nil
}
