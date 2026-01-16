package speedtest

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/showwin/speedtest-go/speedtest/transport"
)

const (
	speedTestServersURL            = "https://www.speedtest.net/api/js/servers"
	speedTestServersAlternativeURL = "https://www.speedtest.net/speedtest-servers-static.php"
	speedTestServersAdvanced       = "https://www.speedtest.net/api/ios-config.php"
)

type payloadType int

const (
	typeJSONPayload payloadType = iota
	typeXMLPayload
)

var (
	// ErrServerNotFound is returned when no server is available or found.
	ErrServerNotFound = errors.New("no server available or found")

	errSpeedtestClientNil = errors.New("speedtest client is nil")
	errHostEmpty          = errors.New("host cannot be empty")
	errPayloadDecode      = errors.New("response payload decoding not implemented")
)

// Server information.
type Server struct {
	URL          string          `json:"url"          xml:"url,attr"`
	Lat          string          `json:"lat"          xml:"lat,attr"`
	Lon          string          `json:"lon"          xml:"lon,attr"`
	Name         string          `json:"name"         xml:"name,attr"`
	Country      string          `json:"country"      xml:"country,attr"`
	Sponsor      string          `json:"sponsor"      xml:"sponsor,attr"`
	ID           string          `json:"id"           xml:"id,attr"`
	Host         string          `json:"host"         xml:"host,attr"`
	Distance     float64         `json:"distance"     xml:"-"`
	Latency      time.Duration   `json:"latency"      xml:"-"`
	MaxLatency   time.Duration   `json:"maxLatency"   xml:"-"`
	MinLatency   time.Duration   `json:"minLatency"   xml:"-"`
	Jitter       time.Duration   `json:"jitter"       xml:"-"`
	DLSpeed      ByteRate        `json:"dlSpeed"      xml:"-"`
	ULSpeed      ByteRate        `json:"ulSpeed"      xml:"-"`
	TestDuration TestDuration    `json:"testDuration" xml:"-"`
	PacketLoss   transport.PLoss `json:"packetLoss"   xml:"-"`
	Context      *Speedtest      `json:"-"            xml:"-"`
}

// TestDuration holds the duration of different test phases.
type TestDuration struct {
	Ping     *time.Duration `json:"ping"`
	Download *time.Duration `json:"download"`
	Upload   *time.Duration `json:"upload"`
	Total    *time.Duration `json:"total"`
}

// CustomServer use defaultClient, given a URL string, return a new Server object, with as much
// filled in as we can.
func CustomServer(host string) (*Server, error) {
	return defaultClient.CustomServer(host)
}

// CustomServer given a URL string, return a new Server object, with as much
// filled in as we can.
func (s *Speedtest) CustomServer(host string) (*Server, error) {
	if s == nil {
		return nil, errSpeedtestClientNil
	}

	if host == "" {
		return nil, errHostEmpty
	}

	parsedURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host URL: %w", err)
	}

	parsedURL.Path = "/speedtest/upload.php"
	parseHost := parsedURL.String()

	return &Server{
		ID:      "Custom",
		Lat:     "?",
		Lon:     "?",
		Country: "?",
		URL:     parseHost,
		Name:    parsedURL.Host,
		Host:    parsedURL.Host,
		Sponsor: "?",
		Context: s,
	}, nil
}

// ServerList list of Server
// Users(Client) also exists with @param speedTestServersAdvanced.
type ServerList struct {
	XMLName xml.Name  `xml:"settings"`
	Servers []*Server `xml:"servers>server" json:"servers"`
	Users   []User    `xml:"client"         json:"users"`
}

// Servers for sorting servers.
type Servers []*Server

// ByDistance for sorting servers.
type ByDistance struct {
	Servers
}

// Available returns servers that have valid latency measurements.
func (servers Servers) Available() *Servers {
	retServer := Servers{}

	for _, server := range servers {
		if server.Latency != PingTimeout {
			retServer = append(retServer, server)
		}
	}

	for i := range len(retServer) - 1 {
		for j := range len(retServer) - i - 1 {
			if retServer[j].Latency > retServer[j+1].Latency {
				retServer[j], retServer[j+1] = retServer[j+1], retServer[j]
			}
		}
	}

	return &retServer
}

// Len finds length of servers. For sorting servers.
func (servers Servers) Len() int {
	return len(servers)
}

// Swap swaps i-th and j-th. For sorting servers.
func (servers Servers) Swap(i, j int) {
	servers[i], servers[j] = servers[j], servers[i]
}

// Hosts return hosts of servers.
func (servers Servers) Hosts() []string {
	if len(servers) == 0 {
		return nil
	}

	retServer := make([]string, 0, len(servers))
	for _, server := range servers {
		retServer = append(retServer, server.Host)
	}

	return retServer
}

// Less compares the distance. For sorting servers.
func (b ByDistance) Less(i, j int) bool {
	return b.Servers[i].Distance < b.Servers[j].Distance
}

// FetchServerByID retrieves a server by given serverID.
func (s *Speedtest) FetchServerByID(serverID string) (*Server, error) {
	return s.FetchServerByIDContext(context.Background(), serverID)
}

// FetchServerByID retrieves a server by given serverID.
func FetchServerByID(serverID string) (*Server, error) {
	return defaultClient.FetchServerByID(serverID)
}

// FetchServerByIDContext retrieves a server by given serverID, observing the given context.
func (s *Speedtest) FetchServerByIDContext(ctx context.Context, serverID string) (*Server, error) {
	if s == nil {
		return nil, errSpeedtestClientNil
	}

	parsedURL, err := url.Parse(speedTestServersAdvanced)
	if err != nil {
		return nil, fmt.Errorf("failed to parse speed test servers advanced URL: %w", err)
	}

	query := parsedURL.Query()
	query.Set(strings.ToLower("serverID"), serverID)
	parsedURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := s.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var list ServerList

	decoder := xml.NewDecoder(resp.Body)

	err = decoder.Decode(&list)
	if err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	for serverIndex := range list.Servers {
		if list.Servers[serverIndex].ID == serverID {
			list.Servers[serverIndex].Context = s
			if len(list.Users) > 0 {
				sLat, _ := strconv.ParseFloat(list.Servers[serverIndex].Lat, 64)
				sLon, _ := strconv.ParseFloat(list.Servers[serverIndex].Lon, 64)
				uLat, _ := strconv.ParseFloat(list.Users[0].Lat, 64)
				uLon, _ := strconv.ParseFloat(list.Users[0].Lon, 64)
				list.Servers[serverIndex].Distance = distance(sLat, sLon, uLat, uLon)
			}

			return list.Servers[serverIndex], nil
		}
	}

	return nil, ErrServerNotFound
}

// pingServers pings all servers to measure latency.
func pingServers(ctx context.Context, servers Servers, pingMode Proto) {
	var waitGroup sync.WaitGroup

	pCtx, cancelFunc := context.WithTimeout(ctx, time.Second*4)

	dbg.Println("Echo each server...")

	for _, server := range servers {
		waitGroup.Add(1)

		go func(serverPtr *Server) {
			var (
				latency []int64
				errPing error
			)

			switch pingMode {
			case TCP:
				latency, errPing = serverPtr.TCPPing(pCtx, 1, time.Millisecond, nil)
			case ICMP:
				latency, errPing = serverPtr.ICMPPing(pCtx, 4*time.Second, 1, time.Millisecond, nil)
			case HTTP:
				latency, errPing = serverPtr.HTTPPing(pCtx, 1, time.Millisecond, nil)
			default:
				latency, errPing = serverPtr.HTTPPing(pCtx, 1, time.Millisecond, nil)
			}

			if errPing != nil || len(latency) < 1 {
				serverPtr.Latency = PingTimeout
			} else {
				serverPtr.Latency = time.Duration(latency[0]) * time.Nanosecond
			}

			waitGroup.Done()
		}(server)
	}

	waitGroup.Wait()
	cancelFunc()
}

// FetchServers retrieves a list of available servers.
func (s *Speedtest) FetchServers() (Servers, error) {
	return s.FetchServerListContext(context.Background())
}

// FetchServers retrieves a list of available servers.
func FetchServers() (Servers, error) {
	return defaultClient.FetchServers()
}

// buildServerListURL constructs the URL for fetching server list.
func (s *Speedtest) buildServerListURL() (*url.URL, error) {
	parsedURL, err := url.Parse(speedTestServersURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse speed test servers URL: %w", err)
	}

	query := parsedURL.Query()
	if len(s.config.Keyword) > 0 {
		query.Set("search", s.config.Keyword)
	}

	if s.config.Location != nil {
		query.Set("lat", strconv.FormatFloat(s.config.Location.Lat, 'f', -1, 64))
		query.Set("lon", strconv.FormatFloat(s.config.Location.Lon, 'f', -1, 64))
	}

	parsedURL.RawQuery = query.Encode()
	dbg.Printf("Retrieving servers: %s\n", parsedURL.String())

	return parsedURL, nil
}

// fetchServerListResponse performs the HTTP request for server list, handling fallback to alternative URL.
func (s *Speedtest) fetchServerListResponse(
	ctx context.Context,
	reqURL *url.URL,
) (*http.Response, payloadType, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := s.doer.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	payloadType := typeJSONPayload

	if resp.ContentLength == 0 {
		_ = resp.Body.Close()

		req, err = http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			speedTestServersAlternativeURL,
			nil,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create alternative HTTP request: %w", err)
		}

		resp, err = s.doer.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to perform alternative HTTP request: %w", err)
		}

		payloadType = typeXMLPayload
	}

	return resp, payloadType, nil
}

// decodeServerList decodes the server list from the HTTP response based on payload type.
func (s *Speedtest) decodeServerList(resp *http.Response, pType payloadType) (Servers, error) {
	var servers Servers

	switch pType {
	case typeJSONPayload:
		decoder := json.NewDecoder(resp.Body)

		err := decoder.Decode(&servers)
		if err != nil {
			return servers, fmt.Errorf("failed to decode JSON response: %w", err)
		}
	case typeXMLPayload:
		var list ServerList

		decoder := xml.NewDecoder(resp.Body)

		err := decoder.Decode(&list)
		if err != nil {
			return servers, fmt.Errorf("failed to decode XML response: %w", err)
		}

		servers = list.Servers
	default:
		return servers, errPayloadDecode
	}

	dbg.Printf("Servers Num: %d\n", len(servers))

	return servers, nil
}

// FetchServerListContext retrieves a list of available servers, observing the given context.
func (s *Speedtest) FetchServerListContext(ctx context.Context) (Servers, error) {
	if s == nil {
		return Servers{}, errSpeedtestClientNil
	}

	reqURL, err := s.buildServerListURL()
	if err != nil {
		return Servers{}, err
	}

	resp, payloadType, err := s.fetchServerListResponse(ctx, reqURL)
	if err != nil {
		return Servers{}, err
	}

	defer func() { _ = resp.Body.Close() }()

	servers, err := s.decodeServerList(resp, payloadType)
	if err != nil {
		return servers, err
	}

	// set context for servers
	for _, server := range servers {
		server.Context = s
	}

	// ping servers
	pingServers(ctx, servers, s.config.PingMode)

	// Calculate distance
	// If we don't call FetchUserInfo() before FetchServers(),
	// we don't calculate the distance, instead we use the
	// remote computing distance provided by Ookla as default.
	if s.User != nil {
		for _, server := range servers {
			sLat, _ := strconv.ParseFloat(server.Lat, 64)
			sLon, _ := strconv.ParseFloat(server.Lon, 64)
			uLat, _ := strconv.ParseFloat(s.User.Lat, 64)
			uLon, _ := strconv.ParseFloat(s.User.Lon, 64)
			server.Distance = distance(sLat, sLon, uLat, uLon)
		}
	}

	// Sort by distance
	sort.Sort(ByDistance{servers})

	if len(servers) == 0 {
		return servers, ErrServerNotFound
	}

	return servers, nil
}

// FetchServerListContext retrieves a list of available servers, observing the given context.
func FetchServerListContext(ctx context.Context) (Servers, error) {
	return defaultClient.FetchServerListContext(ctx)
}

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	radius := 6378.137

	phi1 := lat1 * math.Pi / 180.0
	phi2 := lat2 * math.Pi / 180.0

	deltaPhiHalf := (lat1 - lat2) * math.Pi / 360.0
	deltaLambdaHalf := (lon1 - lon2) * math.Pi / 360.0
	sinePhiHalf2 := math.Sin(
		deltaPhiHalf,
	)*math.Sin(
		deltaPhiHalf,
	) + math.Cos(
		phi1,
	)*math.Cos(
		phi2,
	)*math.Sin(
		deltaLambdaHalf,
	)*math.Sin(
		deltaLambdaHalf,
	) // phi half-angle sine ^ 2
	delta := 2 * math.Atan2(
		math.Sqrt(sinePhiHalf2),
		math.Sqrt(1-sinePhiHalf2),
	) // 2 arc sine

	return radius * delta // r * delta
}

// FindServer finds server by serverID in given server list.
// If the id is not found in the given list, return the server with the lowest latency.
func (servers Servers) FindServer(serverID []int) (Servers, error) {
	retServer := Servers{}

	if len(servers) == 0 {
		return retServer, ErrServerNotFound
	}

	for _, sid := range serverID {
		for _, s := range servers {
			id, _ := strconv.Atoi(s.ID)
			if sid == id {
				retServer = append(retServer, s)

				break
			}
		}
	}

	if len(retServer) == 0 {
		// choose the lowest latency server
		var (
			minLatency     int64 = math.MaxInt64
			minServerIndex int
		)

		for index, server := range servers {
			if server.Latency <= 0 {
				continue
			}

			if minLatency > server.Latency.Milliseconds() {
				minLatency = server.Latency.Milliseconds()
				minServerIndex = index
			}
		}

		retServer = append(retServer, servers[minServerIndex])
	}

	return retServer, nil
}

// String representation of ServerList.
func (servers ServerList) String() string {
	slr := ""

	var slrSb409 strings.Builder
	for _, server := range servers.Servers {
		slrSb409.WriteString(server.String())
	}

	slr += slrSb409.String()

	return slr
}

// String representation of Servers.
func (servers Servers) String() string {
	slr := ""

	var slrSb418 strings.Builder
	for _, server := range servers {
		slrSb418.WriteString(server.String())
	}

	slr += slrSb418.String()

	return slr
}

// String representation of Server.
func (s *Server) String() string {
	if s == nil {
		return "<nil server>"
	}

	if s.Sponsor == "?" {
		return fmt.Sprintf("[%4s] %s", s.ID, s.Name)
	}

	if len(s.Country) == 0 {
		return fmt.Sprintf("[%4s] %.2fkm %s by %s", s.ID, s.Distance, s.Name, s.Sponsor)
	}

	return fmt.Sprintf("[%4s] %.2fkm %s (%s) by %s", s.ID, s.Distance, s.Name, s.Country, s.Sponsor)
}

// CheckResultValid checks that results are logical given UL and DL speeds.
func (s *Server) CheckResultValid() bool {
	if s == nil {
		return false
	}

	return s.DLSpeed*100 >= s.ULSpeed && s.DLSpeed <= s.ULSpeed*100
}

func (s *Server) testDurationTotalCount() {
	if s == nil {
		return
	}

	total := s.getNotNullValue(s.TestDuration.Ping) +
		s.getNotNullValue(s.TestDuration.Download) +
		s.getNotNullValue(s.TestDuration.Upload)

	s.TestDuration.Total = &total
}

func (s *Server) getNotNullValue(time *time.Duration) time.Duration {
	if time == nil {
		return 0
	}

	return *time
}
