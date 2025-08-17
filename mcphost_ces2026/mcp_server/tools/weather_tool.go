package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Weather API structures
type WeatherData struct {
	Location    string        `json:"location"`
	Temperature Temperature   `json:"temperature"`
	Conditions  string        `json:"conditions"`
	Humidity    int           `json:"humidity"`
	WindSpeed   float64       `json:"wind_speed"`
	WindDir     string        `json:"wind_direction"`
	Pressure    float64       `json:"pressure"`
	Visibility  float64       `json:"visibility"`
	UVIndex     int           `json:"uv_index"`
	Forecast    []DayForecast `json:"forecast"`
	LastUpdated string        `json:"last_updated"`
}

type Temperature struct {
	Current   float64 `json:"current"`
	FeelsLike float64 `json:"feels_like"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Unit      string  `json:"unit"`
}

type DayForecast struct {
	Date          string      `json:"date"`
	Temperature   Temperature `json:"temperature"`
	Conditions    string      `json:"conditions"`
	Precipitation float64     `json:"precipitation"`
}

// WeatherAPI.com API response structures
type WeatherAPIResponse struct {
	Location LocationInfo `json:"location"`
	Current  CurrentInfo  `json:"current"`
	Forecast ForecastInfo `json:"forecast,omitempty"`
}

type LocationInfo struct {
	Name    string  `json:"name"`
	Country string  `json:"country"`
	Region  string  `json:"region"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

type CurrentInfo struct {
	TempC      float64       `json:"temp_c"`
	TempF      float64       `json:"temp_f"`
	Condition  ConditionInfo `json:"condition"`
	WindKph    float64       `json:"wind_kph"`
	WindMph    float64       `json:"wind_mph"`
	WindDir    string        `json:"wind_dir"`
	Humidity   int           `json:"humidity"`
	PressureMb float64       `json:"pressure_mb"`
	PressureIn float64       `json:"pressure_in"`
	Vis        float64       `json:"vis_km"`
	UV         float64       `json:"uv"`
	FeelsLikeC float64       `json:"feelslike_c"`
	FeelsLikeF float64       `json:"feelslike_f"`
}

type ConditionInfo struct {
	Text string `json:"text"`
	Code int    `json:"code"`
}

type ForecastInfo struct {
	ForecastDay []ForecastDay `json:"forecastday"`
}

type ForecastDay struct {
	Date string  `json:"date"`
	Day  DayInfo `json:"day"`
}

type DayInfo struct {
	MaxTempC  float64       `json:"maxtemp_c"`
	MaxTempF  float64       `json:"maxtemp_f"`
	MinTempC  float64       `json:"mintemp_c"`
	MinTempF  float64       `json:"mintemp_f"`
	Condition ConditionInfo `json:"condition"`
}

// AddWeatherTools adds weather-related tools to the MCP server
func AddWeatherTools(s *server.MCPServer) {
	log.Printf("🔧 Tool Registry: Starting to register weather tools...")
	
	// Get weather tool
	weatherTool := mcp.NewTool("get_weather",
		mcp.WithDescription("Get real-time weather information for any location worldwide using WeatherAPI.com"),
		mcp.WithString("location",
			mcp.Required(),
			mcp.Description("Location name (city, country, coordinates, IP address, or zipcode). Examples: 'Taipei', 'Tokyo, Japan', '37.7749,-122.4194', '10001'"),
		),
		mcp.WithString("units",
			mcp.Description("Temperature units: 'celsius' (default) or 'fahrenheit'"),
		),
		mcp.WithBoolean("include_forecast",
			mcp.Description("Whether to include 3-day weather forecast (default: true)"),
		),
		mcp.WithString("test_mode",
			mcp.Description("Test mode: 'normal', 'network_error', 'api_error', 'invalid_location', 'rate_limit', 'timeout'"),
		),
		mcp.WithBoolean("validate_input",
			mcp.Description("Whether to perform strict input validation (default: true)"),
		),
	)

	log.Printf("🔧 Tool Registry: Weather tool created")
	log.Printf("   └─ Name: get_weather")
	log.Printf("   └─ Description: Get real-time weather information for any location worldwide using WeatherAPI.com")
	log.Printf("   └─ Parameters: location (required), units (optional), include_forecast (optional)")
	log.Printf("   └─ API Integration: WeatherAPI.com (%s)", config.WeatherAPI.BaseURL)

	s.AddTool(weatherTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("🌤️  MCP Server: Received get_weather tool call request")

		location, err := request.RequireString("location")
		if err != nil {
			log.Printf("❌ MCP Server: get_weather parameter error - location parameter is required")
			return mcp.NewToolResultError("location parameter is required"), nil
		}

		units := request.GetString("units", "celsius")
		includeForecast := request.GetBool("include_forecast", true)
		testMode := request.GetString("test_mode", "normal")
		validateInput := request.GetBool("validate_input", true)

		log.Printf("📍 MCP Server: Processing weather query - location: %s, units: %s, include forecast: %t, test mode: %s", 
			location, units, includeForecast, testMode)

		// Perform input validation if enabled
		if validateInput {
			if validationErr := validateWeatherInput(location, units, testMode); validationErr != nil {
				log.Printf("❌ MCP Server: Weather input validation failed: %v", validationErr)
				return mcp.NewToolResultError(fmt.Sprintf("Input validation failed: %v", validationErr)), nil
			}
		}

		// Handle test modes
		if testMode != "normal" {
			testResult := handleWeatherTestMode(testMode, location, units, includeForecast)
			if testResult != nil {
				jsonData, _ := json.MarshalIndent(testResult, "", "  ")
				log.Printf("🧪 MCP Server: Weather test mode executed - mode: %s", testMode)
				return mcp.NewToolResultText(string(jsonData)), nil
			}
		}

		// Get real weather data from WeatherAPI.com
		weatherData, err := getWeatherData(location, units, includeForecast)
		if err != nil {
			log.Printf("❌ MCP Server: WeatherAPI call failed: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get weather data: %v", err)), nil
		}

		// Convert structured data to JSON string
		jsonData, err := json.MarshalIndent(weatherData, "", "  ")
		if err != nil {
			log.Printf("❌ MCP Server: Weather data serialization failed: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Weather data serialization failed: %v", err)), nil
		}

		log.Printf("✅ MCP Server: get_weather tool call successful, returning %d bytes of weather data", len(jsonData))
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	log.Printf("✅ Tool Registry: Successfully registered weather tool 'get_weather'")
	log.Printf("🔧 Tool Registry: Weather tools registration completed (1 tool registered)")
}

// getWeatherData calls WeatherAPI.com API to get real weather data
func getWeatherData(location, units string, includeForecast bool) (WeatherData, error) {
	// Build API URL
	url := fmt.Sprintf("%s/current.json?key=%s&q=%s&aqi=no",
		config.WeatherAPI.BaseURL, config.WeatherAPI.APIKey, location)

	if includeForecast {
		url = fmt.Sprintf("%s/forecast.json?key=%s&q=%s&days=3&aqi=no",
			config.WeatherAPI.BaseURL, config.WeatherAPI.APIKey, location)
	}

	// Make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return WeatherData{}, fmt.Errorf("failed to call WeatherAPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("WeatherAPI returned status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherData{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response
	var apiResponse WeatherAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return WeatherData{}, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to internal format
	return convertWeatherAPIToInternal(apiResponse, units, includeForecast), nil
}

// convertWeatherAPIToInternal converts WeatherAPI.com response to internal format
func convertWeatherAPIToInternal(apiResponse WeatherAPIResponse, units string, includeForecast bool) WeatherData {
	// Determine temperature unit
	var currentTemp, feelsLike, minTemp, maxTemp float64
	var tempUnit string

	if units == "fahrenheit" {
		currentTemp = apiResponse.Current.TempF
		feelsLike = apiResponse.Current.FeelsLikeF
		tempUnit = "°F"
	} else {
		currentTemp = apiResponse.Current.TempC
		feelsLike = apiResponse.Current.FeelsLikeC
		tempUnit = "°C"
	}

	// For current weather, min/max are the same as current
	minTemp = currentTemp
	maxTemp = currentTemp

	// If forecast is available, get min/max from today's forecast
	if includeForecast && len(apiResponse.Forecast.ForecastDay) > 0 {
		today := apiResponse.Forecast.ForecastDay[0]
		if units == "fahrenheit" {
			minTemp = today.Day.MinTempF
			maxTemp = today.Day.MaxTempF
		} else {
			minTemp = today.Day.MinTempC
			maxTemp = today.Day.MaxTempC
		}
	}

	// Build internal weather data
	weatherData := WeatherData{
		Location: fmt.Sprintf("%s, %s", apiResponse.Location.Name, apiResponse.Location.Country),
		Temperature: Temperature{
			Current:   currentTemp,
			FeelsLike: feelsLike,
			Min:       minTemp,
			Max:       maxTemp,
			Unit:      tempUnit,
		},
		Conditions:  apiResponse.Current.Condition.Text,
		Humidity:    apiResponse.Current.Humidity,
		WindSpeed:   apiResponse.Current.WindKph,
		WindDir:     apiResponse.Current.WindDir,
		Pressure:    apiResponse.Current.PressureMb,
		Visibility:  apiResponse.Current.Vis,
		UVIndex:     int(apiResponse.Current.UV),
		LastUpdated: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Add forecast if requested
	if includeForecast && len(apiResponse.Forecast.ForecastDay) > 0 {
		for _, day := range apiResponse.Forecast.ForecastDay {
			var dayMinTemp, dayMaxTemp float64
			if units == "fahrenheit" {
				dayMinTemp = day.Day.MinTempF
				dayMaxTemp = day.Day.MaxTempF
			} else {
				dayMinTemp = day.Day.MinTempC
				dayMaxTemp = day.Day.MaxTempC
			}

			forecast := DayForecast{
				Date: day.Date,
				Temperature: Temperature{
					Current:   (dayMinTemp + dayMaxTemp) / 2,
					FeelsLike: (dayMinTemp + dayMaxTemp) / 2,
					Min:       dayMinTemp,
					Max:       dayMaxTemp,
					Unit:      tempUnit,
				},
				Conditions:    day.Day.Condition.Text,
				Precipitation: 0.0, // WeatherAPI doesn't provide this in the free tier
			}
			weatherData.Forecast = append(weatherData.Forecast, forecast)
		}
	}

	return weatherData
}

// validateWeatherInput performs input validation for weather queries
func validateWeatherInput(location, units, testMode string) error {
	// Check for empty location
	if strings.TrimSpace(location) == "" {
		return fmt.Errorf("location cannot be empty")
	}

	// Check for dangerous patterns
	dangerousPatterns := []string{
		"../", "..\\", "/etc/", "C:\\", 
		"<script>", "</script>", "javascript:", 
		"'", "\"", "--", "DROP", "DELETE", "INSERT", "UPDATE", "UNION", "SELECT",
		"|", "&", ";", "$(", "`", "rm -rf", "format c:",
	}

	locationLower := strings.ToLower(location)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(locationLower, strings.ToLower(pattern)) {
			return fmt.Errorf("potentially dangerous input detected: contains '%s'", pattern)
		}
	}

	// Check location length (reasonable limits)
	if len(location) > 200 {
		return fmt.Errorf("location name too long (max 200 characters)")
	}

	// Validate units parameter
	if units != "celsius" && units != "fahrenheit" {
		return fmt.Errorf("invalid units: must be 'celsius' or 'fahrenheit'")
	}

	// Validate test mode
	validTestModes := []string{"normal", "network_error", "api_error", "invalid_location", "rate_limit", "timeout"}
	isValidTestMode := false
	for _, mode := range validTestModes {
		if testMode == mode {
			isValidTestMode = true
			break
		}
	}
	if !isValidTestMode {
		return fmt.Errorf("invalid test_mode: must be one of %v", validTestModes)
	}

	return nil
}

// handleWeatherTestMode handles different test modes for weather tool
func handleWeatherTestMode(testMode, location, units string, includeForecast bool) interface{} {
	switch testMode {
	case "network_error":
		return map[string]interface{}{
			"error": "Network timeout occurred",
			"code":  "NETWORK_ERROR",
			"message": "Simulated network error - connection timeout",
			"test_mode": testMode,
			"location": location,
			"recoverable": true,
		}
	
	case "api_error":
		return map[string]interface{}{
			"error": "WeatherAPI service unavailable",
			"code":  "API_ERROR",
			"message": "Simulated API error - service returned HTTP 503",
			"test_mode": testMode,
			"location": location,
			"recoverable": false,
		}
	
	case "invalid_location":
		return map[string]interface{}{
			"error": "Location not found",
			"code":  "LOCATION_NOT_FOUND",
			"message": fmt.Sprintf("Simulated invalid location error for: %s", location),
			"test_mode": testMode,
			"location": location,
			"recoverable": false,
		}
	
	case "rate_limit":
		return map[string]interface{}{
			"error": "API rate limit exceeded",
			"code":  "RATE_LIMIT",
			"message": "Simulated rate limit error - too many requests",
			"test_mode": testMode,
			"location": location,
			"retry_after": "60s",
			"recoverable": true,
		}
	
	case "timeout":
		// Simulate a timeout with delay
		time.Sleep(2 * time.Second)
		return map[string]interface{}{
			"error": "Request timeout",
			"code":  "TIMEOUT",
			"message": "Simulated timeout error - request took too long",
			"test_mode": testMode,
			"location": location,
			"timeout_duration": "2s",
			"recoverable": true,
		}
	
	default:
		return nil // Normal mode, proceed with real API call
	}
}