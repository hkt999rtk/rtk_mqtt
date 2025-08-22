# RTK Controller Makefile
# 支援多平台交叉編譯

# 專案基本資訊
APP_NAME = rtk_controller
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date '+%Y-%m-%d_%H:%M:%S')
COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
MAIN_PATH = ./cmd/controller/main.go

# 編譯參數
LDFLAGS = -ldflags="-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.Commit=$(COMMIT)"
BUILD_FLAGS = -trimpath

# 輸出目錄
DIST_DIR = dist
BUILD_DIR = build_dir

# 目標平台定義
TARGETS = \
	linux-arm64 \
	darwin-arm64 \
	linux-amd64 \
	windows-amd64

# 測試程序名稱
TEST_PROGRAMS = test_topology_simple test_diagnostics test_qos

# 預設目標
.PHONY: all
all: clean build-all

# 顯示幫助
.PHONY: help
help:
	@echo "RTK Controller Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all                - 編譯所有平台版本"
	@echo "  build              - 編譯本機版本（RTK Controller）"
	@echo "  build-wrapper      - 編譯 RTK MQTT Wrapper"
	@echo "  build-all-components - 編譯所有組件（Controller + Wrapper）"
	@echo "  build-all          - 編譯所有平台版本"
	@echo "  test               - 執行測試"
	@echo "  release            - 建立完整的發行版本包"
	@echo "  clean              - 清理編譯產物"
	@echo "  deps               - 下載相依套件"
	@echo "  lint               - 程式碼檢查"
	@echo "  fmt                - 格式化程式碼"
	@echo ""
	@echo "單一平台編譯:"
	@echo "  linux-arm64        - ARM64 Linux 版本"
	@echo "  darwin-arm64       - ARM64 macOS 版本"
	@echo "  linux-amd64        - x86_64 Linux 版本"
	@echo "  windows-amd64      - x86_64 Windows 版本"
	@echo ""
	@echo "Environment Variables:"
	@echo "  VERSION            - 版本號 (預設: git describe)"

# 下載相依套件
.PHONY: deps
deps:
	@echo "正在下載 Go 模組..."
	go mod download
	go mod tidy

# 建立輸出目錄
$(DIST_DIR):
	@mkdir -p $(DIST_DIR)

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# 編譯所有平台
.PHONY: build-all
build-all: $(DIST_DIR) linux-arm64 darwin-arm64 linux-amd64 windows-amd64

# 平台特定編譯規則
.PHONY: linux-arm64 darwin-arm64 linux-amd64 windows-amd64

linux-arm64: deps $(DIST_DIR)
	@echo "正在編譯 Linux ARM64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
		$(BUILD_FLAGS) \
		$(LDFLAGS) \
		-o $(DIST_DIR)/$(APP_NAME)-linux-arm64 \
		$(MAIN_PATH)
	@echo "✓ Linux ARM64 編譯完成: $(DIST_DIR)/$(APP_NAME)-linux-arm64"

darwin-arm64: deps $(DIST_DIR)
	@echo "正在編譯 macOS ARM64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
		$(BUILD_FLAGS) \
		$(LDFLAGS) \
		-o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 \
		$(MAIN_PATH)
	@echo "✓ macOS ARM64 編譯完成: $(DIST_DIR)/$(APP_NAME)-darwin-arm64"

linux-amd64: deps $(DIST_DIR)
	@echo "正在編譯 Linux x86_64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		$(BUILD_FLAGS) \
		$(LDFLAGS) \
		-o $(DIST_DIR)/$(APP_NAME)-linux-amd64 \
		$(MAIN_PATH)
	@echo "✓ Linux x86_64 編譯完成: $(DIST_DIR)/$(APP_NAME)-linux-amd64"

windows-amd64: deps $(DIST_DIR)
	@echo "正在編譯 Windows x86_64..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
		$(BUILD_FLAGS) \
		$(LDFLAGS) \
		-o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe \
		$(MAIN_PATH)
	@echo "✓ Windows x86_64 編譯完成: $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe"

# 本機開發版本
.PHONY: build
build: deps $(BUILD_DIR)
	@echo "正在編譯本機開發版本..."
	@go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "✓ 開發版本編譯完成: $(BUILD_DIR)/$(APP_NAME)"

# MQTT Wrapper 編譯
.PHONY: build-wrapper
build-wrapper: deps $(BUILD_DIR)
	@echo "正在編譯 RTK MQTT Wrapper..."
	@cd ../rtk_wrapper && go build $(BUILD_FLAGS) $(LDFLAGS) -o ../rtk_controller/$(BUILD_DIR)/rtk_wrapper cmd/main.go
	@echo "✓ RTK MQTT Wrapper 編譯完成: $(BUILD_DIR)/rtk_wrapper"

# 編譯所有組件
.PHONY: build-all-components
build-all-components: build build-wrapper
	@echo "✓ 所有組件編譯完成"

# 執行測試
.PHONY: test
test:
	@echo "正在執行測試..."
	@go test -v -timeout 30m ./internal/... ./pkg/...

# 測試覆蓋率
.PHONY: coverage
coverage:
	@echo "正在生成測試覆蓋率報告..."
	@mkdir -p test/results/coverage
	@go test -v -timeout 30m -coverprofile=test/results/coverage/coverage.out -covermode=atomic ./internal/... ./pkg/...
	@go tool cover -html=test/results/coverage/coverage.out -o test/results/coverage/coverage.html
	@go tool cover -func=test/results/coverage/coverage.out

# 程式碼檢查
.PHONY: lint
lint:
	@echo "正在進行程式碼檢查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "警告: golangci-lint 未安裝，使用 go vet 進行基本檢查"; \
		go vet ./...; \
	fi

# 格式化程式碼
.PHONY: fmt
fmt:
	@echo "正在格式化程式碼..."
	@go fmt ./...

# 清理編譯產物
.PHONY: clean
clean:
	@echo "正在清理編譯產物..."
	@rm -rf $(DIST_DIR) $(BUILD_DIR)
	@rm -f $(APP_NAME) rtk-controller
	@rm -rf test/results/
	@rm -rf logs/ data/ release/
	@echo "✓ 清理完成"

# 顯示編譯資訊
.PHONY: info
info:
	@echo "編譯資訊:"
	@echo "  應用程式名稱: $(APP_NAME)"
	@echo "  版本: $(VERSION)"
	@echo "  編譯時間: $(BUILD_TIME)"
	@echo "  Commit: $(COMMIT)"
	@echo "  目標平台: $(TARGETS)"

# 開發工具
.PHONY: run run-cli demo
run: build
	@echo "啟動 RTK Controller..."
	$(BUILD_DIR)/$(APP_NAME) --config configs/controller.yaml

run-cli: build
	@echo "啟動 RTK Controller CLI..."
	$(BUILD_DIR)/$(APP_NAME) --cli --config configs/controller.yaml

demo: build
	@echo "運行演示腳本..."
	@if [ -f ./demo_cli.sh ]; then \
		./demo_cli.sh; \
	else \
		echo "警告: demo_cli.sh 不存在"; \
	fi

# 建立發行版本 (整合原 release.sh 功能)
.PHONY: release
release: clean build-all build-test-tools
	@echo "正在建立發行版本..."
	@mkdir -p release
	@for target in $(TARGETS); do \
		echo "正在準備 $$target 發行包..."; \
		PACKAGE_DIR="$(DIST_DIR)/release-$(APP_NAME)-$$target"; \
		rm -rf $$PACKAGE_DIR; \
		mkdir -p $$PACKAGE_DIR/bin; \
		mkdir -p $$PACKAGE_DIR/configs; \
		mkdir -p $$PACKAGE_DIR/docs; \
		mkdir -p $$PACKAGE_DIR/test/scripts; \
		echo "複製二進位檔案..."; \
		if [ "$$target" = "windows-amd64" ]; then \
			cp $(DIST_DIR)/$(APP_NAME)-$$target.exe $$PACKAGE_DIR/bin/; \
		else \
			cp $(DIST_DIR)/$(APP_NAME)-$$target $$PACKAGE_DIR/bin/; \
			chmod +x $$PACKAGE_DIR/bin/$(APP_NAME)-$$target; \
		fi; \
		echo "複製配置檔案..."; \
		cp configs/controller.yaml $$PACKAGE_DIR/configs/; \
		echo "複製測試工具..."; \
		if [ -d $(DIST_DIR)/test-tools-$$target ]; then \
			cp -r $(DIST_DIR)/test-tools-$$target $$PACKAGE_DIR/test-tools; \
			if [ "$$target" != "windows-amd64" ]; then \
				chmod +x $$PACKAGE_DIR/test-tools/* 2>/dev/null || true; \
			fi; \
		fi; \
		echo "複製文檔..."; \
		cp docs/MANUAL.md $$PACKAGE_DIR/ 2>/dev/null || echo "警告: MANUAL.md 不存在"; \
		cp -r docs/* $$PACKAGE_DIR/docs/ 2>/dev/null || true; \
		cp test/scripts/*.sh $$PACKAGE_DIR/test/scripts/ 2>/dev/null || true; \
		cp demo_cli.sh $$PACKAGE_DIR/ 2>/dev/null || true; \
		echo "建立 LICENSE..."; \
		if [ ! -f LICENSE ]; then \
			echo "MIT License" > $$PACKAGE_DIR/LICENSE; \
			echo "" >> $$PACKAGE_DIR/LICENSE; \
			echo "Copyright (c) 2024 Realtek Semiconductor Corp." >> $$PACKAGE_DIR/LICENSE; \
		else \
			cp LICENSE $$PACKAGE_DIR/; \
		fi; \
		echo "建立 VERSION 檔案..."; \
		echo "RTK Controller" > $$PACKAGE_DIR/VERSION; \
		echo "Version: $(VERSION)" >> $$PACKAGE_DIR/VERSION; \
		echo "Build Date: $(BUILD_TIME)" >> $$PACKAGE_DIR/VERSION; \
		echo "Git Commit: $(COMMIT)" >> $$PACKAGE_DIR/VERSION; \
		echo "Platform: $$target" >> $$PACKAGE_DIR/VERSION; \
		echo "打包 $$target..."; \
		(cd $(DIST_DIR) && tar -czf rtk_controller_$(VERSION)_$$target.tar.gz release-$(APP_NAME)-$$target --transform="s/release-$(APP_NAME)-$$target/$(APP_NAME)-$$target/"); \
		mv $(DIST_DIR)/rtk_controller_$(VERSION)_$$target.tar.gz release/; \
		echo "✓ $$target 打包完成: release/rtk_controller_$(VERSION)_$$target.tar.gz"; \
	done
	@echo "✓ 所有發行版本建立完成"
	@echo ""
	@echo "==================================="
	@echo "  RTK Controller Release"
	@echo "==================================="
	@echo "版本號: $(VERSION)"
	@echo "發行檔案列表:"
	@ls -la release/rtk_controller_$(VERSION)_*.tar.gz 2>/dev/null || echo "  無發行檔案"
	@echo ""
	@echo "使用說明："
	@echo "1. 將 rtk_controller_$(VERSION)_[platform].tar.gz 傳送給客戶"
	@echo "2. 客戶解壓縮後請參考 MANUAL.md 進行部署"
	@echo "==================================="

# 編譯測試工具
.PHONY: build-test-tools
build-test-tools: $(DIST_DIR)
	@echo "正在編譯測試工具..."
	@for target in $(TARGETS); do \
		echo "正在編譯 $$target 測試工具..."; \
		mkdir -p $(DIST_DIR)/test-tools-$$target; \
		for prog in $(TEST_PROGRAMS); do \
			if [ "$$target" = "linux-arm64" ]; then \
				CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
					$(BUILD_FLAGS) $(LDFLAGS) \
					-o $(DIST_DIR)/test-tools-$$target/$$prog \
					./test/$$prog.go 2>/dev/null || echo "跳過 $$prog (檔案不存在)"; \
			elif [ "$$target" = "darwin-arm64" ]; then \
				CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
					$(BUILD_FLAGS) $(LDFLAGS) \
					-o $(DIST_DIR)/test-tools-$$target/$$prog \
					./test/$$prog.go 2>/dev/null || echo "跳過 $$prog (檔案不存在)"; \
			elif [ "$$target" = "linux-amd64" ]; then \
				CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
					$(BUILD_FLAGS) $(LDFLAGS) \
					-o $(DIST_DIR)/test-tools-$$target/$$prog \
					./test/$$prog.go 2>/dev/null || echo "跳過 $$prog (檔案不存在)"; \
			elif [ "$$target" = "windows-amd64" ]; then \
				CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
					$(BUILD_FLAGS) $(LDFLAGS) \
					-o $(DIST_DIR)/test-tools-$$target/$$prog.exe \
					./test/$$prog.go 2>/dev/null || echo "跳過 $$prog (檔案不存在)"; \
			fi; \
		done; \
		echo "✓ $$target 測試工具編譯完成"; \
	done
	@echo "✓ 所有測試工具編譯完成"

# 列出編譯產物
.PHONY: list
list:
	@echo "編譯產物:"
	@if [ -d "$(DIST_DIR)" ]; then \
		ls -la $(DIST_DIR)/; \
	else \
		echo "  無編譯產物 (執行 'make all' 進行編譯)"; \
	fi

# Default goal
.DEFAULT_GOAL := help