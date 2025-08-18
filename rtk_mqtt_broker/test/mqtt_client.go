package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("收到訊息: Topic=%s, Message=%s\n", msg.Topic(), msg.Payload())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("連接到 MQTT Broker 成功")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("連接中斷: %v\n", err)
}

func main() {
	broker := "localhost"
	port := 1883
	
	fmt.Println("=== RTK MQTT Broker 測試程序 ===")
	
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID("mqtt_test_client")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("連接失敗: %v", token.Error())
	}

	testTopics := []string{
		"test/basic",
		"sensor/temperature",
		"sensor/humidity", 
		"device/status",
	}

	var wg sync.WaitGroup
	receivedMessages := make(map[string]int)
	var mutex sync.Mutex

	for _, topic := range testTopics {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			
			token := client.Subscribe(t, 0, func(client mqtt.Client, msg mqtt.Message) {
				mutex.Lock()
				receivedMessages[t]++
				mutex.Unlock()
				fmt.Printf("📥 [%s] %s\n", msg.Topic(), msg.Payload())
			})
			token.Wait()
			
			if token.Error() != nil {
				fmt.Printf("訂閱 %s 失敗: %v\n", t, token.Error())
			} else {
				fmt.Printf("✅ 訂閱 %s 成功\n", t)
			}
		}(topic)
	}

	time.Sleep(1 * time.Second)

	fmt.Println("\n開始發布測試訊息...")
	
	testMessages := map[string][]string{
		"test/basic":         {"Hello MQTT!", "Test message 1", "Test message 2"},
		"sensor/temperature": {"23.5°C", "24.1°C", "22.8°C"},
		"sensor/humidity":    {"65%", "68%", "62%"},
		"device/status":      {"online", "active", "standby"},
	}

	for topic, messages := range testMessages {
		for _, message := range messages {
			token := client.Publish(topic, 0, false, message)
			token.Wait()
			
			if token.Error() != nil {
				fmt.Printf("❌ 發布到 %s 失敗: %v\n", topic, token.Error())
			} else {
				fmt.Printf("📤 [%s] %s\n", topic, message)
			}
			
			time.Sleep(500 * time.Millisecond)
		}
	}

	time.Sleep(2 * time.Second)

	fmt.Println("\n=== QoS 等級測試 ===")
	qosTestTopic := "test/qos"
	
	for qos := 0; qos <= 2; qos++ {
		token := client.Subscribe(qosTestTopic, byte(qos), func(client mqtt.Client, msg mqtt.Message) {
			fmt.Printf("📥 QoS %d: [%s] %s\n", msg.Qos(), msg.Topic(), msg.Payload())
		})
		token.Wait()
		
		message := fmt.Sprintf("QoS %d test message", qos)
		token = client.Publish(qosTestTopic, byte(qos), false, message)
		token.Wait()
		
		if token.Error() != nil {
			fmt.Printf("❌ QoS %d 測試失敗: %v\n", qos, token.Error())
		} else {
			fmt.Printf("📤 QoS %d: %s\n", qos, message)
		}
		
		time.Sleep(1 * time.Second)
	}

	wg.Wait()

	fmt.Println("\n=== 測試結果統計 ===")
	mutex.Lock()
	totalReceived := 0
	for topic, count := range receivedMessages {
		fmt.Printf("Topic: %s, 收到訊息數: %d\n", topic, count)
		totalReceived += count
	}
	mutex.Unlock()
	
	expectedMessages := 0
	for _, messages := range testMessages {
		expectedMessages += len(messages)
	}
	
	fmt.Printf("預期訊息數: %d, 實際收到: %d\n", expectedMessages, totalReceived)
	
	if totalReceived >= expectedMessages {
		fmt.Println("✅ 測試通過!")
	} else {
		fmt.Println("❌ 測試失敗: 訊息遺失")
	}

	fmt.Println("\n斷開連接...")
	client.Disconnect(250)
	fmt.Println("測試完成!")
}