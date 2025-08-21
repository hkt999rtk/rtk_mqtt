package broker

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
	
	"rtk_mqtt_broker/config"
	"rtk_mqtt_broker/logger"
)

type AllowHook struct {
	mqtt.HookBase
}

func (h *AllowHook) ID() string {
	return "allow-all"
}

func (h *AllowHook) Provides(b byte) bool {
	return b == mqtt.OnConnectAuthenticate ||
		   b == mqtt.OnACLCheck
}

func (h *AllowHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	return true
}

func (h *AllowHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	return true
}

type Broker struct {
	server     *mqtt.Server
	config     *config.Config
	logger     *logger.Logger
}

func New(cfg *config.Config) *Broker {
	logger := logger.New(cfg.Logging.Level)
	
	server := mqtt.New(&mqtt.Options{
		InlineClient: true,
	})
	
	server.AddHook(new(AllowHook), nil)

	return &Broker{
		server: server,
		config: cfg,
		logger: logger,
	}
}

func (b *Broker) Start() error {
	// Display banner
	b.printBanner()

	tcp := listeners.NewTCP(listeners.Config{
		ID:      "tcp",
		Address: fmt.Sprintf("%s:%d", b.config.Server.Host, b.config.Server.Port),
	})

	err := b.server.AddListener(tcp)
	if err != nil {
		return fmt.Errorf("failed to add TCP listener: %w", err)
	}

	go func() {
		err := b.server.Serve()
		if err != nil {
			b.logger.Error(fmt.Sprintf("Server error: %v", err))
		}
	}()

	// Display startup information
	address := fmt.Sprintf("%s:%d", b.config.Server.Host, b.config.Server.Port)
	b.logger.Info(fmt.Sprintf("Realtek Embedded MQTT Broker started successfully"))
	b.logger.Info(fmt.Sprintf("Listening on interface: %s", b.config.Server.Host))
	b.logger.Info(fmt.Sprintf("Listening on port: %d", b.config.Server.Port))
	b.logger.Info(fmt.Sprintf("Full address: %s", address))
	b.logger.Info(fmt.Sprintf("Max clients: %d", b.config.Server.MaxClients))
	
	if b.config.Server.EnableStats {
		go b.printStats()
	}

	return nil
}

func (b *Broker) Stop() error {
	b.logger.Info("Stopping Realtek Embedded MQTT Broker...")
	return b.server.Close()
}

func (b *Broker) WaitForSignal() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh
}

func (b *Broker) printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║                  Realtek Embedded MQTT Broker                ║
║                                                               ║
║  High-performance MQTT 3.1.1 broker for RTK IoT devices     ║
║  Copyright (c) 2025 Realtek Semiconductor Corp.              ║
╚═══════════════════════════════════════════════════════════════╝`

	fmt.Println(banner)
}

func (b *Broker) printStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			info := b.server.Info
			b.logger.Info(fmt.Sprintf("Stats - Clients: %d, Messages Received: %d, Messages Sent: %d", 
				info.ClientsConnected, info.MessagesReceived, info.MessagesSent))
		}
	}
}