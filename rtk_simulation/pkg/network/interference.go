package network

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// InterferenceSimulator 信號干擾模擬器
type InterferenceSimulator struct {
	devices          map[string]*WirelessDevice
	interferenceMap  map[string]*InterferenceSource
	channels         map[string]*ChannelInfo
	environment      *EnvironmentModel
	propagationModel *PropagationModel
	running          bool
	mu               sync.RWMutex
	logger           *logrus.Entry
	config           *InterferenceConfig
}

// WirelessDevice 無線設備
type WirelessDevice struct {
	DeviceID       string
	Position       Position3D
	TransmitPower  float64 // dBm
	AntennaGain    float64 // dBi
	Frequency      float64 // MHz
	Channel        int
	Band           string  // 2.4GHz, 5GHz, 6GHz
	SignalStrength float64 // dBm
	NoiseFloor     float64 // dBm
	SNR            float64 // dB
	Interference   float64 // dBm
	ChannelWidth   int     // MHz (20, 40, 80, 160)
	MCS            int     // Modulation and Coding Scheme
	MaxDataRate    float64 // Mbps
	ActualDataRate float64 // Mbps
	Neighbors      []string
	LastUpdate     time.Time
}

// InterferenceSource 干擾源
type InterferenceSource struct {
	ID              string
	Type            string // microwave, bluetooth, zigbee, radar, etc.
	Position        Position3D
	Power           float64 // dBm
	Frequency       float64 // MHz
	Bandwidth       float64 // MHz
	Pattern         string  // continuous, periodic, burst, random
	DutyCycle       float64 // 0-1
	Active          bool
	StartTime       time.Time
	Duration        time.Duration
	AffectedDevices []string
}

// ChannelInfo 頻道資訊
type ChannelInfo struct {
	Channel        int
	Frequency      float64 // 中心頻率 MHz
	Band           string  // 2.4GHz, 5GHz, 6GHz
	Width          int     // MHz
	Utilization    float64 // 0-100%
	NoiseLevel     float64 // dBm
	DeviceCount    int
	Interference   float64         // dBm
	Quality        float64         // 0-100
	OverlapFactors map[int]float64 // 與其他頻道的重疊係數
}

// Position3D 3D 位置
type Position3D struct {
	X float64 // meters
	Y float64 // meters
	Z float64 // meters
}

// EnvironmentModel 環境模型
type EnvironmentModel struct {
	Type             string  // indoor, outdoor, urban, suburban, rural
	Temperature      float64 // Celsius
	Humidity         float64 // %
	Pressure         float64 // hPa
	WallAttenuation  float64 // dB per wall
	FloorAttenuation float64 // dB per floor
	Obstacles        []Obstacle
	Materials        map[string]float64 // 材質衰減係數
}

// Obstacle 障礙物
type Obstacle struct {
	Position    Position3D
	Size        Size3D
	Material    string
	Attenuation float64 // dB
}

// Size3D 3D 尺寸
type Size3D struct {
	Width  float64
	Height float64
	Depth  float64
}

// PropagationModel 傳播模型
type PropagationModel struct {
	Type              string // free-space, two-ray, log-distance, indoor
	PathLossExponent  float64
	ShadowingStdDev   float64
	ReferenceDistance float64
	ReferenceLoss     float64
}

// InterferenceConfig 干擾配置
type InterferenceConfig struct {
	UpdateInterval   time.Duration
	MaxInterference  float64 // dBm
	NoiseFloor       float64 // dBm
	ChannelOverlap   bool
	MultiPathFading  bool
	DopplerEffect    bool
	EnvironmentType  string
	PropagationModel string
}

// NewInterferenceSimulator 創建新的干擾模擬器
func NewInterferenceSimulator(config *InterferenceConfig) *InterferenceSimulator {
	if config == nil {
		config = &InterferenceConfig{
			UpdateInterval:   100 * time.Millisecond,
			MaxInterference:  -30, // dBm
			NoiseFloor:       -95, // dBm
			ChannelOverlap:   true,
			MultiPathFading:  true,
			DopplerEffect:    false,
			EnvironmentType:  "indoor",
			PropagationModel: "log-distance",
		}
	}

	is := &InterferenceSimulator{
		devices:         make(map[string]*WirelessDevice),
		interferenceMap: make(map[string]*InterferenceSource),
		channels:        make(map[string]*ChannelInfo),
		config:          config,
		logger:          logrus.WithField("component", "interference_simulator"),
	}

	// 初始化環境模型
	is.environment = is.createEnvironmentModel(config.EnvironmentType)

	// 初始化傳播模型
	is.propagationModel = is.createPropagationModel(config.PropagationModel)

	// 初始化頻道資訊
	is.initializeChannels()

	return is
}

// Start 啟動干擾模擬器
func (is *InterferenceSimulator) Start(ctx context.Context) error {
	is.mu.Lock()
	defer is.mu.Unlock()

	if is.running {
		return fmt.Errorf("interference simulator is already running")
	}

	is.running = true
	is.logger.Info("Starting interference simulator")

	// 啟動模擬循環
	go is.simulationLoop(ctx)
	go is.interferenceGenerationLoop(ctx)
	go is.channelScanLoop(ctx)

	return nil
}

// Stop 停止干擾模擬器
func (is *InterferenceSimulator) Stop() error {
	is.mu.Lock()
	defer is.mu.Unlock()

	if !is.running {
		return fmt.Errorf("interference simulator is not running")
	}

	is.running = false
	is.logger.Info("Stopping interference simulator")
	return nil
}

// RegisterWirelessDevice 註冊無線設備
func (is *InterferenceSimulator) RegisterWirelessDevice(deviceID string, position Position3D, frequency float64, channel int, band string, power float64) {
	is.mu.Lock()
	defer is.mu.Unlock()

	device := &WirelessDevice{
		DeviceID:       deviceID,
		Position:       position,
		TransmitPower:  power,
		AntennaGain:    2.0, // 預設 2 dBi
		Frequency:      frequency,
		Channel:        channel,
		Band:           band,
		SignalStrength: power,
		NoiseFloor:     is.config.NoiseFloor,
		ChannelWidth:   20, // 預設 20 MHz
		Neighbors:      make([]string, 0),
		LastUpdate:     time.Now(),
	}

	// 計算初始 SNR
	device.SNR = device.SignalStrength - device.NoiseFloor

	// 設定最大數據率
	device.MaxDataRate = is.calculateMaxDataRate(device)

	is.devices[deviceID] = device

	is.logger.WithFields(logrus.Fields{
		"device_id": deviceID,
		"frequency": frequency,
		"channel":   channel,
		"band":      band,
	}).Info("Wireless device registered")
}

// AddInterferenceSource 添加干擾源
func (is *InterferenceSimulator) AddInterferenceSource(sourceType string, position Position3D, power float64, frequency float64) string {
	is.mu.Lock()
	defer is.mu.Unlock()

	sourceID := fmt.Sprintf("interference_%s_%d", sourceType, time.Now().UnixNano())

	source := &InterferenceSource{
		ID:              sourceID,
		Type:            sourceType,
		Position:        position,
		Power:           power,
		Frequency:       frequency,
		Active:          true,
		StartTime:       time.Now(),
		AffectedDevices: make([]string, 0),
	}

	// 設定干擾源特性
	switch sourceType {
	case "microwave":
		source.Bandwidth = 20
		source.Pattern = "periodic"
		source.DutyCycle = 0.5
	case "bluetooth":
		source.Bandwidth = 1
		source.Pattern = "burst"
		source.DutyCycle = 0.1
	case "zigbee":
		source.Bandwidth = 2
		source.Pattern = "continuous"
		source.DutyCycle = 0.3
	case "radar":
		source.Bandwidth = 10
		source.Pattern = "periodic"
		source.DutyCycle = 0.2
	default:
		source.Bandwidth = 5
		source.Pattern = "random"
		source.DutyCycle = 0.3
	}

	is.interferenceMap[sourceID] = source

	is.logger.WithFields(logrus.Fields{
		"source_id": sourceID,
		"type":      sourceType,
		"frequency": frequency,
		"power":     power,
	}).Info("Interference source added")

	return sourceID
}

// simulationLoop 主模擬循環
func (is *InterferenceSimulator) simulationLoop(ctx context.Context) {
	ticker := time.NewTicker(is.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			is.updateInterference()
			is.updateSignalQuality()
			is.updateDataRates()
		}
	}
}

// updateInterference 更新干擾
func (is *InterferenceSimulator) updateInterference() {
	is.mu.Lock()
	defer is.mu.Unlock()

	// 對每個設備計算干擾
	for deviceID, device := range is.devices {
		totalInterference := is.config.NoiseFloor

		// 計算來自其他設備的干擾
		for otherID, other := range is.devices {
			if otherID == deviceID {
				continue
			}

			// 檢查頻道重疊
			overlap := is.calculateChannelOverlap(device.Channel, other.Channel, device.Band)
			if overlap > 0 {
				// 計算路徑損耗
				distance := is.calculateDistance(device.Position, other.Position)
				pathLoss := is.calculatePathLoss(distance, device.Frequency)

				// 計算接收功率
				rxPower := other.TransmitPower + other.AntennaGain - pathLoss

				// 根據重疊係數調整干擾
				interference := rxPower * overlap

				// 累加干擾（轉換為線性後相加）
				totalInterference = is.addPowerDBm(totalInterference, interference)
			}
		}

		// 計算來自干擾源的干擾
		for _, source := range is.interferenceMap {
			if !source.Active {
				continue
			}

			// 檢查頻率重疊
			if is.frequencyOverlap(device.Frequency, float64(device.ChannelWidth), source.Frequency, source.Bandwidth) {
				distance := is.calculateDistance(device.Position, source.Position)
				pathLoss := is.calculatePathLoss(distance, source.Frequency)

				// 考慮占空比
				effectivePower := source.Power
				if source.Pattern != "continuous" {
					effectivePower = source.Power + 10*math.Log10(source.DutyCycle)
				}

				rxPower := effectivePower - pathLoss
				totalInterference = is.addPowerDBm(totalInterference, rxPower)

				// 記錄受影響的設備
				if !is.contains(source.AffectedDevices, deviceID) {
					source.AffectedDevices = append(source.AffectedDevices, deviceID)
				}
			}
		}

		device.Interference = totalInterference
	}
}

// updateSignalQuality 更新信號質量
func (is *InterferenceSimulator) updateSignalQuality() {
	is.mu.RLock()
	defer is.mu.RUnlock()

	for _, device := range is.devices {
		// 計算 SINR (Signal to Interference plus Noise Ratio)
		noise := is.addPowerDBm(device.NoiseFloor, device.Interference)
		device.SNR = device.SignalStrength - noise

		// 多徑衰落效應
		if is.config.MultiPathFading {
			fading := is.calculateMultiPathFading()
			device.SNR += fading
		}

		// 都卜勒效應
		if is.config.DopplerEffect {
			doppler := is.calculateDopplerShift(device)
			device.SNR += doppler
		}

		// 限制 SNR 範圍
		if device.SNR < -10 {
			device.SNR = -10
		} else if device.SNR > 50 {
			device.SNR = 50
		}

		device.LastUpdate = time.Now()
	}
}

// updateDataRates 更新數據率
func (is *InterferenceSimulator) updateDataRates() {
	is.mu.RLock()
	defer is.mu.RUnlock()

	for _, device := range is.devices {
		// 根據 SNR 選擇 MCS
		device.MCS = is.selectMCS(device.SNR, device.Band)

		// 計算實際數據率
		baseRate := is.getMCSDataRate(device.MCS, device.ChannelWidth)

		// 考慮干擾的影響
		efficiency := is.calculateEfficiency(device.SNR)

		device.ActualDataRate = baseRate * efficiency

		// 限制在最大數據率內
		if device.ActualDataRate > device.MaxDataRate {
			device.ActualDataRate = device.MaxDataRate
		}
	}
}

// interferenceGenerationLoop 干擾生成循環
func (is *InterferenceSimulator) interferenceGenerationLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			is.generateRandomInterference()
		}
	}
}

// generateRandomInterference 生成隨機干擾
func (is *InterferenceSimulator) generateRandomInterference() {
	// 隨機生成一些干擾源
	if rand.Float64() > 0.7 {
		interferenceTypes := []string{"microwave", "bluetooth", "zigbee", "unknown"}
		sourceType := interferenceTypes[rand.Intn(len(interferenceTypes))]

		position := Position3D{
			X: rand.Float64() * 50,
			Y: rand.Float64() * 50,
			Z: rand.Float64() * 3,
		}

		power := -50 + rand.Float64()*30       // -50 to -20 dBm
		frequency := 2400 + rand.Float64()*100 // 2.4-2.5 GHz

		is.AddInterferenceSource(sourceType, position, power, frequency)
	}
}

// channelScanLoop 頻道掃描循環
func (is *InterferenceSimulator) channelScanLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			is.scanChannels()
		}
	}
}

// scanChannels 掃描頻道
func (is *InterferenceSimulator) scanChannels() {
	is.mu.Lock()
	defer is.mu.Unlock()

	// 更新每個頻道的資訊
	for _, channel := range is.channels {
		channel.DeviceCount = 0
		channel.Utilization = 0
		channel.Interference = is.config.NoiseFloor

		// 統計設備數量和利用率
		for _, device := range is.devices {
			if device.Channel == channel.Channel && device.Band == channel.Band {
				channel.DeviceCount++
				channel.Utilization += 10 // 每個設備貢獻 10% 利用率
			}
		}

		// 計算頻道干擾
		for _, source := range is.interferenceMap {
			if source.Active && is.frequencyOverlap(channel.Frequency, float64(channel.Width), source.Frequency, source.Bandwidth) {
				channel.Interference = is.addPowerDBm(channel.Interference, source.Power)
			}
		}

		// 計算頻道質量
		channel.Quality = 100 - channel.Utilization
		if channel.Interference > -70 {
			channel.Quality *= (1 - (channel.Interference+70)/30)
		}

		if channel.Quality < 0 {
			channel.Quality = 0
		}
	}
}

// initializeChannels 初始化頻道
func (is *InterferenceSimulator) initializeChannels() {
	// 2.4GHz 頻道
	for i := 1; i <= 14; i++ {
		freq := 2412.0 + float64(i-1)*5
		if i == 14 {
			freq = 2484.0
		}

		channelKey := fmt.Sprintf("2.4GHz_ch%d", i)
		is.channels[channelKey] = &ChannelInfo{
			Channel:        i,
			Frequency:      freq,
			Band:           "2.4GHz",
			Width:          20,
			NoiseLevel:     is.config.NoiseFloor,
			OverlapFactors: is.calculate24GHzOverlap(i),
		}
	}

	// 5GHz 頻道 (部分)
	channels5GHz := []int{36, 40, 44, 48, 52, 56, 60, 64, 100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 149, 153, 157, 161, 165}
	for _, ch := range channels5GHz {
		freq := 5000.0 + float64(ch)*5

		channelKey := fmt.Sprintf("5GHz_ch%d", ch)
		is.channels[channelKey] = &ChannelInfo{
			Channel:        ch,
			Frequency:      freq,
			Band:           "5GHz",
			Width:          20,
			NoiseLevel:     is.config.NoiseFloor,
			OverlapFactors: make(map[int]float64),
		}
	}
}

// 輔助函數

// calculateDistance 計算距離
func (is *InterferenceSimulator) calculateDistance(p1, p2 Position3D) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	dz := p1.Z - p2.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// calculatePathLoss 計算路徑損耗
func (is *InterferenceSimulator) calculatePathLoss(distance, frequency float64) float64 {
	if distance < 0.01 {
		distance = 0.01 // 最小距離 1cm
	}

	switch is.propagationModel.Type {
	case "free-space":
		// 自由空間路徑損耗
		return 20*math.Log10(distance) + 20*math.Log10(frequency) - 27.55

	case "log-distance":
		// 對數距離路徑損耗
		if distance < is.propagationModel.ReferenceDistance {
			distance = is.propagationModel.ReferenceDistance
		}
		pl := is.propagationModel.ReferenceLoss +
			10*is.propagationModel.PathLossExponent*math.Log10(distance/is.propagationModel.ReferenceDistance)

		// 添加陰影衰落
		if is.propagationModel.ShadowingStdDev > 0 {
			pl += rand.NormFloat64() * is.propagationModel.ShadowingStdDev
		}
		return pl

	case "indoor":
		// 室內模型
		pl := 20*math.Log10(frequency) + 20*math.Log10(distance) + 32.45

		// 添加牆壁衰減
		numWalls := int(distance / 10) // 假設每 10 米一堵牆
		pl += float64(numWalls) * is.environment.WallAttenuation

		return pl

	default:
		// 預設使用自由空間模型
		return 20*math.Log10(distance) + 20*math.Log10(frequency) - 27.55
	}
}

// calculateChannelOverlap 計算頻道重疊
func (is *InterferenceSimulator) calculateChannelOverlap(ch1, ch2 int, band string) float64 {
	if !is.config.ChannelOverlap {
		if ch1 == ch2 {
			return 1.0
		}
		return 0
	}

	if band == "2.4GHz" {
		// 2.4GHz 頻道重疊計算
		diff := abs(ch1 - ch2)
		switch diff {
		case 0:
			return 1.0
		case 1:
			return 0.7
		case 2:
			return 0.4
		case 3:
			return 0.2
		case 4:
			return 0.1
		default:
			return 0
		}
	} else if band == "5GHz" {
		// 5GHz 頻道通常不重疊
		if ch1 == ch2 {
			return 1.0
		}
		return 0
	}

	return 0
}

// calculate24GHzOverlap 計算 2.4GHz 頻道重疊係數
func (is *InterferenceSimulator) calculate24GHzOverlap(channel int) map[int]float64 {
	overlap := make(map[int]float64)
	for i := 1; i <= 14; i++ {
		if i == channel {
			overlap[i] = 1.0
		} else {
			diff := abs(channel - i)
			switch diff {
			case 1:
				overlap[i] = 0.7
			case 2:
				overlap[i] = 0.4
			case 3:
				overlap[i] = 0.2
			case 4:
				overlap[i] = 0.1
			default:
				overlap[i] = 0
			}
		}
	}
	return overlap
}

// frequencyOverlap 檢查頻率重疊
func (is *InterferenceSimulator) frequencyOverlap(f1, bw1, f2, bw2 float64) bool {
	lower1 := f1 - bw1/2
	upper1 := f1 + bw1/2
	lower2 := f2 - bw2/2
	upper2 := f2 + bw2/2

	return !(upper1 < lower2 || upper2 < lower1)
}

// addPowerDBm 添加功率（dBm）
func (is *InterferenceSimulator) addPowerDBm(p1, p2 float64) float64 {
	// 轉換為線性功率
	linear1 := math.Pow(10, p1/10)
	linear2 := math.Pow(10, p2/10)

	// 相加後轉回 dBm
	return 10 * math.Log10(linear1+linear2)
}

// calculateMultiPathFading 計算多徑衰落
func (is *InterferenceSimulator) calculateMultiPathFading() float64 {
	// 簡化的瑞利衰落模型
	return rand.NormFloat64() * 3 // ±3 dB 變化
}

// calculateDopplerShift 計算都卜勒頻移
func (is *InterferenceSimulator) calculateDopplerShift(device *WirelessDevice) float64 {
	// 簡化模型，假設低速移動
	velocity := rand.Float64() * 2 // 0-2 m/s
	c := 3e8                       // 光速
	doppler := (velocity / c) * device.Frequency * 1e6
	return doppler / 1e6 // 轉換回 MHz 影響很小
}

// selectMCS 選擇調製編碼方案
func (is *InterferenceSimulator) selectMCS(snr float64, band string) int {
	// 基於 SNR 選擇 MCS
	if band == "2.4GHz" || band == "5GHz" {
		switch {
		case snr < 5:
			return 0
		case snr < 10:
			return 1
		case snr < 15:
			return 2
		case snr < 20:
			return 3
		case snr < 25:
			return 4
		case snr < 30:
			return 5
		case snr < 35:
			return 6
		case snr < 40:
			return 7
		case snr < 45:
			return 8
		default:
			return 9
		}
	}
	return 0
}

// getMCSDataRate 獲取 MCS 數據率
func (is *InterferenceSimulator) getMCSDataRate(mcs, channelWidth int) float64 {
	// 802.11n/ac MCS 數據率表（簡化）
	baseRates := []float64{6.5, 13, 19.5, 26, 39, 52, 58.5, 65, 78, 86.7}

	if mcs >= len(baseRates) {
		mcs = len(baseRates) - 1
	}

	rate := baseRates[mcs]

	// 根據頻道寬度調整
	switch channelWidth {
	case 40:
		rate *= 2.1
	case 80:
		rate *= 4.5
	case 160:
		rate *= 9.0
	}

	return rate
}

// calculateMaxDataRate 計算最大數據率
func (is *InterferenceSimulator) calculateMaxDataRate(device *WirelessDevice) float64 {
	// 根據頻段和頻道寬度計算理論最大數據率
	if device.Band == "2.4GHz" {
		switch device.ChannelWidth {
		case 20:
			return 72.2
		case 40:
			return 150
		default:
			return 72.2
		}
	} else if device.Band == "5GHz" {
		switch device.ChannelWidth {
		case 20:
			return 86.7
		case 40:
			return 200
		case 80:
			return 433.3
		case 160:
			return 866.7
		default:
			return 86.7
		}
	}
	return 54 // 預設 802.11g
}

// calculateEfficiency 計算效率
func (is *InterferenceSimulator) calculateEfficiency(snr float64) float64 {
	// 基於 SNR 的傳輸效率
	if snr < 0 {
		return 0.1
	} else if snr < 10 {
		return 0.3
	} else if snr < 20 {
		return 0.5
	} else if snr < 30 {
		return 0.7
	} else if snr < 40 {
		return 0.85
	} else {
		return 0.95
	}
}

// createEnvironmentModel 創建環境模型
func (is *InterferenceSimulator) createEnvironmentModel(envType string) *EnvironmentModel {
	model := &EnvironmentModel{
		Type:        envType,
		Temperature: 20,
		Humidity:    50,
		Pressure:    1013,
		Obstacles:   make([]Obstacle, 0),
		Materials:   make(map[string]float64),
	}

	switch envType {
	case "indoor":
		model.WallAttenuation = 5
		model.FloorAttenuation = 15
		model.Materials["concrete"] = 10
		model.Materials["wood"] = 3
		model.Materials["glass"] = 2
		model.Materials["metal"] = 20

	case "outdoor":
		model.WallAttenuation = 0
		model.FloorAttenuation = 0
		model.Materials["vegetation"] = 3
		model.Materials["rain"] = 0.1

	case "urban":
		model.WallAttenuation = 8
		model.FloorAttenuation = 18
		model.Materials["concrete"] = 12
		model.Materials["steel"] = 25

	default:
		model.WallAttenuation = 3
		model.FloorAttenuation = 10
	}

	return model
}

// createPropagationModel 創建傳播模型
func (is *InterferenceSimulator) createPropagationModel(modelType string) *PropagationModel {
	model := &PropagationModel{
		Type: modelType,
	}

	switch modelType {
	case "free-space":
		model.PathLossExponent = 2.0
		model.ShadowingStdDev = 0

	case "log-distance":
		model.PathLossExponent = 3.0
		model.ShadowingStdDev = 8.0
		model.ReferenceDistance = 1.0
		model.ReferenceLoss = 40.0

	case "indoor":
		model.PathLossExponent = 3.5
		model.ShadowingStdDev = 7.0
		model.ReferenceDistance = 1.0
		model.ReferenceLoss = 40.0

	case "two-ray":
		model.PathLossExponent = 4.0
		model.ShadowingStdDev = 6.0
		model.ReferenceDistance = 1.0
		model.ReferenceLoss = 40.0

	default:
		model.PathLossExponent = 2.5
		model.ShadowingStdDev = 4.0
		model.ReferenceDistance = 1.0
		model.ReferenceLoss = 40.0
	}

	return model
}

// contains 檢查切片是否包含元素
func (is *InterferenceSimulator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// abs 絕對值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GetDeviceSignalQuality 獲取設備信號質量
func (is *InterferenceSimulator) GetDeviceSignalQuality(deviceID string) (*WirelessDevice, error) {
	is.mu.RLock()
	defer is.mu.RUnlock()

	device, exists := is.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}

	// 返回副本
	return &WirelessDevice{
		DeviceID:       device.DeviceID,
		Position:       device.Position,
		TransmitPower:  device.TransmitPower,
		AntennaGain:    device.AntennaGain,
		Frequency:      device.Frequency,
		Channel:        device.Channel,
		Band:           device.Band,
		SignalStrength: device.SignalStrength,
		NoiseFloor:     device.NoiseFloor,
		SNR:            device.SNR,
		Interference:   device.Interference,
		ChannelWidth:   device.ChannelWidth,
		MCS:            device.MCS,
		MaxDataRate:    device.MaxDataRate,
		ActualDataRate: device.ActualDataRate,
		Neighbors:      append([]string{}, device.Neighbors...),
		LastUpdate:     device.LastUpdate,
	}, nil
}

// GetChannelInfo 獲取頻道資訊
func (is *InterferenceSimulator) GetChannelInfo(band string, channel int) (*ChannelInfo, error) {
	is.mu.RLock()
	defer is.mu.RUnlock()

	channelKey := fmt.Sprintf("%s_ch%d", band, channel)
	info, exists := is.channels[channelKey]
	if !exists {
		return nil, fmt.Errorf("channel %d in band %s not found", channel, band)
	}

	// 返回副本
	return &ChannelInfo{
		Channel:        info.Channel,
		Frequency:      info.Frequency,
		Band:           info.Band,
		Width:          info.Width,
		Utilization:    info.Utilization,
		NoiseLevel:     info.NoiseLevel,
		DeviceCount:    info.DeviceCount,
		Interference:   info.Interference,
		Quality:        info.Quality,
		OverlapFactors: info.OverlapFactors,
	}, nil
}

// OptimizeChannel 優化頻道選擇
func (is *InterferenceSimulator) OptimizeChannel(deviceID string) (int, error) {
	is.mu.Lock()
	defer is.mu.Unlock()

	device, exists := is.devices[deviceID]
	if !exists {
		return 0, fmt.Errorf("device %s not found", deviceID)
	}

	bestChannel := device.Channel
	bestQuality := 0.0

	// 搜尋最佳頻道
	for _, channel := range is.channels {
		if channel.Band != device.Band {
			continue
		}

		if channel.Quality > bestQuality {
			bestQuality = channel.Quality
			bestChannel = channel.Channel
		}
	}

	// 切換到最佳頻道
	if bestChannel != device.Channel {
		device.Channel = bestChannel
		for _, ch := range is.channels {
			if ch.Channel == bestChannel && ch.Band == device.Band {
				device.Frequency = ch.Frequency
				break
			}
		}

		is.logger.WithFields(logrus.Fields{
			"device_id":   deviceID,
			"new_channel": bestChannel,
			"quality":     bestQuality,
		}).Info("Channel optimized")
	}

	return bestChannel, nil
}
