package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	broker := "localhost"
	port := 1883
	
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID("simple_publisher")
	
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("連接失敗: %v", token.Error())
	}
	defer client.Disconnect(250)

	fmt.Println("開始發布訊息到 sensor/ topic...")
	
	topics := []string{
		"sensor/temperature",
		"sensor/humidity", 
		"sensor/pressure",
	}
	
	for i := 0; i < 5; i++ {
		for _, topic := range topics {
			message := fmt.Sprintf("測試訊息 #%d", i+1)
			token := client.Publish(topic, 0, false, message)
			token.Wait()
			
			if token.Error() != nil {
				fmt.Printf("❌ 發布到 %s 失敗: %v\n", topic, token.Error())
			} else {
				fmt.Printf("📤 [%s] %s\n", topic, message)
			}
			
			time.Sleep(1 * time.Second)
		}
	}
	
	fmt.Println("發布完成!")
}