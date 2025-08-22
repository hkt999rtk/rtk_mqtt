# 構建系統整合指南

## Makefile 擴展

### 添加 Wrapper 建置支援

修改主要的 `Makefile` 以支援 wrapper 建置：

```makefile
# 添加到現有的 Makefile 中

# Wrapper 相關目標
WRAPPER_BINARY := build_dir/rtk_wrapper
WRAPPER_SRC := wrapper/cmd/*.go wrapper/internal/**/*.go wrapper/pkg/**/*.go wrapper/wrappers/**/*.go

.PHONY: build-wrapper clean-wrapper test-wrapper run-wrapper

# 建置 wrapper
build-wrapper:
	@echo "Building wrapper..."
	@mkdir -p build_dir
	cd wrapper && go build -o ../$(WRAPPER_BINARY) ./cmd

# 跨平台編譯 wrapper
build-wrapper-all: clean-wrapper
	@echo "Building wrapper for all platforms..."
	@mkdir -p dist
	cd wrapper && \
	GOOS=linux GOARCH=amd64 go build -o ../dist/rtk_wrapper-linux-amd64 ./cmd && \
	GOOS=linux GOARCH=arm64 go build -o ../dist/rtk_wrapper-linux-arm64 ./cmd && \
	GOOS=darwin GOARCH=amd64 go build -o ../dist/rtk_wrapper-darwin-amd64 ./cmd && \
	GOOS=darwin GOARCH=arm64 go build -o ../dist/rtk_wrapper-darwin-arm64 ./cmd && \
	GOOS=windows GOARCH=amd64 go build -o ../dist/rtk_wrapper-windows-amd64.exe ./cmd

# 清理 wrapper 建置產物
clean-wrapper:
	@echo "Cleaning wrapper build artifacts..."
	@rm -f $(WRAPPER_BINARY)
	@rm -f dist/rtk_wrapper-*

# 測試 wrapper
test-wrapper:
	@echo "Running wrapper tests..."
	cd wrapper && go test -v -race -coverprofile=../coverage/wrapper.out ./...

# 運行 wrapper（開發模式）
run-wrapper: build-wrapper
	@echo "Running wrapper in development mode..."
	$(WRAPPER_BINARY) --config wrapper/configs/wrapper.yaml

# 示範 wrapper
demo-wrapper: build-wrapper
	@echo "Running wrapper demo..."
	./wrapper/demo_wrapper.sh

# 修改主要建置目標以包含 wrapper
build-all: build build-wrapper

clean: clean-build clean-wrapper

test: test-build test-wrapper
```

### RTK Controller 主程序整合

修改 `cmd/controller/main.go` 添加 wrapper 模式支援：

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	
	// 現有導入...
	"rtk_controller/internal/config"
	"rtk_controller/internal/mqtt"
	"rtk_wrapper/internal/wrapper"
)

var (
	wrapperMode bool
	hybridMode  bool
	wrapperConfig string
)

func init() {
	// 添加 wrapper 相關 flags
	rootCmd.PersistentFlags().BoolVar(&wrapperMode, "wrapper", false, "Run in wrapper mode only")
	rootCmd.PersistentFlags().BoolVar(&hybridMode, "hybrid", false, "Run in hybrid mode (controller + wrapper)")
	rootCmd.PersistentFlags().StringVar(&wrapperConfig, "wrapper-config", "wrapper/configs/wrapper.yaml", "Wrapper configuration file")
}

var rootCmd = &cobra.Command{
	Use:   "rtk_controller",
	Short: "RTK MQTT Controller with optional wrapper support",
	Run:   run,
}

func run(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 處理信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if wrapperMode {
		// 純 wrapper 模式
		runWrapperMode(ctx)
	} else if hybridMode {
		// 混合模式（同時運行 controller 和 wrapper）
		runHybridMode(ctx)
	} else {
		// 原本的 controller 模式
		runControllerMode(ctx)
	}

	// 等待信號
	<-sigChan
	fmt.Println("Shutting down...")
	cancel()
}

func runWrapperMode(ctx context.Context) {
	fmt.Println("Starting RTK Controller in wrapper mode...")
	
	// 載入 wrapper 配置
	wrapperMgr, err := wrapper.NewManager(wrapperConfig)
	if err != nil {
		fmt.Printf("Failed to create wrapper manager: %v\n", err)
		os.Exit(1)
	}

	// 啟動 wrapper
	if err := wrapperMgr.Start(ctx); err != nil {
		fmt.Printf("Failed to start wrapper: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Wrapper started successfully")
	<-ctx.Done()
	
	fmt.Println("Stopping wrapper...")
	wrapperMgr.Stop()
}

func runHybridMode(ctx context.Context) {
	fmt.Println("Starting RTK Controller in hybrid mode...")
	
	// 啟動 controller
	go runControllerMode(ctx)
	
	// 啟動 wrapper
	go runWrapperMode(ctx)
	
	<-ctx.Done()
}

func runControllerMode(ctx context.Context) {
	// 原本的 controller 邏輯
	fmt.Println("Starting RTK Controller...")
	// ... 現有的 controller 啟動邏輯
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
```

## 獨立 Wrapper 程序

創建獨立的 wrapper 可執行程序 `wrapper/cmd/main.go`：

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"rtk_wrapper/internal/config"
	"rtk_wrapper/internal/wrapper"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "configs/wrapper.yaml", "Wrapper configuration file")
	flag.Parse()

	// 載入配置
	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 創建 wrapper manager
	manager, err := wrapper.NewManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create wrapper manager: %v", err)
	}

	// 設置信號處理
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Received shutdown signal...")
		cancel()
	}()

	// 啟動 wrapper
	fmt.Println("Starting RTK MQTT Wrapper...")
	if err := manager.Start(ctx); err != nil {
		log.Fatalf("Failed to start wrapper: %v", err)
	}

	// 等待關閉
	<-ctx.Done()
	fmt.Println("Shutting down wrapper...")
	manager.Stop()
	fmt.Println("Wrapper stopped")
}
```

## 跨平台編譯支援

### 編譯腳本

創建 `wrapper/build.sh` 編譯腳本：

```bash
#!/bin/bash

set -e

# 設定
BINARY_NAME="rtk_wrapper"
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse HEAD)

# Build flags
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

echo "Building RTK MQTT Wrapper..."
echo "Version: ${VERSION}"
echo "Build Time: ${BUILD_TIME}"
echo "Git Commit: ${GIT_COMMIT}"

# 清理舊的建置產物
rm -rf dist/
mkdir -p dist/

# 平台列表
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64" 
    "windows/amd64"
)

# 編譯各平台
for platform in "${platforms[@]}"; do
    IFS='/' read -r -a array <<< "$platform"
    GOOS="${array[0]}"
    GOARCH="${array[1]}"
    
    output_name="${BINARY_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo "Building for ${GOOS}/${GOARCH}..."
    env GOOS="$GOOS" GOARCH="$GOARCH" go build \
        -ldflags="$LDFLAGS" \
        -o "dist/${output_name}" \
        ./cmd
        
    echo "Built: dist/${output_name}"
done

echo "Build completed successfully!"
echo "Binaries are in the dist/ directory:"
ls -la dist/
```

### 發行版本整合

修改主要的 `release.sh` 以包含 wrapper：

```bash
#!/bin/bash

# 原有的 release 腳本內容...

echo "Building wrapper components..."

# 建置 wrapper
cd wrapper
./build.sh
cd ..

# 複製 wrapper 到 release 目錄
cp -r wrapper/dist/* rtk_controller_release/bin/
cp wrapper/configs/* rtk_controller_release/configs/
cp wrapper/README.md rtk_controller_release/docs/WRAPPER.md

echo "Wrapper components added to release"
```

## 測試整合

### 建置驗證腳本

創建 `scripts/verify_build.sh`：

```bash
#!/bin/bash

set -e

echo "Verifying RTK Controller build integration..."

# 測試主程序建置
echo "Testing main controller build..."
make build
if [ ! -f "build_dir/rtk_controller" ]; then
    echo "ERROR: Controller binary not found"
    exit 1
fi

# 測試 wrapper 建置  
echo "Testing wrapper build..."
make build-wrapper
if [ ! -f "build_dir/rtk_wrapper" ]; then
    echo "ERROR: Wrapper binary not found"
    exit 1
fi

# 測試跨平台編譯
echo "Testing cross-platform builds..."
make build-all
if [ ! -d "dist" ]; then
    echo "ERROR: Distribution directory not found"
    exit 1
fi

# 檢查平台特定的二進位檔案
platforms=("linux-amd64" "linux-arm64" "darwin-amd64" "darwin-arm64" "windows-amd64.exe")
for platform in "${platforms[@]}"; do
    controller_binary="dist/rtk_controller-${platform}"
    wrapper_binary="dist/rtk_wrapper-${platform}"
    
    if [ ! -f "$controller_binary" ]; then
        echo "ERROR: Controller binary for $platform not found: $controller_binary"
        exit 1
    fi
    
    if [ ! -f "$wrapper_binary" ]; then
        echo "ERROR: Wrapper binary for $platform not found: $wrapper_binary"
        exit 1
    fi
    
    echo "✓ Found binaries for $platform"
done

echo "Build integration verification completed successfully!"
```

## 部署配置

### Systemd 服務文件

創建 `scripts/rtk-wrapper.service`：

```ini
[Unit]
Description=RTK MQTT Wrapper
After=network.target
Wants=network.target

[Service]
Type=simple
User=rtk
Group=rtk
WorkingDirectory=/opt/rtk-wrapper
ExecStart=/opt/rtk-wrapper/bin/rtk_wrapper --config /opt/rtk-wrapper/configs/wrapper.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### Docker 整合

創建 `wrapper/Dockerfile`：

```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .

# 建置 wrapper
RUN cd wrapper && go mod download
RUN cd wrapper && CGO_ENABLED=0 GOOS=linux go build -o rtk_wrapper ./cmd

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# 複製二進位檔案和配置
COPY --from=builder /app/wrapper/rtk_wrapper .
COPY --from=builder /app/wrapper/configs ./configs

# 設置權限
RUN chmod +x rtk_wrapper

# 暴露配置目錄
VOLUME ["/root/configs"]

# 預設命令
CMD ["./rtk_wrapper", "--config", "configs/wrapper.yaml"]
```

### Docker Compose

創建 `docker-compose.wrapper.yml`：

```yaml
version: '3.8'

services:
  rtk-wrapper:
    build:
      context: .
      dockerfile: wrapper/Dockerfile
    container_name: rtk-wrapper
    restart: unless-stopped
    volumes:
      - ./wrapper/configs:/root/configs
    environment:
      - TZ=Asia/Taipei
    networks:
      - rtk-network
    depends_on:
      - mqtt-broker
      
  mqtt-broker:
    image: eclipse-mosquitto:2
    container_name: mqtt-broker
    restart: unless-stopped
    ports:
      - "1883:1883"
      - "9001:9001"
    volumes:
      - ./mosquitto.conf:/mosquitto/config/mosquitto.conf
    networks:
      - rtk-network

networks:
  rtk-network:
    driver: bridge
```

## 持續整合

### GitHub Actions 整合

修改 `.github/workflows/build.yml`：

```yaml
name: Build and Test

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.23'
        
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        
    # Controller 建置和測試
    - name: Build Controller
      run: make build
      
    - name: Test Controller  
      run: make test
      
    # Wrapper 建置和測試
    - name: Build Wrapper
      run: make build-wrapper
      
    - name: Test Wrapper
      run: make test-wrapper
      
    # 跨平台建置
    - name: Cross-platform Build
      run: make build-all
      
    - name: Verify Build
      run: ./scripts/verify_build.sh
      
    # 上傳建置產物
    - name: Upload Artifacts
      uses: actions/upload-artifact@v3
      with:
        name: rtk-binaries
        path: |
          dist/
          build_dir/
```

## 升級和遷移

### 現有系統升級指南

1. **備份現有配置**：
   ```bash
   cp configs/controller.yaml configs/controller.yaml.backup
   ```

2. **安裝 wrapper**：
   ```bash
   make build-wrapper
   ```

3. **配置 wrapper**：
   ```bash
   cp wrapper/configs/wrapper.yaml.example wrapper/configs/wrapper.yaml
   # 編輯配置文件
   ```

4. **測試獨立運行**：
   ```bash
   make run-wrapper
   ```

5. **切換到混合模式**：
   ```bash
   ./build_dir/rtk_controller --hybrid --wrapper-config wrapper/configs/wrapper.yaml
   ```

### 向後兼容性保證

- RTK Controller 主程序保持完全向後兼容
- 所有現有的 MQTT 訊息格式繼續支援
- 配置文件格式不變
- CLI 命令保持一致
- Wrapper 為可選功能，不影響現有部署