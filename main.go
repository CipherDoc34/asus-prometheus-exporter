package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"encoding/base64"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const COOKIE_EXPIRY_FILE_DIR string = "CookieExpiryTime.json"
const COOKIE_FILE_DIR string = "cookie.json"
const COOKIE_EXPIRY_DURATION time.Duration = 2 * time.Hour
const NAMESPACE = "asus"
var (
	hostPort = flag.Int("prom.port", 8000, "port to expose prometeus metrics")
	UserName = flag.String("uname", "nil", "login username for asus router")
	Password = flag.String("passwd", "nil", "login password for asus router")
	AUTHENTICATION string
)

// unused
type hookDetails struct {
	Caller         string
	SubstringStart int
}

// unused
type allHooks struct {
	Name    string
	Details hookDetails
}

// stores asus_token
type asusCookie struct {
	AsusToken string `json:"asus_token"`
}

// stores the time in a broken storable format
type timeFormatted struct {
	Year, Month, Day, Hour, Minute, Second, Nsec int
}

// nothing
type testInterface struct {
	Name, Test string
	Cookie     asusCookie
}

func SetAuth(uname string, passwd string) {
	data := fmt.Sprintf("%s:%s", uname, passwd)
	enc := base64.StdEncoding.EncodeToString([]byte(data))
	AUTHENTICATION = fmt.Sprintf("login_authorization=%s", enc)
}

// returns
//
//	2 if current time > what is in the file
//	1 if current time == what is in the file
//	0 if current time < what is in the file
func checkTime(fileName string, currentTime time.Time) (int, error) {
	// file := fmt.Sprintf("%s.json", fileName)
	// data, err := os.ReadFile(file)
	data, err := os.ReadFile(fileName)
	if err != nil {
		return -2, fmt.Errorf("Read File Error: %s", err)
	}
	var previousTimeFormatted timeFormatted
	err = json.Unmarshal(data, &previousTimeFormatted)
	if err != nil {
		return -2, fmt.Errorf("json Unmarshal Error: %s", err)
	}
	previousTime := time.Date(
		previousTimeFormatted.Year,
		time.Month(previousTimeFormatted.Month),
		previousTimeFormatted.Day,
		previousTimeFormatted.Hour,
		previousTimeFormatted.Minute,
		previousTimeFormatted.Second,
		previousTimeFormatted.Nsec,
		time.Local,
	)
	return currentTime.Compare(previousTime) + 1, nil
}

// Can write a JSON of any structure
func writeJson(j interface{}, fileName string) error {
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return fmt.Errorf("Marshal Indent Error: %s", err)
	}

	// file := fmt.Sprintf("%s.json", fileName)
	// err = os.WriteFile(file, data, 0666)
	err = os.WriteFile(fileName, data, 0666)
	if err != nil {
		return fmt.Errorf("Write File Error: %s", err)
	}

	return nil
}

// Reads Json file only formatted as asusCookie
func readJson(fileName string) (asusCookie, error) {
	// file := fmt.Sprintf("%s.json", fileName)
	// data, err := os.ReadFile(file)
	data, err := os.ReadFile(fileName)
	if err != nil {
		return asusCookie{}, fmt.Errorf("Read File Error: %s", err)
	}
	var j asusCookie
	err = json.Unmarshal(data, &j)
	if err != nil {
		return asusCookie{}, fmt.Errorf("Read File Error: %s", err)
	}
	return j, nil
}

// gets new cookie from the router
func getCookieFromRouter() (asusCookie, error) {
	var MainAsusCookie asusCookie
	targetHost := "router.asus.com"
	targetPath := "/login.cgi"
	loginAuth := AUTHENTICATION
	method := "POST"

	uri := fmt.Sprintf("http://%s%s", targetHost, targetPath)

	payload := strings.NewReader(loginAuth)

	client := &http.Client{}

	req, err := http.NewRequest(method, uri, payload)
	if err != nil {
		return MainAsusCookie, fmt.Errorf("Error Setting Request: %s", err)
	}

	req.Header.Add("User-Agent", "asusrouter-Android-DUTUtil-1.0.0.245")
	req.Header.Add("Content-Type", "appliction/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return MainAsusCookie, fmt.Errorf("Error client do: %s", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return MainAsusCookie, fmt.Errorf("Error ReadAll: %s", err)
	}

	var b interface{}

	err = json.Unmarshal([]byte(body), &b)
	if err != nil {
		return MainAsusCookie, fmt.Errorf("Json unmarshal error: %s", err)
	}

	asusMap := b.(map[string]interface{})
	for k, v := range asusMap {
		switch k {
		case "asus_token":
			MainAsusCookie.AsusToken = v.(string)
		default:
			continue
		}
	}
	return MainAsusCookie, nil
}

// should be unused
func testgetcookie() (asusCookie, error) {
	a := asusCookie{AsusToken: "token"}
	err := writeJson(a, COOKIE_FILE_DIR)
	if err != nil {
		return a, fmt.Errorf("Error Writing Json: %s", err)
	}
	log.Println(a)

	testing, err := readJson(COOKIE_FILE_DIR)
	if err != nil {
		return a, fmt.Errorf("Error Reading Json: %s", err)
	}
	log.Println(testing)

	return testing, nil
}

// writes the new cookie to the JSON
func writeNewCookie() error {
	cookie, err := getCookieFromRouter()
	if err != nil {
		return fmt.Errorf("getCookieFromRouter Error: %s", err)
	}
	if err = writeJson(cookie, COOKIE_FILE_DIR); err != nil {
		return fmt.Errorf("writeJson Error: %s", err)
	}
	return nil
}

// writes the new cookie expiry time to the JSON
func writeNewCookieExpiry(duration time.Duration) error {
	newExpiryTime := time.Now().Add(duration)
	newExpiryTimeFormatted := timeFormatted{
		Year:   newExpiryTime.Year(),
		Month:  int(newExpiryTime.Month()),
		Day:    newExpiryTime.Day(),
		Hour:   newExpiryTime.Hour(),
		Minute: newExpiryTime.Minute(),
		Second: newExpiryTime.Second(),
		Nsec:   newExpiryTime.Nanosecond(),
	}

	err := writeJson(newExpiryTimeFormatted, COOKIE_EXPIRY_FILE_DIR)
	if err != nil {
		return fmt.Errorf("writeJson Error: %s", err)
	}
	return nil
}

// if the cookie is expired it will get a new one and then write the expiry time of the cookie
// if the cookie is good then it will do nothing
func refreshCookie(duration time.Duration) error {
	// check if COOKIE_EXPIRY_FILE_DIR or COOKIE_FILE_DIR exists
	// ------------- next time just use one file for both data values -----------------
	_, ExpiryErr := os.Stat(COOKIE_EXPIRY_FILE_DIR)
	_, CookieErr := os.Stat(COOKIE_FILE_DIR)
	if errors.Is(ExpiryErr, os.ErrNotExist) || errors.Is(CookieErr, os.ErrNotExist) {
		// if either file is does not exist generate/re-generate both
		// write new cookie expiry
		log.Printf("%s does not exist\n", COOKIE_EXPIRY_FILE_DIR)
		if err := writeNewCookieExpiry(duration); err != nil {
			return fmt.Errorf("writeNewCookieExpiry Error: %s", err)
		}
		// write new cookie
		log.Printf("%s does not exist\n", COOKIE_FILE_DIR)
		if err := writeNewCookie(); err != nil {
			return fmt.Errorf("writeNewCookie Error: %s", err)
		}
		return nil
	}

	// check against current time
	checkedTime, err := checkTime(COOKIE_EXPIRY_FILE_DIR, time.Now())
	if err != nil {
		return fmt.Errorf("Check Time Error: %s", err)
	}
	switch checkedTime {
	// cookie still good
	case 0:
		return nil

	// cookie is expired
	case 1, 2:
		// write new expiry
		if err = writeNewCookieExpiry(duration); err != nil {
			return fmt.Errorf("writeNewCookieExpiry Error: %s", err)
		}

		// write new cookie
		if err = writeNewCookie(); err != nil {
			return fmt.Errorf("writeNewCookie Error: %s", err)
		}

		return nil
	default:
		return fmt.Errorf("refreshCookie default case reached, checkedTime returned: %d", checkedTime)
	}
}

// get cookie returns a working cookie
func getCookie() (asusCookie, error) {
	err := refreshCookie(COOKIE_EXPIRY_DURATION)
	if err != nil {
		return asusCookie{}, fmt.Errorf("refreshCookie Error: %s", err)
	}

	a, err := readJson(COOKIE_FILE_DIR)
	if err != nil {
		return asusCookie{}, fmt.Errorf("readJson Error: %s", err)
	}

	return a, nil
}

// uses a hook and the cookie to return the usage data
func getHook(hook string, asusInterface asusCookie) ([]byte, error) {
	targetHost := "router.asus.com"
	targetPath := "/appGet.cgi"
	method := "POST"
	formattedHook := fmt.Sprintf("hook=%s", hook)
	payload := strings.NewReader(formattedHook)
	asusTokenString := fmt.Sprintf("asus_token=%s", asusInterface.AsusToken)

	uri := fmt.Sprintf("http://%s%s", targetHost, targetPath)

	client := &http.Client{}
	req, err := http.NewRequest(method, uri, payload)
	if err != nil {
		return nil, fmt.Errorf("Error Setting Request: %s", err)
	}

	req.Header.Add("User-Agent", "asusrouter-Android-DUTUtil-1.0.0.245")
	req.Header.Add("Content-Type", "appliction/x-www-form-urlencoded")
	req.Header.Add("Cookie", asusTokenString)
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error client do: %s", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Error ReadAll: %s", err)
	}

	return body, nil
}

// Gets the metrics from the router returns []byte
func getMetricsStrings(hook string) ([]byte, error) {
	asusCookieToken, err := getCookie()
	if err != nil {
		return nil, fmt.Errorf("Error Getting Cookie: %s", err)
	}

	data, err := getHook(hook, asusCookieToken)
	if err != nil {
		return nil, fmt.Errorf("Error Getting Hook: %s", err)
	}

	substring := strings.TrimSpace(string(data)[len(hook)+3 : len(data)-2])

	return []byte(fmt.Sprintf("{%s}", substring)), nil
}

// Gets the metrics from the router returns []byte
func getMetrics(hook string) ([]byte, error) {
	asusCookieToken, err := getCookie()
	if err != nil {
		return nil, fmt.Errorf("Error Getting Cookie: %s", err)
	}

	data, err := getHook(hook, asusCookieToken)
	if err != nil {
		return nil, fmt.Errorf("Error Getting Hook: %s", err)
	}

	return data, nil
}

// prometheus description of the required metrics
type metrics struct {
	cpuUsage         prometheus.Gauge
	BridgeTraffic    prometheus.Counter
	InternetTraffic  prometheus.Counter
	WiredTraffic     prometheus.Counter
	Wireless0Traffic prometheus.Counter
	Wireless1Traffic prometheus.Counter
	MemoryUsage      prometheus.Gauge
}

// represents the usage of single CPU core
type cpuUsage struct {
	CPU_total float64
	CPU_usage float64
}

// parses the raw CPU hook data
type cpuUsageUnstructured struct {
	CPU1total string `json:"cpu1_total"`
	CPU1usage string `json:"cpu1_usage"`
	CPU2total string `json:"cpu2_total"`
	CPU2usage string `json:"cpu2_usage"`
	CPU3total string `json:"cpu3_total"`
	CPU3usage string `json:"cpu3_usage"`
}

type trafficTransmission struct {
	TX float64
	RX float64
}

type netdevTraffic struct {
	Bridge    trafficTransmission
	Internet  trafficTransmission
	Wired     trafficTransmission
	Wireless0 trafficTransmission
	Wireless1 trafficTransmission
}

type netdevTrafficUnstructured struct {
	BRIDGErx    string `json:"BRIDGE_rx"`
	BRIDGEtx    string `json:"BRIDGE_tx"`
	INTERNETrx  string `json:"INTERNET_rx"`
	INTERNETtx  string `json:"INTERNET_tx"`
	WIREDrx     string `json:"WIRED_rx"`
	WIREDtx     string `json:"WIRED_tx"`
	WIRELESS0rx string `json:"WIRELESS0_rx"`
	WIRELESS0tx string `json:"WIRELESS0_tx"`
	WIRELESS1rx string `json:"WIRELESS1_rx"`
	WIRELESS1tx string `json:"WIRELESS1_tx"`
}

type MemoryUsage struct {
	MemTotal float64
	MemFree  float64
	MemUsed  float64
}

type MemoryUsageUnstructured struct {
	MemTotal string `json:"mem_total"`
	MemFree  string `json:"mem_free"`
	MemUsed  string `json:"mem_used"`
}

// all the metrics from the router
type AsusStats struct {
	CPUusage   []cpuUsage
	NetDev     netdevTraffic
	MemMetrics MemoryUsage
}

type UnstructuredData struct {
	CPUusage   cpuUsageUnstructured
	NetDev     netdevTrafficUnstructured
	MemMetrics MemoryUsageUnstructured
}

// Called to setup new prometheus metrics
func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		cpuUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: NAMESPACE,
			Name:      "cpu_usage",
			Help:      "monitored cpu usage",
		}),
		BridgeTraffic: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: NAMESPACE,
			Name:      "bridge_traffic",
			Help:      "bridge monitored traffic",
		}),
		InternetTraffic: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: NAMESPACE,
			Name:      "internet_traffic",
			Help:      "internet monitored traffic",
		}),
		WiredTraffic: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: NAMESPACE,
			Name:      "wired_traffic",
			Help:      "wired monitored traffic",
		}),
		Wireless0Traffic: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: NAMESPACE,
			Name:      "wireless0_traffic",
			Help:      "wireless0 monitored traffic",
		}),
		Wireless1Traffic: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: NAMESPACE,
			Name:      "wireless1_traffic",
			Help:      "wireless1 monitored traffic",
		}),
		MemoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: NAMESPACE,
			Name:      "memory_metrics",
			Help:      "memory: total - free - used",
		}),
	}
	reg.MustRegister(m.cpuUsage)
	reg.MustRegister(m.BridgeTraffic)
	reg.MustRegister(m.InternetTraffic)
	reg.MustRegister(m.WiredTraffic)
	reg.MustRegister(m.Wireless0Traffic)
	reg.MustRegister(m.Wireless1Traffic)
	reg.MustRegister(m.MemoryUsage)
	return m
}

// collector for prometheus
var _ prometheus.Collector = &basicCollector{}

// implements prometheus basic collector
type basicCollector struct {
	cpuUsage         *prometheus.Desc
	BridgeTraffic    *prometheus.Desc
	InternetTraffic  *prometheus.Desc
	WiredTraffic     *prometheus.Desc
	Wireless0Traffic *prometheus.Desc
	Wireless1Traffic *prometheus.Desc
	MemoryUsage      *prometheus.Desc
	stats            func() ([]AsusStats, error)
}

// implements prometheus.Collector
func NewBasicCollector(stats func() ([]AsusStats, error)) prometheus.Collector {
	return &basicCollector{
		cpuUsage: prometheus.NewDesc(
			"cpu_usage",
			"the usage of each cpu core",
			[]string{"cpu_core_number"},
			nil,
		),
		BridgeTraffic: prometheus.NewDesc(
			"bridge_traffic",
			"total traffic rx and tx",
			[]string{"traffic_type"},
			nil,
		),
		InternetTraffic: prometheus.NewDesc(
			"internet_traffic",
			"total traffic rx and tx",
			[]string{"traffic_type"},
			nil,
		),
		WiredTraffic: prometheus.NewDesc(
			"wired_traffic",
			"total traffic rx and tx",
			[]string{"traffic_type"},
			nil,
		),
		Wireless0Traffic: prometheus.NewDesc(
			"wireless0_traffic",
			"total traffic rx and tx",
			[]string{"traffic_type"},
			nil,
		),
		Wireless1Traffic: prometheus.NewDesc(
			"wireless1_traffic",
			"total traffic rx and tx",
			[]string{"traffic_type"},
			nil,
		),
		MemoryUsage: prometheus.NewDesc(
			"memory_metrics",
			"memory metrics (total - free - used)",
			[]string{"metric"},
			nil,
		),
		stats: stats,
	}
}

// interfaces Describe
func (c *basicCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.cpuUsage,
		c.BridgeTraffic,
		c.InternetTraffic,
		c.WiredTraffic,
		c.Wireless0Traffic,
		c.Wireless1Traffic,
		c.MemoryUsage,
	}
	for _, d := range ds {
		ch <- d
	}
}

func fillTraffic(c float64, descpt *prometheus.Desc, channel string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		descpt,
		prometheus.CounterValue,
		c,
		channel,
	)
}

// interfaces Collect
func (c *basicCollector) Collect(ch chan<- prometheus.Metric) {
	// Take a stats snapshot. Must be concurrency safe.
	stats, err := c.stats()
	if err != nil {
		// If an error occurs, send an invalid metric to notify
		// Prometheus of the problem.
		ch <- prometheus.NewInvalidMetric(c.cpuUsage, err)
		ch <- prometheus.NewInvalidMetric(c.BridgeTraffic, err)
		ch <- prometheus.NewInvalidMetric(c.InternetTraffic, err)
		ch <- prometheus.NewInvalidMetric(c.WiredTraffic, err)
		ch <- prometheus.NewInvalidMetric(c.Wireless0Traffic, err)
		ch <- prometheus.NewInvalidMetric(c.Wireless1Traffic, err)
		ch <- prometheus.NewInvalidMetric(c.MemoryUsage, err)
		return
	}
	for _, s := range stats {
		for i, usage := range s.CPUusage {
			usages := []struct {
				CPUCore  string
				CPUTotal float64
				CPUusage float64
			}{
				{CPUCore: fmt.Sprintf("CPU %d", i+1),
					CPUTotal: usage.CPU_total,
					CPUusage: usage.CPU_usage},
			}
			for _, usageT := range usages {
				ch <- prometheus.MustNewConstMetric(
					c.cpuUsage,
					prometheus.GaugeValue,
					usageT.CPUusage,
					usageT.CPUCore,
				)
			}
		}
		// ------------------------  Traffic Metrics  --------------------------
		ch <- fillTraffic(s.NetDev.Bridge.RX, c.BridgeTraffic, "rx")
		ch <- fillTraffic(s.NetDev.Bridge.TX, c.BridgeTraffic, "tx")

		ch <- fillTraffic(s.NetDev.Internet.RX, c.InternetTraffic, "rx")
		ch <- fillTraffic(s.NetDev.Internet.TX, c.InternetTraffic, "tx")

		ch <- fillTraffic(s.NetDev.Wired.RX, c.WiredTraffic, "rx")
		ch <- fillTraffic(s.NetDev.Wired.TX, c.WiredTraffic, "tx")

		ch <- fillTraffic(s.NetDev.Wireless0.RX, c.Wireless0Traffic, "rx")
		ch <- fillTraffic(s.NetDev.Wireless0.TX, c.Wireless0Traffic, "tx")

		ch <- fillTraffic(s.NetDev.Wireless1.RX, c.Wireless1Traffic, "rx")
		ch <- fillTraffic(s.NetDev.Wireless1.TX, c.Wireless1Traffic, "tx")
		// ---------------------------------------------------------------------

		ch <- prometheus.MustNewConstMetric(
			c.MemoryUsage,
			prometheus.GaugeValue,
			s.MemMetrics.MemTotal,
			"mem_total",
		)
		ch <- prometheus.MustNewConstMetric(
			c.MemoryUsage,
			prometheus.GaugeValue,
			s.MemMetrics.MemFree,
			"mem_free",
		)
		ch <- prometheus.MustNewConstMetric(
			c.MemoryUsage,
			prometheus.GaugeValue,
			s.MemMetrics.MemUsed,
			"mem_used",
		)

	}
}

// Parses CPU stats
func ScanCPUStats(data cpuUsageUnstructured) ([]cpuUsage, error) {
	var cpuStats []cpuUsage
	//var asusStats AsusStats

	var unstructuredCPUStats [6]float64
	var err error

	unstructuredCPUStats[0], err = strconv.ParseFloat(data.CPU1total, 64)
	unstructuredCPUStats[1], err = strconv.ParseFloat(data.CPU1usage, 64)
	unstructuredCPUStats[2], err = strconv.ParseFloat(data.CPU2total, 64)
	unstructuredCPUStats[3], err = strconv.ParseFloat(data.CPU2usage, 64)
	unstructuredCPUStats[4], err = strconv.ParseFloat(data.CPU3total, 64)
	unstructuredCPUStats[5], err = strconv.ParseFloat(data.CPU3usage, 64)
	if err != nil {
		return nil, fmt.Errorf("strconv.ParseFloat Error: %s", err)
	}

	cpuStats = append(cpuStats, cpuUsage{unstructuredCPUStats[0], unstructuredCPUStats[1]})
	cpuStats = append(cpuStats, cpuUsage{unstructuredCPUStats[2], unstructuredCPUStats[3]})
	cpuStats = append(cpuStats, cpuUsage{unstructuredCPUStats[4], unstructuredCPUStats[5]})

	//asusStats.CPUusage = cpuStats

	return cpuStats, nil
}

func hexToFloat(hexStr string, factor float64) (float64, error) {
	// remove 0x suffix if found in the input string
	cleaned := strings.Replace(hexStr, "0x", "", -1)

	// base 16 for hexadecimal
	result, err := strconv.ParseUint(cleaned, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("strconv.ParseFloat Error: %s", err)
	}
	ret := float64(result) / factor
	return ret, nil
}

func ScanTrafficStats(data netdevTrafficUnstructured) (netdevTraffic, error) {
	var trafficStats netdevTraffic

	var err error
	trafficStats.Bridge.RX, err = hexToFloat(data.BRIDGErx, 1000000)
	trafficStats.Bridge.TX, err = hexToFloat(data.BRIDGEtx, 1000000)
	trafficStats.Internet.RX, err = hexToFloat(data.INTERNETrx, 1000000)
	trafficStats.Internet.TX, err = hexToFloat(data.INTERNETtx, 1000000)
	trafficStats.Wired.RX, err = hexToFloat(data.WIREDrx, 1000000)
	trafficStats.Wired.TX, err = hexToFloat(data.WIREDtx, 1000000)
	trafficStats.Wireless0.RX, err = hexToFloat(data.WIRELESS0rx, 1000000)
	trafficStats.Wireless0.TX, err = hexToFloat(data.WIRELESS0tx, 1000000)
	trafficStats.Wireless1.RX, err = hexToFloat(data.WIRELESS1rx, 1000000)
	trafficStats.Wireless1.TX, err = hexToFloat(data.WIRELESS1tx, 1000000)
	if err != nil {
		return trafficStats, fmt.Errorf("strconv.ParseFloat Error: %s", err)
	}
	return trafficStats, nil
}

func ScanMemMetrics(data MemoryUsageUnstructured) (MemoryUsage, error) {
	var memoryMetrics MemoryUsage
	var err error
	memoryMetrics.MemTotal, err = strconv.ParseFloat(data.MemTotal, 64)
	memoryMetrics.MemFree, err = strconv.ParseFloat(data.MemFree, 64)
	memoryMetrics.MemUsed, err = strconv.ParseFloat(data.MemUsed, 64)
	if err != nil {
		return memoryMetrics, fmt.Errorf("strconv.ParseFloat Error: %s", err)
	}
	memoryMetrics.MemTotal = memoryMetrics.MemTotal / 1024
	memoryMetrics.MemFree = memoryMetrics.MemFree / 1024
	memoryMetrics.MemUsed = memoryMetrics.MemUsed / 1024
	return memoryMetrics, nil
}

// Parses all unstructured data
func ScanStats(data UnstructuredData) ([]AsusStats, error) {
	var stats []AsusStats
	cpuUsage, err := ScanCPUStats(data.CPUusage)
	if err != nil {
		return nil, fmt.Errorf("ScanCPUStats Error: %s", err)
	}
	trafficUsage, err := ScanTrafficStats(data.NetDev)
	if err != nil {
		return nil, fmt.Errorf("ScanTrafficStats Error: %s", err)
	}
	memoryMetrics, err := ScanMemMetrics(data.MemMetrics)
	if err != nil {
		return nil, fmt.Errorf("ScanMemMetrics Error: %s", err)
	}

	stats = append(stats, AsusStats{
		CPUusage:   cpuUsage,
		NetDev:     trafficUsage,
		MemMetrics: memoryMetrics,
	})

	return stats, nil
}

func main() {
	flag.Parse()

	mainHandler := func(w http.ResponseWriter, r *http.Request) {
		asusCookieToken, err := getCookie()
		if err != nil {
			log.Fatalf("Error Getting Cookie: %s", err)
		}

		hook := fmt.Sprint("cpu_usage()") // --------- to change into parameter ---------

		cpuUsage, err := getHook(hook, asusCookieToken)
		if err != nil {
			log.Fatalf("Error Getting Hook: %s", err)
		}

		substring := strings.TrimSpace(string(cpuUsage)[len(hook)+3 : len(cpuUsage)-2])
		var netdev struct {
			NetDev netdevTrafficUnstructured `json:"netdev"`
		}
		body, err := getMetrics("netdev(appobj)")
		if err != nil {
			log.Fatalf("getMetrics Error: %s", err)
		}
		if err = json.Unmarshal(body, &netdev); err != nil {
			log.Fatalf("json unmarshal error: %s", err)
		}
		trafficUsage, err := ScanTrafficStats(netdev.NetDev)
		if err != nil {
			log.Fatalf("ScanTrafficStats Error: %s", err)
		}
		data, err := json.MarshalIndent(trafficUsage, "", "  ")
		if err != nil {
			log.Fatalf("json Marshal Error: %s", err)
		}

		fmt.Fprintf(w, "{%s}\n%s", substring, data)
	}

	cookie := func(w http.ResponseWriter, r *http.Request) {
		asusCookieToken, err := getCookie()
		if err != nil {
			log.Fatalf("Error Getting Cookie: %s", err)
		}
		fmt.Fprintf(w, "%s", asusCookieToken)
	}

	basicStats := func() ([]AsusStats, error) {

		// fetch cpu_usage
		var cpuData struct {
			CPUusage cpuUsageUnstructured `json:"cpu_usage"`
		}
		body, err := getMetrics("cpu_usage(appobj)")
		if err != nil {
			return nil, fmt.Errorf("getMetrics Error: %s", err)
		}
		if err = json.Unmarshal(body, &cpuData); err != nil {
			return nil, fmt.Errorf("json unmarshal error: %s", err)
		}

		var netdev struct {
			NetDev netdevTrafficUnstructured `json:"netdev"`
		}
		body, err = getMetrics("netdev(appobj)")
		if err != nil {
			return nil, fmt.Errorf("getMetrics Error: %s", err)
		}
		if err = json.Unmarshal(body, &netdev); err != nil {
			return nil, fmt.Errorf("json unmarshal error: %s", err)
		}

		var memusage struct {
			MemoryUsage MemoryUsageUnstructured `json:"memory_usage"`
		}
		body, err = getMetrics("memory_usage(appobj)")
		if err != nil {
			return nil, fmt.Errorf("getMetrics Error: %s", err)
		}
		if err = json.Unmarshal(body, &memusage); err != nil {
			return nil, fmt.Errorf("json unmarshal error: %s", err)
		}

		// create Unstructured data
		data := UnstructuredData{
			CPUusage:   cpuData.CPUusage,
			NetDev:     netdev.NetDev,
			MemMetrics: memusage.MemoryUsage,
		}

		return ScanStats(data)
	}

	SetAuth(*UserName, *Password)

	bc := NewBasicCollector(basicStats)
	reg := prometheus.NewRegistry()
	reg.MustRegister(bc)
	
	mux := http.NewServeMux()
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	mux.HandleFunc("/main", mainHandler)
	mux.HandleFunc("/getcookie", cookie)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	mux.Handle("/metrics", promHandler)

	// Start listening for HTTP connections
	port := fmt.Sprintf(":%d", *hostPort)
	log.Printf("starting asus exporter on %q/metrics", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("cannot start asus exporter: %s", err)
	}
}
