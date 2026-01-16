package speedtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_MultiDownloadTestContext(t *testing.T) {
	type args struct {
		servers Servers
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{servers: Servers{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.MultiDownloadTestContext(ctx, tt.args.servers)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_MultiUploadTestContext(t *testing.T) {
	type args struct {
		servers Servers
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{servers: Servers{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.MultiUploadTestContext(ctx, tt.args.servers)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_DownloadTest(t *testing.T) {
	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.s.DownloadTest()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_DownloadTestContext(t *testing.T) {
	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.DownloadTestContext(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_downloadTestContext(t *testing.T) {
	type args struct {
		downloadRequest downloadFunc
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.downloadTestContext(ctx, tt.args.downloadRequest)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_UploadTest(t *testing.T) {
	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.s.UploadTest()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_UploadTestContext(t *testing.T) {
	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.UploadTestContext(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_uploadTestContext(t *testing.T) {
	type args struct {
		uploadRequest uploadFunc
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.uploadTestContext(ctx, tt.args.uploadRequest)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_downloadRequest(t *testing.T) {
	type args struct {
		s *Server
		w int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			args:    args{s: nil, w: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := downloadRequest(ctx, tt.args.s, tt.args.w)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_uploadRequest(t *testing.T) {
	type args struct {
		s *Server
		w int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			args:    args{s: nil, w: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := uploadRequest(ctx, tt.args.s, tt.args.w)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_PingTest(t *testing.T) {
	type args struct {
		callback func(latency time.Duration)
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{callback: func(_ time.Duration) {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.s.PingTest(tt.args.callback)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_PingTestContext(t *testing.T) {
	type args struct {
		callback func(latency time.Duration)
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{callback: func(_ time.Duration) {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.PingTestContext(ctx, tt.args.callback)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAll(t *testing.T) {
	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.s.TestAll()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_TCPPing(t *testing.T) {
	type args struct {
		echoTimes int
		echoFreq  time.Duration
		callback  func(latency time.Duration)
	}

	tests := []struct {
		name          string
		s             *Server
		args          args
		wantLatencies []int64
		wantErr       bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{echoTimes: 1, echoFreq: time.Second, callback: func(_ time.Duration) {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			gotLatencies, err := tt.s.TCPPing(
				ctx,
				tt.args.echoTimes,
				tt.args.echoFreq,
				tt.args.callback,
			)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLatencies, gotLatencies)
		})
	}
}

func TestServer_HTTPPing(t *testing.T) {
	type args struct {
		echoTimes int
		echoFreq  time.Duration
		callback  func(latency time.Duration)
	}

	tests := []struct {
		name          string
		s             *Server
		args          args
		wantLatencies []int64
		wantErr       bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{echoTimes: 1, echoFreq: time.Second, callback: func(_ time.Duration) {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			gotLatencies, err := tt.s.HTTPPing(
				ctx,
				tt.args.echoTimes,
				tt.args.echoFreq,
				tt.args.callback,
			)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLatencies, gotLatencies)
		})
	}
}

func TestServer_ICMPPing(t *testing.T) {
	type args struct {
		readTimeout time.Duration
		echoTimes   int
		echoFreq    time.Duration
		callback    func(latency time.Duration)
	}

	tests := []struct {
		name          string
		s             *Server
		args          args
		wantLatencies []int64
		wantErr       bool
	}{
		{
			name: "nil server",
			s:    nil,
			args: args{
				readTimeout: time.Second,
				echoTimes:   1,
				echoFreq:    time.Second,
				callback:    func(_ time.Duration) {},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			gotLatencies, err := tt.s.ICMPPing(
				ctx,
				tt.args.readTimeout,
				tt.args.echoTimes,
				tt.args.echoFreq,
				tt.args.callback,
			)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLatencies, gotLatencies)
		})
	}
}

func Test_checkSum(t *testing.T) {
	type args struct {
		data []byte
	}

	tests := []struct {
		name string
		args args
		want uint16
	}{
		{
			name: "checksum of empty data",
			args: args{data: []byte{}},
			want: 0xffff,
		},
		{
			name: "checksum of data",
			args: args{data: []byte{1, 2, 3, 4}},
			want: 0xfbf9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := checkSum(tt.args.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandardDeviation(t *testing.T) {
	type args struct {
		vector []int64
	}

	tests := []struct {
		name         string
		args         args
		wantMean     int64
		wantVariance int64
		wantStdDev   int64
		wantMin      int64
		wantMax      int64
	}{
		{
			name:         "standard deviation of single value",
			args:         args{vector: []int64{5}},
			wantMean:     5,
			wantVariance: 0,
			wantStdDev:   0,
			wantMin:      5,
			wantMax:      5,
		},
		{
			name:         "standard deviation of multiple values",
			args:         args{vector: []int64{1, 2, 3, 4, 5}},
			wantMean:     3,
			wantVariance: 2,
			wantStdDev:   1,
			wantMin:      1,
			wantMax:      5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMean, gotVariance, gotStdDev, gotMin, gotMax := StandardDeviation(tt.args.vector)
			assert.Equal(t, tt.wantMean, gotMean)
			assert.Equal(t, tt.wantVariance, gotVariance)
			assert.Equal(t, tt.wantStdDev, gotStdDev)
			assert.Equal(t, tt.wantMin, gotMin)
			assert.Equal(t, tt.wantMax, gotMax)
		})
	}
}
