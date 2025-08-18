package main

import (
	"flag"
	"fmt"
	"os"

	"rtk_mqtt_broker/broker"
	"rtk_mqtt_broker/config"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	mqttBroker := broker.New(cfg)

	err = mqttBroker.Start()
	if err != nil {
		fmt.Printf("Failed to start broker: %v\n", err)
		os.Exit(1)
	}

	mqttBroker.WaitForSignal()

	err = mqttBroker.Stop()
	if err != nil {
		fmt.Printf("Failed to stop broker: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Realtek Embedded MQTT Broker stopped gracefully")
}