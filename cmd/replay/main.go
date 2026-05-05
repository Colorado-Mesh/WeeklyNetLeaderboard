package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"meshmonday/internal/config"
)

type replayLine struct {
	Topic      string         `json:"topic"`
	PayloadHex string         `json:"payload_hex"`
	ObservedAt string         `json:"observed_at"`
	Meta       map[string]any `json:"meta"`
}

func main() {
	_ = config.LoadEnvFile(".env.local")
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	fixturePath := flag.String("fixture", "fixtures/mqtt/monday-sea.jsonl", "path to jsonl fixture")
	flag.Parse()

	opts := paho.NewClientOptions()
	opts.AddBroker(cfg.MQTTBrokerURL)
	opts.SetClientID(cfg.MQTTClientID + "-replay")
	if cfg.MQTTUsername != "" {
		opts.SetUsername(cfg.MQTTUsername)
		opts.SetPassword(cfg.MQTTPassword)
	}
	client := paho.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer client.Disconnect(200)

	file, err := os.Open(*fixturePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	published := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		var entry replayLine
		if err := json.Unmarshal(line, &entry); err != nil {
			panic(fmt.Errorf("decode fixture line: %w", err))
		}
		if entry.Topic == "" || entry.PayloadHex == "" {
			continue
		}
		token := client.Publish(entry.Topic, 1, false, []byte(entry.PayloadHex))
		token.Wait()
		if err := token.Error(); err != nil {
			panic(err)
		}
		published++
		time.Sleep(cfg.ReplayDelay)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Printf("published %d fixture messages\n", published)
}
