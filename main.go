package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

type StockInfo struct {
	At      string `json:"@"`
	Tv      string `json:"tv"`
	Ps      string `json:"ps"`
	Nu      string `json:"nu"`
	Pid     string `json:"pid"`
	Pz      string `json:"pz"`
	Bp      string `json:"bp"`
	Fv      string `json:"fv"`
	Oa      string `json:"oa"`
	Ob      string `json:"ob"`
	M       string `json:"m%"`
	Key     string `json:"key"`
	Caret   string `json:"^"`
	A       string `json:"a"`
	B       string `json:"b"`
	C       string `json:"c"`
	Hash    string `json:"#"`
	D       string `json:"d"`
	Percent string `json:"%"`
	Ch      string `json:"ch"`
	Tlong   string `json:"tlong"`
	Ot      string `json:"ot"`
	F       string `json:"f"`
	G       string `json:"g"`
	Ip      string `json:"ip"`
	Mt      string `json:"mt"`
	Ov      string `json:"ov"`
	H       string `json:"h"`
	It      string `json:"it"`
	Oz      string `json:"oz"`
	L       string `json:"l"`
	N       string `json:"n"`
	O       string `json:"o"`
	P       string `json:"p"`
	Ex      string `json:"ex"`
	S       string `json:"s"`
	T       string `json:"t"`
	U       string `json:"u"`
	V       string `json:"v"`
	W       string `json:"w"`
	Nf      string `json:"nf"`
	Y       string `json:"y"`
	Z       string `json:"z"`
	Ts      string `json:"ts"`
}

type Response struct {
	MsgArray []StockInfo `json:"msgArray"`
}

type Config struct {
	ExChList []string `yaml:"exChList"`
	Address  string   `yaml:"address"`
	Port     int      `yaml:"port"`
}

var (
	cacheData      []StockInfo
	cacheTimestamp time.Time
	cacheMutex     sync.Mutex
)

func fetchStockInfo(exChList []string) ([]StockInfo, error) {
	// 將 string list 轉換為以 '|' 分隔的字串
	exCh := strings.Join(exChList, "|")

	// construct url
	url := fmt.Sprintf("http://mis.twse.com.tw/stock/api/getStockInfo.jsp?ex_ch=%s", exCh)

	// Send HTTP request
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch stock info: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	// 解析 JSON 响应
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return response.MsgArray, nil
}

func getCachedStockInfo(exChList []string) ([]StockInfo, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if time.Since(cacheTimestamp) < 5*time.Second {
		return cacheData, nil
	}

	stockInfos, err := fetchStockInfo(exChList)
	if err != nil {
		return nil, err
	}

	cacheData = stockInfos
	cacheTimestamp = time.Now()

	return stockInfos, nil
}

func main() {
	// 解析命令行参数
	configFile := flag.String("config", "config.yaml", "Path to the config file")
	flag.Parse()

	// 读取配置文件
	file, err := os.Open(*configFile)
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()

	// 解析配置文件
	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Failed to decode config file: %v", err)
	}

	// Create HTTP handler to expose metrics
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// 使用 exChList
		stockInfos, err := getCachedStockInfo(config.ExChList)
		if err != nil {
			log.Printf("Error fetching stock info: %v", err)
			http.Error(w, "Failed to fetch stock info", http.StatusInternalServerError)
			return
		}

		// Create registry
		registry := prometheus.NewRegistry()

		for _, info := range stockInfos {
			metricName := fmt.Sprintf("%s_%s_gauge", info.Ex, info.C)
			metricHelp := fmt.Sprintf("%s_%s於%s %s的價格%s", info.Ex, info.At, info.D, info.Percent, info.Pz)
			// Create gauge
			gaugeMetric := prometheus.NewGauge(prometheus.GaugeOpts{
				Name: metricName, // metric name
				Help: metricHelp, // metric help
			})

			// Convert Pz to float64
			price, err := strconv.ParseFloat(info.Pz, 64)
			if err != nil {
				log.Printf("Failed to parse price: %v", err)
				http.Error(w, "Failed to parse price", http.StatusInternalServerError)
				return
			}

			// Set value
			gaugeMetric.Set(price)

			// register gauge
			registry.MustRegister(gaugeMetric)
		}

		// Use promhttp.HandlerFor
		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})

	// Start web server
	listenAddress := fmt.Sprintf("%s:%d", config.Address, config.Port)
	http.ListenAndServe(listenAddress, nil)
}
