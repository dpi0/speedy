package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type SpeedTestResult struct {
	Type       string    `json:"type"`
	Timestamp  string    `json:"timestamp"`
	Ping       PingData  `json:"ping"`
	Download   Transfer  `json:"download"`
	Upload     Transfer  `json:"upload"`
	PacketLoss float64   `json:"packetLoss,omitempty"` // May not be present in all results
	ISP        string    `json:"isp"`
	Interface  Interface `json:"interface"`
	Server     Server    `json:"server"`
	Result     Result    `json:"result"`
}

type PingData struct {
	Jitter  float64 `json:"jitter"`
	Latency float64 `json:"latency"`
	Low     float64 `json:"low"`
	High    float64 `json:"high"`
}

type Transfer struct {
	Bandwidth int64       `json:"bandwidth"`
	Bytes     int64       `json:"bytes"`
	Elapsed   int         `json:"elapsed"`
	Latency   LatencyData `json:"latency"`
}

type LatencyData struct {
	IQM    float64 `json:"iqm"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Jitter float64 `json:"jitter"`
}

type Interface struct {
	InternalIP string `json:"internalIp"`
	Name       string `json:"name"`
	MacAddr    string `json:"macAddr"`
	IsVPN      bool   `json:"isVpn"`
	ExternalIP string `json:"externalIp"`
}

type Server struct {
	ID       int    `json:"id"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Country  string `json:"country"`
	IP       string `json:"ip"`
}

type Result struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Persisted bool   `json:"persisted"`
}

var (
	latestResult SpeedTestResult
	resultMutex  sync.RWMutex
	lastRunTime  time.Time
	runSuccess   bool
)

func runSpeedTest() error {
	log.Println("Starting speedtest...")
	cmd := exec.Command("./cli/speedtest", "--accept-license", "--accept-gdpr", "--format=json")
	output, err := cmd.Output()

	if err != nil {
		log.Printf("Speedtest failed: %v", err)
		runSuccess = false
		return err
	}

	resultMutex.Lock()
	defer resultMutex.Unlock()

	err = json.Unmarshal(output, &latestResult)
	if err != nil {
		log.Printf("Failed to parse speedtest output: %v", err)
		runSuccess = false
		return err
	}

	lastRunTime = time.Now()
	runSuccess = true
	log.Println("Speedtest completed successfully")
	return nil
}

func generateMetrics() string {
	resultMutex.RLock()
	defer resultMutex.RUnlock()

	var sb strings.Builder

	sb.WriteString("# HELP speedtest_ping_latency_ms Latency in milliseconds\n")
	sb.WriteString("# TYPE speedtest_ping_latency_ms gauge\n")
	sb.WriteString(fmt.Sprintf("speedtest_ping_latency_ms %.3f\n", latestResult.Ping.Latency))

	sb.WriteString("# HELP speedtest_ping_jitter_ms Jitter in milliseconds\n")
	sb.WriteString("# TYPE speedtest_ping_jitter_ms gauge\n")
	sb.WriteString(fmt.Sprintf("speedtest_ping_jitter_ms %.3f\n", latestResult.Ping.Jitter))

	sb.WriteString("# HELP speedtest_download_bandwidth_mbps Download bandwidth in Mbps\n")
	sb.WriteString("# TYPE speedtest_download_bandwidth_mbps gauge\n")
	sb.WriteString(
		fmt.Sprintf(
			"speedtest_download_bandwidth_mbps %.3f\n",
			float64(latestResult.Download.Bandwidth)/125000.0,
		),
	)

	sb.WriteString("# HELP speedtest_upload_bandwidth_mbps Upload bandwidth in Mbps\n")
	sb.WriteString("# TYPE speedtest_upload_bandwidth_mbps gauge\n")
	sb.WriteString(
		fmt.Sprintf(
			"speedtest_upload_bandwidth_mbps %.3f\n",
			float64(latestResult.Upload.Bandwidth)/125000.0,
		),
	)

	sb.WriteString("# HELP speedtest_download_bytes_total Total bytes downloaded\n")
	sb.WriteString("# TYPE speedtest_download_bytes_total counter\n")
	sb.WriteString(fmt.Sprintf("speedtest_download_bytes_total %d\n", latestResult.Download.Bytes))

	sb.WriteString("# HELP speedtest_upload_bytes_total Total bytes uploaded\n")
	sb.WriteString("# TYPE speedtest_upload_bytes_total counter\n")
	sb.WriteString(fmt.Sprintf("speedtest_upload_bytes_total %d\n", latestResult.Upload.Bytes))

	// Packet loss might not be present in all results
	sb.WriteString("# HELP speedtest_packet_loss_percent Packet loss percentage\n")
	sb.WriteString("# TYPE speedtest_packet_loss_percent gauge\n")
	packetLoss := 0.0 // Default to 0 if not present
	if latestResult.PacketLoss > 0 ||
		strings.Contains(fmt.Sprintf("%v", latestResult), "packetLoss") {
		packetLoss = latestResult.PacketLoss
	}
	sb.WriteString(fmt.Sprintf("speedtest_packet_loss_percent %.3f\n", packetLoss))

	sb.WriteString(
		"# HELP speedtest_last_run_timestamp_seconds Timestamp of the last speedtest run\n",
	)
	sb.WriteString("# TYPE speedtest_last_run_timestamp_seconds gauge\n")
	sb.WriteString(fmt.Sprintf("speedtest_last_run_timestamp_seconds %d\n", lastRunTime.Unix()))

	sb.WriteString("# HELP speedtest_run_success Whether the last speedtest run was successful\n")
	sb.WriteString("# TYPE speedtest_run_success gauge\n")
	successValue := 0
	if runSuccess {
		successValue = 1
	}
	sb.WriteString(fmt.Sprintf("speedtest_run_success %d\n", successValue))

	sb.WriteString(
		"# HELP speedtest_download_latency_iqm_ms Download latency IQM in milliseconds\n",
	)
	sb.WriteString("# TYPE speedtest_download_latency_iqm_ms gauge\n")
	sb.WriteString(
		fmt.Sprintf("speedtest_download_latency_iqm_ms %.3f\n", latestResult.Download.Latency.IQM),
	)

	sb.WriteString(
		"# HELP speedtest_download_latency_low_ms Download latency low in milliseconds\n",
	)
	sb.WriteString("# TYPE speedtest_download_latency_low_ms gauge\n")
	sb.WriteString(
		fmt.Sprintf("speedtest_download_latency_low_ms %.3f\n", latestResult.Download.Latency.Low),
	)

	sb.WriteString(
		"# HELP speedtest_download_latency_high_ms Download latency high in milliseconds\n",
	)
	sb.WriteString("# TYPE speedtest_download_latency_high_ms gauge\n")
	sb.WriteString(
		fmt.Sprintf(
			"speedtest_download_latency_high_ms %.3f\n",
			latestResult.Download.Latency.High,
		),
	)

	sb.WriteString(
		"# HELP speedtest_download_latency_jitter_ms Download latency jitter in milliseconds\n",
	)
	sb.WriteString("# TYPE speedtest_download_latency_jitter_ms gauge\n")
	sb.WriteString(
		fmt.Sprintf(
			"speedtest_download_latency_jitter_ms %.3f\n",
			latestResult.Download.Latency.Jitter,
		),
	)

	sb.WriteString("# HELP speedtest_upload_latency_iqm_ms Upload latency IQM in milliseconds\n")
	sb.WriteString("# TYPE speedtest_upload_latency_iqm_ms gauge\n")
	sb.WriteString(
		fmt.Sprintf("speedtest_upload_latency_iqm_ms %.3f\n", latestResult.Upload.Latency.IQM),
	)

	sb.WriteString("# HELP speedtest_upload_latency_low_ms Upload latency low in milliseconds\n")
	sb.WriteString("# TYPE speedtest_upload_latency_low_ms gauge\n")
	sb.WriteString(
		fmt.Sprintf("speedtest_upload_latency_low_ms %.3f\n", latestResult.Upload.Latency.Low),
	)

	sb.WriteString("# HELP speedtest_upload_latency_high_ms Upload latency high in milliseconds\n")
	sb.WriteString("# TYPE speedtest_upload_latency_high_ms gauge\n")
	sb.WriteString(
		fmt.Sprintf("speedtest_upload_latency_high_ms %.3f\n", latestResult.Upload.Latency.High),
	)

	sb.WriteString(
		"# HELP speedtest_upload_latency_jitter_ms Upload latency jitter in milliseconds\n",
	)
	sb.WriteString("# TYPE speedtest_upload_latency_jitter_ms gauge\n")
	sb.WriteString(
		fmt.Sprintf(
			"speedtest_upload_latency_jitter_ms %.3f\n",
			latestResult.Upload.Latency.Jitter,
		),
	)

	return sb.String()
}

func main() {
	// Run speedtest on startup
	log.Println("Running initial speedtest...")
	err := runSpeedTest()
	if err != nil {
		log.Printf("Initial speedtest failed: %v", err)
	}

	ticker := time.NewTicker(2 * time.Minute)
	go func() {
		for range ticker.C {
			log.Println("Running scheduled speedtest...")
			runSpeedTest()
		}
	}()

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(generateMetrics()))
	})

	log.Println("Server started at http://localhost:8080")
	log.Println("Access /metrics endpoint for Prometheus metrics")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
