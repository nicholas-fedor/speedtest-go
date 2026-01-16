package speedtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintCityList(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "print city list",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			PrintCityList() // Just ensure no panic
		})
	}
}

func TestGetLocation(t *testing.T) {
	type args struct {
		locationName string
	}

	tests := []struct {
		name    string
		args    args
		want    *Location
		wantErr bool
	}{
		{
			name:    "existing city",
			args:    args{locationName: "newyork"},
			want:    &Location{Name: "newyork", CC: "us", Lat: 40.7200876, Lon: -74.0220945},
			wantErr: false,
		},
		{
			name:    "non-existing city",
			args:    args{locationName: "Nonexistent"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetLocation(tt.args.locationName)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewLocation(t *testing.T) {
	type args struct {
		locationName string
		latitude     float64
		longitude    float64
	}

	tests := []struct {
		name string
		args args
		want *Location
	}{
		{
			name: "create location",
			args: args{
				locationName: "Test City",
				latitude:     40.7128,
				longitude:    -74.0060,
			},
			want: &Location{
				Name: "Test City",
				Lat:  40.7128,
				Lon:  -74.0060,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewLocation(tt.args.locationName, tt.args.latitude, tt.args.longitude)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseLocation(t *testing.T) {
	type args struct {
		locationName  string
		coordinateStr string
	}

	tests := []struct {
		name    string
		args    args
		want    *Location
		wantErr bool
	}{
		{
			name: "valid coordinates with name",
			args: args{
				locationName:  "Test",
				coordinateStr: "40.7128,-74.0060",
			},
			want: &Location{
				Name: "Custom-Test",
				Lat:  40.7128,
				Lon:  -74.0060,
			},
			wantErr: false,
		},
		{
			name: "valid coordinates without name",
			args: args{
				locationName:  "",
				coordinateStr: "51.5074,-0.1278",
			},
			want: &Location{
				Name: "Custom-Default",
				Lat:  51.5074,
				Lon:  -0.1278,
			},
			wantErr: false,
		},
		{
			name: "invalid coordinate format",
			args: args{
				locationName:  "Test",
				coordinateStr: "invalid",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "latitude out of range",
			args: args{
				locationName:  "Test",
				coordinateStr: "100,-74.0060",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseLocation(tt.args.locationName, tt.args.coordinateStr)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLocation_String(t *testing.T) {
	tests := []struct {
		name string
		l    *Location
		want string
	}{
		{
			name: "location string representation",
			l: &Location{
				Name: "Test City",
				Lat:  40.7128,
				Lon:  -74.0060,
			},
			want: "(Test City) [40.7128, -74.006]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.l.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_betweenRange(t *testing.T) {
	type args struct {
		inputStrValue string
		interval      float64
	}

	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "valid value within range",
			args: args{
				inputStrValue: "45.0",
				interval:      90.0,
			},
			want:    45.0,
			wantErr: false,
		},
		{
			name: "invalid non-numeric",
			args: args{
				inputStrValue: "invalid",
				interval:      90.0,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "value above range",
			args: args{
				inputStrValue: "100.0",
				interval:      90.0,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "value below range",
			args: args{
				inputStrValue: "-100.0",
				interval:      90.0,
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := betweenRange(tt.args.inputStrValue, tt.args.interval)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}
