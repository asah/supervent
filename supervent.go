package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/valyala/fasthttp"
)

const DEFAULT_BATCH_SIZE = 100

type Config struct {
	Sources []SourceConfig `json:"sources"`
}

type SourceConfig struct {
	Vendor          string           `json:"vendor"`
	TimestampFormat string           `json:"timestamp_format"`
	Fields          map[string]Field `json:"fields"`
}

type Field struct {
	Type          string                 `json:"type"`
	AllowedValues []string               `json:"allowed_values,omitempty"`
	Weights       []float64              `json:"weights,omitempty"`
	Constraints   map[string]interface{} `json:"constraints,omitempty"`
	Distribution  string                 `json:"distribution,omitempty"`
	Mean          float64                `json:"mean,omitempty"`
	Stddev        float64                `json:"stddev,omitempty"`
	Lambda        float64                `json:"lambda,omitempty"`
	S             float64                `json:"s,omitempty"`
	Alpha         float64                `json:"alpha,omitempty"`
	Format        string                 `json:"format,omitempty"`
	Formats       []string               `json:"formats,omitempty"`
}

type EventGenerator struct {
	Dataset      string
	APIKey       string
	URL          string
	BatchSize    int
	Batch        []map[string]interface{}
	PostgresConn *sql.DB
}

func NewEventGenerator(dataset, apiKey string, batchSize int, postgresConfig *PostgresConfig) *EventGenerator {
	eg := &EventGenerator{
		Dataset:   dataset,
		APIKey:    apiKey,
		URL:       fmt.Sprintf("https://api.axiom.co/v1/datasets/%s/ingest", dataset),
		BatchSize: batchSize,
		Batch:     make([]map[string]interface{}, 0, batchSize),
	}

	if postgresConfig != nil {
		connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
			postgresConfig.Host, postgresConfig.Port, postgresConfig.DBName, postgresConfig.User, postgresConfig.Password)
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		eg.PostgresConn = db
	}

	return eg
}

func (eg *EventGenerator) Emit(record map[string]interface{}) {
	record["_time"] = time.Now().UTC().Format(time.RFC3339)
	eg.Batch = append(eg.Batch, record)
	if len(eg.Batch) >= eg.BatchSize {
		eg.SendBatch()
	}
}

func (eg *EventGenerator) SendBatch() {
	if len(eg.Batch) == 0 {
		return
	}

	fmt.Println("sending batch")
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", eg.APIKey),
	}

	body, err := json.Marshal(eg.Batch)
	if err != nil {
		log.Fatalf("Failed to marshal batch: %v", err)
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(eg.URL)
	req.Header.SetMethod("POST")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.SetBody(body)

	resp := fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		log.Printf("Failed to send batch: %v", err)
	} else if resp.StatusCode() != fasthttp.StatusOK {
		log.Printf("Failed to send batch: %d", resp.StatusCode())
	} else {
		fmt.Println("Batch sent successfully")
	}

	if eg.PostgresConn != nil {
		eg.SendToPostgres(eg.Batch)
	}

	eg.Batch = eg.Batch[:0]
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
}

func (eg *EventGenerator) SendToPostgres(batch []map[string]interface{}) {
	for _, record := range batch {
		columns := make([]string, 0, len(record))
		values := make([]interface{}, 0, len(record))
		for k, v := range record {
			columns = append(columns, k)
			values = append(values, v)
		}
		insertStatement := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			eg.Dataset, join(columns, ","), placeholders(len(values)))
		_, err := eg.PostgresConn.Exec(insertStatement, values...)
		if err != nil {
			log.Printf("Failed to insert record into PostgreSQL: %v", err)
		}
	}
}

func join(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func placeholders(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("$%d", i+1)
	}
	return result
}

type PostgresConfig struct {
	Host     string
	Port     int
	DBName   string
	User     string
	Password string
}

func loadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func generateEvent(sourceConfig SourceConfig) map[string]interface{} {
	event := map[string]interface{}{
		"source": sourceConfig.Vendor,
	}
	for field, details := range sourceConfig.Fields {
		switch details.Type {
		case "datetime":
			event[field] = generateDatetime(details, sourceConfig.TimestampFormat)
		case "string":
			event[field] = generateString(details, field)
		case "int":
			event[field] = generateInt(details)
		}
	}
	fmt.Println(event) // Debug statement to print the complete event
	return event
}

func generateDatetime(details Field, timestampFormat string) string {
	switch timestampFormat {
	case "UTC":
		return time.Now().UTC().Format(details.Format)
	case "ISO":
		return time.Now().Format(time.RFC3339)
	case "Unix":
		return fmt.Sprintf("%d", time.Now().Unix())
	case "RFC3339":
		return time.Now().Format(time.RFC3339)
	default:
		return time.Now().Format(details.Format)
	}
}

func generateString(details Field, fieldName string) string {
	if len(details.AllowedValues) > 0 {
		if len(details.Weights) > 0 {
			return weightedChoice(details.AllowedValues, details.Weights)
		}
		return details.AllowedValues[rand.Intn(len(details.AllowedValues))]
	}
	if len(details.Formats) > 0 {
		selectedFormat := details.Formats[rand.Intn(len(details.Formats))]
		return fmt.Sprintf(selectedFormat, time.Now().Format(details.Format))
	}
	if fieldName == "client_ip" || fieldName == "server_ip" || fieldName == "source_ip" || fieldName == "destination_ip" {
		return generateRandomIPAddress()
	}
	return uuid.New().String()
}

func generateInt(details Field) int {
	min := int(details.Constraints["min"].(float64))
	max := int(details.Constraints["max"].(float64))
	switch details.Distribution {
	case "uniform":
		return rand.Intn(max-min+1) + min
	case "normal":
		return int(rand.NormFloat64()*details.Stddev + details.Mean)
	case "exponential":
		return int(rand.ExpFloat64() / details.Lambda)
	case "zipfian":
		return int(randZipf(details.S))
	case "long_tail":
		return int(randPareto(details.Alpha))
	case "random":
		return rand.Intn(max-min+1) + min
	default:
		return rand.Intn(max-min+1) + min
	}
}

func generateRandomIPAddress() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
}

func weightedChoice(values []string, weights []float64) string {
	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}
	r := rand.Float64() * totalWeight
	for i, weight := range weights {
		if r < weight {
			return values[i]
		}
		r -= weight
	}
	return values[len(values)-1]
}

func randZipf(s float64) float64 {
	return rand.ExpFloat64() / s
}

func randPareto(alpha float64) float64 {
	return rand.ExpFloat64() / alpha
}

func main() {
	configPath := flag.String("config", "config.json", "Path to the configuration file")
	dataset := flag.String("dataset", "", "Axiom dataset name")
	apiKey := flag.String("api_key", "", "Axiom API key")
	batchSize := flag.Int("batch_size", DEFAULT_BATCH_SIZE, "Batch size for HTTP requests")
	postgresHost := flag.String("postgres_host", "", "PostgreSQL host")
	postgresPort := flag.Int("postgres_port", 5432, "PostgreSQL port")
	postgresDB := flag.String("postgres_db", "", "PostgreSQL database name")
	postgresUser := flag.String("postgres_user", "", "PostgreSQL user")
	postgresPassword := flag.String("postgres_password", "", "PostgreSQL password")
	flag.Parse()

	if *dataset == "" || *apiKey == "" {
		log.Fatal("dataset and api_key are required")
	}

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var postgresConfig *PostgresConfig
	if *postgresHost != "" && *postgresDB != "" && *postgresUser != "" && *postgresPassword != "" {
		postgresConfig = &PostgresConfig{
			Host:     *postgresHost,
			Port:     *postgresPort,
			DBName:   *postgresDB,
			User:     *postgresUser,
			Password: *postgresPassword,
		}
	}

	eventGenerator := NewEventGenerator(*dataset, *apiKey, *batchSize, postgresConfig)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	go func() {
		<-sigChan
		fmt.Println("Received interrupt signal, sending remaining events...")
		eventGenerator.SendBatch()
		if eventGenerator.PostgresConn != nil {
			eventGenerator.PostgresConn.Close()
		}
		os.Exit(0)
	}()

	for {
		for _, source := range config.Sources {
			event := generateEvent(source)
			eventGenerator.Emit(event)
		}
		time.Sleep(1 * time.Second) // Adjust the sleep duration as needed
	}
}
