package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MessageStats struct {
	Count     int
	FirstMsg  time.Time
	LastMsg   time.Time
	StartTime time.Time
}

func main() {
	var (
		broker   = flag.String("broker", "localhost", "MQTT broker 地址")
		port     = flag.Int("port", 1883, "MQTT broker 埠號")
		topic    = flag.String("topic", "test/+", "要監聽的 Topic (支援通配符)")
		clientID = flag.String("client", "topic_listener", "客戶端 ID")
		qos      = flag.Int("qos", 0, "QoS 等級 (0, 1, 2)")
		duration = flag.Int("duration", 0, "監聽時間 (秒, 0=無限)")
		verbose  = flag.Bool("verbose", false, "顯示詳細訊息")
	)
	flag.Parse()

	if *qos < 0 || *qos > 2 {
		log.Fatal("QoS 等級必須是 0, 1, 或 2")
	}

	fmt.Printf("=== RTK MQTT Topic 監聽器 ===\n")
	fmt.Printf("Broker: %s:%d\n", *broker, *port)
	fmt.Printf("Topic: %s\n", *topic)
	fmt.Printf("QoS: %d\n", *qos)
	if *duration > 0 {
		fmt.Printf("監聽時間: %d 秒\n", *duration)
	} else {
		fmt.Printf("監聽時間: 無限 (按 Ctrl+C 停止)\n")
	}
	fmt.Printf("----------------------------------------\n")

	stats := &MessageStats{
		StartTime: time.Now(),
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", *broker, *port))
	opts.SetClientID(*clientID)
	opts.SetCleanSession(true)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	
	opts.OnConnect = func(client mqtt.Client) {
		fmt.Printf("✅ 已連接到 MQTT Broker\n")
		fmt.Printf("📡 開始監聽 Topic: %s\n", *topic)
		fmt.Printf("----------------------------------------\n")
	}
	
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		fmt.Printf("❌ 連接中斷: %v\n", err)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("連接失敗: %v", token.Error())
	}
	defer client.Disconnect(250)

	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		now := time.Now()
		stats.Count++
		
		if stats.FirstMsg.IsZero() {
			stats.FirstMsg = now
		}
		stats.LastMsg = now
		
		timestamp := now.Format("2006-01-02 15:04:05.000")
		
		if *verbose {
			fmt.Printf("📥 [%s] Topic: %s\n", timestamp, msg.Topic())
			fmt.Printf("   QoS: %d, Retained: %v, Duplicate: %v\n", 
				msg.Qos(), msg.Retained(), msg.Duplicate())
			fmt.Printf("   Message: %s\n", string(msg.Payload()))
			fmt.Printf("   Length: %d bytes\n", len(msg.Payload()))
			fmt.Printf("----------------------------------------\n")
		} else {
			fmt.Printf("📥 [%s] [%s] %s\n", timestamp, msg.Topic(), string(msg.Payload()))
		}
	}

	if token := client.Subscribe(*topic, byte(*qos), messageHandler); token.Wait() && token.Error() != nil {
		log.Fatalf("訂閱失敗: %v", token.Error())
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	var timeoutCh <-chan time.Time
	if *duration > 0 {
		timeoutCh = time.After(time.Duration(*duration) * time.Second)
	}

	statsTicker := time.NewTicker(10 * time.Second)
	defer statsTicker.Stop()

	for {
		select {
		case <-signalCh:
			fmt.Printf("\n收到停止信號，正在退出...\n")
			goto cleanup
			
		case <-timeoutCh:
			fmt.Printf("\n監聽時間到達，正在退出...\n")
			goto cleanup
			
		case <-statsTicker.C:
			if stats.Count > 0 {
				elapsed := time.Since(stats.StartTime)
				rate := float64(stats.Count) / elapsed.Seconds()
				fmt.Printf("📊 統計: 收到 %d 條訊息, 速率: %.2f msg/sec\n", stats.Count, rate)
			}
		}
	}

cleanup:
	fmt.Printf("----------------------------------------\n")
	fmt.Printf("=== 監聽結束統計 ===\n")
	
	totalTime := time.Since(stats.StartTime)
	fmt.Printf("總監聽時間: %v\n", totalTime.Round(time.Second))
	fmt.Printf("收到訊息總數: %d\n", stats.Count)
	
	if stats.Count > 0 {
		rate := float64(stats.Count) / totalTime.Seconds()
		fmt.Printf("平均訊息速率: %.2f msg/sec\n", rate)
		fmt.Printf("第一條訊息時間: %s\n", stats.FirstMsg.Format("2006-01-02 15:04:05"))
		fmt.Printf("最後一條訊息時間: %s\n", stats.LastMsg.Format("2006-01-02 15:04:05"))
		
		if !stats.FirstMsg.IsZero() && !stats.LastMsg.IsZero() {
			msgSpan := stats.LastMsg.Sub(stats.FirstMsg)
			if msgSpan > 0 {
				msgRate := float64(stats.Count-1) / msgSpan.Seconds()
				fmt.Printf("訊息期間速率: %.2f msg/sec\n", msgRate)
			}
		}
	} else {
		fmt.Printf("未收到任何訊息\n")
	}
	
	fmt.Printf("監聽器正常退出\n")
}