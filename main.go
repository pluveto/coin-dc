package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	token := os.Getenv("INFLUXDB_TOKEN")
	if token == "" {
		log.Fatal("INFLUXDB_TOKEN environment variable not set")
	}
	instId := os.Getenv("INST_ID")
	if instId == "" {
		instId = "BTC-USDT"
	}

	url := os.Getenv("INFLUXDB_URL")
	if url == "" {
		url = "http://localhost:8086"
	}

	client := influxdb2.NewClient(url, token)

	org := os.Getenv("INFLUXDB_ORG")
	if org == "" {
		org = "MyOrg"
	}

	bucket := os.Getenv("INFLUXDB_BUCKET")
	if bucket == "" {
		bucket = "MyBucket"
	}

	interval := 5 // seconds
	if os.Getenv("INTERVAL") != "" {
		interval_, err := strconv.Atoi(os.Getenv("INTERVAL"))
		if err != nil {
			log.Fatal("Invalid INTERVAL value:", os.Getenv("INTERVAL"))
		} else {
			interval = interval_
		}
	}

	writeAPI := client.WriteAPIBlocking(org, bucket)

	for {
		data, err := fetchData(instId)
		if err != nil {
			log.Println("Failed to fetch data:", err)
			continue
		}

		if data.Code != "0" {
			log.Println("Failed to fetch data:", data.Msg)
			continue
		}

		if len(data.Data) == 0 {
			log.Println("No data received")
		}

		instId := data.Data[0].InstId
		idxPx := data.Data[0].IdxPx

        point := influxdb2.NewPointWithMeasurement("coin_price").
            AddTag("instId", instId).
            AddField("idxPx", idxPx).
            SetTime(time.Now())

		if err := writeAPI.WritePoint(context.Background(), point); err != nil {
			log.Fatal(err)
		}

		log.Println("Wrote data to InfluxDB:", point)

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

type Response struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		InstId  string `json:"instId"`
		IdxPx   string `json:"idxPx"`
		High24h string `json:"high24h"`
		SodUtc0 string `json:"sodUtc0"`
		Open24h string `json:"open24h"`
		Low24h  string `json:"low24h"`
		SodUtc8 string `json:"sodUtc8"`
		Ts      string `json:"ts"`
	} `json:"data"`
}

func fetchData(instId string) (*Response, error) {
	url := "https://www.okx.com/api/v5/market/index-tickers?instId=" + instId

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data Response
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}
