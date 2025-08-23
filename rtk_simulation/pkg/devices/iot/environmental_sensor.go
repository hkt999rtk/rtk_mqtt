package iot

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	// "github.com/sirupsen/logrus"
	// "rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices/base"
)

type EnvironmentalSensor struct {
	*base.BaseDevice

	temperature     float64
	humidity        float64
	airQuality      AirQualityData
	pressure        float64
	lightLevel      float64
	noiseLevel      float64
	calibration     CalibrationData
	alertThresholds AlertThresholds
	batteryLevel    float64
	powerSource     string
	mu              sync.RWMutex
}

type AirQualityData struct {
	PM25    float64 `json:"pm2_5"`
	PM10    float64 `json:"pm10"`
	CO2     float64 `json:"co2"`
	CO      float64 `json:"co"`
	VOC     float64 `json:"voc"`
	AQI     int     `json:"aqi"`
	Quality string  `json:"quality"`
}

type CalibrationData struct {
	LastCalibration time.Time          `json:"last_calibration"`
	CalibrationDue  time.Time          `json:"calibration_due"`
	Offsets         map[string]float64 `json:"offsets"`
	Multipliers     map[string]float64 `json:"multipliers"`
	Status          string             `json:"status"`
}

type AlertThresholds struct {
	Temperature struct {
		Min float64 `json:"min"`
		Max float64 `json:"max"`
	} `json:"temperature"`
	Humidity struct {
		Min float64 `json:"min"`
		Max float64 `json:"max"`
	} `json:"humidity"`
	AirQuality struct {
		PM25Max float64 `json:"pm2_5_max"`
		CO2Max  float64 `json:"co2_max"`
		AQIMax  int     `json:"aqi_max"`
	} `json:"air_quality"`
}

func NewEnvironmentalSensor(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*EnvironmentalSensor, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	sensor := &EnvironmentalSensor{
		BaseDevice:   baseDevice,
		temperature:  22.5,
		humidity:     45.0,
		pressure:     1013.25,
		lightLevel:   500.0,
		noiseLevel:   40.0,
		batteryLevel: 100.0,
		powerSource:  "battery",
	}

	sensor.initializeAirQuality()
	sensor.initializeCalibration()
	sensor.initializeThresholds()

	// 從配置中獲取電源類型
	if deviceConfig.Extra != nil {
		if ps, ok := deviceConfig.Extra["power_source"].(string); ok {
			// 驗證電源類型
			if ps == "battery" || ps == "ac" || ps == "solar" || ps == "usb" {
				sensor.powerSource = ps
			}
		}

		// 如果是電池供電，可以設定初始電量
		if sensor.powerSource == "battery" {
			if bl, ok := deviceConfig.Extra["battery_level"].(float64); ok {
				if bl >= 0 && bl <= 100 {
					sensor.batteryLevel = bl
				}
			}
		}
	}

	return sensor, nil
}

func (es *EnvironmentalSensor) initializeAirQuality() {
	es.airQuality = AirQualityData{
		PM25:    12.0,
		PM10:    20.0,
		CO2:     400.0,
		CO:      0.5,
		VOC:     50.0,
		AQI:     25,
		Quality: "Good",
	}
}

func (es *EnvironmentalSensor) initializeCalibration() {
	now := time.Now()
	es.calibration = CalibrationData{
		LastCalibration: now.AddDate(0, -3, 0),
		CalibrationDue:  now.AddDate(0, 3, 0),
		Offsets: map[string]float64{
			"temperature": 0.0,
			"humidity":    0.0,
			"pressure":    0.0,
		},
		Multipliers: map[string]float64{
			"temperature": 1.0,
			"humidity":    1.0,
			"pressure":    1.0,
		},
		Status: "valid",
	}
}

func (es *EnvironmentalSensor) initializeThresholds() {
	es.alertThresholds = AlertThresholds{
		Temperature: struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		}{Min: 10.0, Max: 35.0},
		Humidity: struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		}{Min: 30.0, Max: 70.0},
		AirQuality: struct {
			PM25Max float64 `json:"pm2_5_max"`
			CO2Max  float64 `json:"co2_max"`
			AQIMax  int     `json:"aqi_max"`
		}{PM25Max: 35.0, CO2Max: 1000.0, AQIMax: 100},
	}
}

func (es *EnvironmentalSensor) Start(ctx context.Context) error {
	if err := es.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go es.runSensorReading(ctx)
	go es.runBatteryMonitoring(ctx)
	go es.runCalibrationCheck(ctx)
	go es.runAlertMonitoring(ctx)

	// Environmental sensor started successfully
	return nil
}

func (es *EnvironmentalSensor) runSensorReading(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			es.updateSensorReadings()
		}
	}
}

func (es *EnvironmentalSensor) runBatteryMonitoring(ctx context.Context) {
	if es.powerSource != "battery" {
		return
	}

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			es.updateBatteryLevel()
		}
	}
}

func (es *EnvironmentalSensor) runCalibrationCheck(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			es.checkCalibrationStatus()
		}
	}
}

func (es *EnvironmentalSensor) runAlertMonitoring(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			es.checkAlertConditions()
		}
	}
}

func (es *EnvironmentalSensor) updateSensorReadings() {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.temperature = es.simulateTemperature()
	es.humidity = es.simulateHumidity()
	es.pressure = es.simulatePressure()
	es.lightLevel = es.simulateLightLevel()
	es.noiseLevel = es.simulateNoiseLevel()
	es.updateAirQuality()

	es.temperature = (es.temperature + es.calibration.Offsets["temperature"]) * es.calibration.Multipliers["temperature"]
	es.humidity = (es.humidity + es.calibration.Offsets["humidity"]) * es.calibration.Multipliers["humidity"]
	es.pressure = (es.pressure + es.calibration.Offsets["pressure"]) * es.calibration.Multipliers["pressure"]
}

func (es *EnvironmentalSensor) simulateTemperature() float64 {
	hour := time.Now().Hour()
	base := 22.0

	if hour >= 6 && hour <= 12 {
		base += float64(hour-6) * 0.8
	} else if hour > 12 && hour <= 18 {
		base += 4.8 - float64(hour-12)*0.3
	} else if hour > 18 && hour <= 24 {
		base += 3.0 - float64(hour-18)*0.5
	} else {
		base -= float64(6-hour) * 0.2
	}

	return base + es.getRandomFloat(-2, 2)
}

func (es *EnvironmentalSensor) simulateHumidity() float64 {
	base := 50.0
	tempEffect := (es.temperature - 22.0) * -1.5
	timeEffect := es.getRandomFloat(-10, 10)

	humidity := base + tempEffect + timeEffect
	if humidity < 20 {
		humidity = 20
	}
	if humidity > 90 {
		humidity = 90
	}

	return humidity
}

func (es *EnvironmentalSensor) simulatePressure() float64 {
	base := 1013.25
	weather := es.getRandomFloat(-15, 15)
	return base + weather
}

func (es *EnvironmentalSensor) simulateLightLevel() float64 {
	hour := time.Now().Hour()

	if hour >= 6 && hour <= 18 {
		peakHour := 12
		distance := float64(abs(hour - peakHour))
		maxLight := 1000.0
		return maxLight*(1.0-distance/12.0) + es.getRandomFloat(-50, 50)
	}

	return es.getRandomFloat(0, 50)
}

func (es *EnvironmentalSensor) simulateNoiseLevel() float64 {
	hour := time.Now().Hour()
	base := 35.0

	if hour >= 7 && hour <= 22 {
		base += 15.0
	}

	if hour >= 8 && hour <= 9 || hour >= 17 && hour <= 19 {
		base += 10.0
	}

	return base + es.getRandomFloat(-5, 15)
}

func (es *EnvironmentalSensor) updateAirQuality() {
	es.airQuality.PM25 = 8.0 + es.getRandomFloat(-3, 12)
	es.airQuality.PM10 = es.airQuality.PM25*1.6 + es.getRandomFloat(-5, 10)
	es.airQuality.CO2 = 400.0 + es.getRandomFloat(-50, 200)
	es.airQuality.CO = 0.3 + es.getRandomFloat(-0.2, 0.7)
	es.airQuality.VOC = 30.0 + es.getRandomFloat(-20, 100)

	es.airQuality.AQI = es.calculateAQI()
	es.airQuality.Quality = es.getAQICategory(es.airQuality.AQI)
}

func (es *EnvironmentalSensor) calculateAQI() int {
	pm25AQI := int(es.airQuality.PM25 * 2.5)
	co2AQI := int((es.airQuality.CO2 - 400) / 10)
	vocAQI := int(es.airQuality.VOC / 2)

	maxAQI := pm25AQI
	if co2AQI > maxAQI {
		maxAQI = co2AQI
	}
	if vocAQI > maxAQI {
		maxAQI = vocAQI
	}

	if maxAQI < 0 {
		maxAQI = 0
	}
	if maxAQI > 500 {
		maxAQI = 500
	}

	return maxAQI
}

func (es *EnvironmentalSensor) getAQICategory(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good"
	case aqi <= 100:
		return "Moderate"
	case aqi <= 150:
		return "Unhealthy for Sensitive Groups"
	case aqi <= 200:
		return "Unhealthy"
	case aqi <= 300:
		return "Very Unhealthy"
	default:
		return "Hazardous"
	}
}

func (es *EnvironmentalSensor) updateBatteryLevel() {
	es.mu.Lock()
	defer es.mu.Unlock()

	discharge := es.getRandomFloat(0.5, 2.0)
	es.batteryLevel -= discharge

	if es.batteryLevel <= 0 {
		es.batteryLevel = 0
		// Battery depleted - set offline status
	} else if es.batteryLevel <= 10 {
		event := base.Event{
			EventType: "battery_low",
			Severity:  "warning",
			Message:   fmt.Sprintf("Battery level critically low: %.1f%%", es.batteryLevel),
			Extra: map[string]interface{}{
				"battery_level": es.batteryLevel,
				"device_id":     es.GetDeviceID(),
				"timestamp":     time.Now(),
			},
		}
		es.PublishEvent(event)
	}
}

func (es *EnvironmentalSensor) checkCalibrationStatus() {
	es.mu.Lock()
	defer es.mu.Unlock()

	now := time.Now()
	if now.After(es.calibration.CalibrationDue) {
		es.calibration.Status = "due"

		event := base.Event{
			EventType: "calibration_due",
			Severity:  "warning",
			Message:   "Sensor calibration is due",
			Extra: map[string]interface{}{
				"last_calibration": es.calibration.LastCalibration,
				"calibration_due":  es.calibration.CalibrationDue,
				"timestamp":        now,
			},
		}
		es.PublishEvent(event)
	} else if now.After(es.calibration.CalibrationDue.AddDate(0, 1, 0)) {
		es.calibration.Status = "overdue"

		event := base.Event{
			EventType: "calibration_overdue",
			Severity:  "critical",
			Message:   "Sensor calibration is overdue - readings may be inaccurate",
			Extra: map[string]interface{}{
				"last_calibration": es.calibration.LastCalibration,
				"days_overdue":     int(now.Sub(es.calibration.CalibrationDue).Hours() / 24),
			},
		}
		es.PublishEvent(event)
	}
}

func (es *EnvironmentalSensor) checkAlertConditions() {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if es.temperature < es.alertThresholds.Temperature.Min || es.temperature > es.alertThresholds.Temperature.Max {
		event := base.Event{
			EventType: "temperature_alert",
			Severity:  "warning",
			Message:   fmt.Sprintf("Temperature out of range: %.1f°C", es.temperature),
			Extra: map[string]interface{}{
				"temperature":   es.temperature,
				"min_threshold": es.alertThresholds.Temperature.Min,
				"max_threshold": es.alertThresholds.Temperature.Max,
			},
		}
		es.PublishEvent(event)
	}

	if es.humidity < es.alertThresholds.Humidity.Min || es.humidity > es.alertThresholds.Humidity.Max {
		event := base.Event{
			EventType: "humidity_alert",
			Severity:  "warning",
			Message:   fmt.Sprintf("Humidity out of range: %.1f%%", es.humidity),
			Extra: map[string]interface{}{
				"humidity":      es.humidity,
				"min_threshold": es.alertThresholds.Humidity.Min,
				"max_threshold": es.alertThresholds.Humidity.Max,
			},
		}
		es.PublishEvent(event)
	}

	if es.airQuality.PM25 > es.alertThresholds.AirQuality.PM25Max {
		event := base.Event{
			EventType: "air_quality_alert",
			Severity:  "warning",
			Message:   fmt.Sprintf("PM2.5 levels elevated: %.1f μg/m³", es.airQuality.PM25),
			Extra: map[string]interface{}{
				"pm2_5":     es.airQuality.PM25,
				"threshold": es.alertThresholds.AirQuality.PM25Max,
				"aqi":       es.airQuality.AQI,
				"quality":   es.airQuality.Quality,
			},
		}
		es.PublishEvent(event)
	}

	if es.airQuality.CO2 > es.alertThresholds.AirQuality.CO2Max {
		event := base.Event{
			EventType: "co2_alert",
			Severity:  "warning",
			Message:   fmt.Sprintf("CO2 levels elevated: %.0f ppm", es.airQuality.CO2),
			Extra: map[string]interface{}{
				"co2":       es.airQuality.CO2,
				"threshold": es.alertThresholds.AirQuality.CO2Max,
			},
		}
		es.PublishEvent(event)
	}
}

func (es *EnvironmentalSensor) GenerateStatePayload() base.StatePayload {
	es.mu.RLock()
	defer es.mu.RUnlock()

	baseState := es.BaseDevice.GenerateStatePayload()

	baseState.Extra["sensors"] = map[string]interface{}{
		"temperature": es.temperature,
		"humidity":    es.humidity,
		"pressure":    es.pressure,
		"light_level": es.lightLevel,
		"noise_level": es.noiseLevel,
		"air_quality": es.airQuality,
	}

	if es.powerSource == "battery" {
		baseState.Extra["battery"] = map[string]interface{}{
			"level":        es.batteryLevel,
			"power_source": es.powerSource,
		}
	}

	baseState.Extra["calibration"] = map[string]interface{}{
		"status":           es.calibration.Status,
		"last_calibration": es.calibration.LastCalibration,
		"calibration_due":  es.calibration.CalibrationDue,
	}

	return baseState
}

func (es *EnvironmentalSensor) GenerateTelemetryData() map[string]base.TelemetryPayload {
	es.mu.RLock()
	defer es.mu.RUnlock()

	telemetry := es.BaseDevice.GenerateTelemetryData()

	telemetry["temperature"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     es.temperature,
		"unit":      "celsius",
		"tags": map[string]string{
			"sensor": "temperature",
		},
	}

	telemetry["humidity"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     es.humidity,
		"unit":      "percent",
		"tags": map[string]string{
			"sensor": "humidity",
		},
	}

	telemetry["pressure"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     es.pressure,
		"unit":      "hpa",
		"tags": map[string]string{
			"sensor": "pressure",
		},
	}

	telemetry["light_level"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     es.lightLevel,
		"unit":      "lux",
		"tags": map[string]string{
			"sensor": "light",
		},
	}

	telemetry["noise_level"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     es.noiseLevel,
		"unit":      "db",
		"tags": map[string]string{
			"sensor": "noise",
		},
	}

	telemetry["pm2_5"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     es.airQuality.PM25,
		"unit":      "ugm3",
		"tags": map[string]string{
			"sensor": "air_quality",
			"type":   "pm2_5",
		},
	}

	telemetry["co2"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     es.airQuality.CO2,
		"unit":      "ppm",
		"tags": map[string]string{
			"sensor": "air_quality",
			"type":   "co2",
		},
	}

	telemetry["aqi"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(es.airQuality.AQI),
		"unit":      "index",
		"tags": map[string]string{
			"sensor": "air_quality",
			"type":   "aqi",
		},
	}

	if es.powerSource == "battery" {
		telemetry["battery_level"] = base.TelemetryPayload{
			"timestamp": time.Now(),
			"value":     es.batteryLevel,
			"unit":      "percent",
			"tags": map[string]string{
				"sensor": "battery",
			},
		}
	}

	return telemetry
}

func (es *EnvironmentalSensor) HandleCommand(cmd base.Command) error {
	// Handling command

	switch cmd.Action {
	case "get_sensor_data":
		return es.sendSensorData(cmd)
	case "calibrate":
		return es.performCalibration(cmd)
	case "set_thresholds":
		return es.setThresholds(cmd)
	case "reset_calibration":
		return es.resetCalibration(cmd)
	default:
		return es.BaseDevice.HandleCommand(cmd)
	}
}

func (es *EnvironmentalSensor) sendSensorData(cmd base.Command) error {
	es.mu.RLock()
	sensorData := map[string]interface{}{
		"temperature":   es.temperature,
		"humidity":      es.humidity,
		"pressure":      es.pressure,
		"light_level":   es.lightLevel,
		"noise_level":   es.noiseLevel,
		"air_quality":   es.airQuality,
		"battery_level": es.batteryLevel,
		"calibration":   es.calibration,
	}
	es.mu.RUnlock()

	data, _ := json.Marshal(sensorData)
	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(data),
	}

	return es.PublishCommandResponse(response)
}

func (es *EnvironmentalSensor) performCalibration(cmd base.Command) error {
	var calibrationData struct {
		Offsets     map[string]float64 `json:"offsets"`
		Multipliers map[string]float64 `json:"multipliers"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &calibrationData); err != nil {
		return fmt.Errorf("invalid calibration data: %v", err)
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	now := time.Now()
	es.calibration.LastCalibration = now
	es.calibration.CalibrationDue = now.AddDate(0, 6, 0)
	es.calibration.Status = "valid"

	if calibrationData.Offsets != nil {
		es.calibration.Offsets = calibrationData.Offsets
	}
	if calibrationData.Multipliers != nil {
		es.calibration.Multipliers = calibrationData.Multipliers
	}

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Calibration completed successfully",
	}

	return es.PublishCommandResponse(response)
}

func (es *EnvironmentalSensor) setThresholds(cmd base.Command) error {
	var newThresholds AlertThresholds
	if err := json.Unmarshal([]byte(cmd.Payload), &newThresholds); err != nil {
		return fmt.Errorf("invalid threshold data: %v", err)
	}

	es.mu.Lock()
	es.alertThresholds = newThresholds
	es.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Alert thresholds updated",
	}

	return es.PublishCommandResponse(response)
}

func (es *EnvironmentalSensor) resetCalibration(cmd base.Command) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.calibration.Offsets = map[string]float64{
		"temperature": 0.0,
		"humidity":    0.0,
		"pressure":    0.0,
	}
	es.calibration.Multipliers = map[string]float64{
		"temperature": 1.0,
		"humidity":    1.0,
		"pressure":    1.0,
	}
	es.calibration.Status = "factory_default"

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Calibration reset to factory defaults",
	}

	return es.PublishCommandResponse(response)
}

// Helper functions
func (es *EnvironmentalSensor) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
