#!/bin/bash

# RTK Controller Release Packaging Script
# 此腳本用於打包客戶發行版本

set -e  # 遇到錯誤時立即退出

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 函數：顯示彩色訊息
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 函數：顯示使用說明
show_usage() {
    cat << EOF
RTK Controller Release Script

使用方法:
    $0 [版本號]

參數:
    版本號          發行版本號 (例如: v1.0.0, 1.2.3)
                   如果未指定，將嘗試從 git 取得版本號

範例:
    $0 v1.0.0       # 指定版本號為 v1.0.0
    $0              # 自動偵測版本號

輸出:
    release/release_[版本號].tgz

EOF
}

# 函數：取得版本號
get_version() {
    local version="$1"
    
    if [ -n "$version" ]; then
        echo "$version"
        return 0
    fi
    
    # 嘗試從 git 取得版本號
    if command -v git >/dev/null 2>&1 && git rev-parse --git-dir >/dev/null 2>&1; then
        # 嘗試取得 git tag
        local git_tag=$(git describe --tags --exact-match 2>/dev/null || echo "")
        if [ -n "$git_tag" ]; then
            echo "$git_tag"
            return 0
        fi
        
        # 取得短 commit hash
        local commit=$(git rev-parse --short HEAD 2>/dev/null || echo "")
        if [ -n "$commit" ]; then
            echo "dev-$commit"
            return 0
        fi
    fi
    
    # 預設版本號
    echo "dev-$(date +%Y%m%d)"
}

# 函數：驗證必要檔案
validate_files() {
    local missing_files=()
    
    # 檢查可執行檔案
    if [ ! -d "dist" ]; then
        log_error "dist 目錄不存在。請先執行 'make all' 編譯所有平台版本。"
        exit 1
    fi
    
    local binaries=(
        "dist/rtk_controller-linux-arm64"
        "dist/rtk_controller-linux-amd64"
        "dist/rtk_controller-darwin-arm64"
        "dist/rtk_controller-windows-amd64.exe"
    )
    
    for binary in "${binaries[@]}"; do
        if [ ! -f "$binary" ]; then
            missing_files+=("$binary")
        fi
    done
    
    # 檢查配置檔案
    if [ ! -f "configs/controller.yaml" ]; then
        missing_files+=("configs/controller.yaml")
    fi
    
    # 檢查必要文檔檔案
    local required_docs=(
        "MANUAL.md"
    )
    
    for doc in "${required_docs[@]}"; do
        if [ ! -f "$doc" ]; then
            missing_files+=("$doc")
        fi
    done
    
    # 檢查可選文檔檔案
    local optional_docs=(
        "CLI_USAGE.md"
        "QUICKSTART.md"
        "SPEC.md"
    )
    
    for doc in "${optional_docs[@]}"; do
        if [ ! -f "$doc" ]; then
            log_warning "可選文檔檔案不存在: $doc (將跳過)"
        fi
    done
    
    if [ ${#missing_files[@]} -gt 0 ]; then
        log_error "以下必要檔案不存在："
        for file in "${missing_files[@]}"; do
            echo "  - $file"
        done
        echo ""
        log_info "請確保已執行以下命令："
        echo "  make all          # 編譯所有平台版本"
        echo "  make clean        # 清理後重新編譯"
        exit 1
    fi
}

# 函數：建立發行版本目錄結構
create_release_structure() {
    local temp_dir="$1"
    local version="$2"
    
    log_info "建立發行版本目錄結構..."
    
    # 建立目錄結構
    mkdir -p "$temp_dir/bin"
    mkdir -p "$temp_dir/configs"
    mkdir -p "$temp_dir/docs"
    mkdir -p "$temp_dir/test/scripts"
    
    # 複製可執行檔案
    cp dist/rtk_controller-linux-arm64 "$temp_dir/bin/"
    cp dist/rtk_controller-linux-amd64 "$temp_dir/bin/"
    cp dist/rtk_controller-darwin-arm64 "$temp_dir/bin/"
    cp dist/rtk_controller-windows-amd64.exe "$temp_dir/bin/"
    
    # 複製配置檔案
    cp configs/controller.yaml "$temp_dir/configs/"
    
    # 複製必要文檔檔案到根目錄
    cp MANUAL.md "$temp_dir/"
    
    # 複製技術文檔到 docs 目錄
    [ -f "CLI_USAGE.md" ] && cp CLI_USAGE.md "$temp_dir/docs/"
    [ -f "QUICKSTART.md" ] && cp QUICKSTART.md "$temp_dir/docs/"
    [ -f "SPEC.md" ] && cp SPEC.md "$temp_dir/docs/"
    [ -f "PROJECT_SUMMARY.md" ] && cp PROJECT_SUMMARY.md "$temp_dir/docs/"
    
    # 複製測試腳本
    [ -f "test/scripts/test_cli_commands.sh" ] && cp test/scripts/test_cli_commands.sh "$temp_dir/test/scripts/"
    [ -f "test/scripts/performance_test.sh" ] && cp test/scripts/performance_test.sh "$temp_dir/test/scripts/"
    [ -f "test/scripts/run_all_tests.sh" ] && cp test/scripts/run_all_tests.sh "$temp_dir/test/scripts/"
    
    # 複製 demo 腳本
    [ -f "demo_cli.sh" ] && cp demo_cli.sh "$temp_dir/"
    
    # 建立 LICENSE 檔案 (如果不存在)
    if [ ! -f LICENSE ]; then
        cat > "$temp_dir/LICENSE" << 'EOF'
MIT License

Copyright (c) 2024 Realtek Semiconductor Corp.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
EOF
    else
        cp LICENSE "$temp_dir/"
    fi
    
    # 建立版本資訊檔案
    cat > "$temp_dir/VERSION" << EOF
RTK Controller
Version: $version
Build Date: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
Git Commit: $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

Supported Platforms:
- Linux ARM64
- Linux x86_64
- macOS ARM64
- Windows x86_64

Components Included:
- RTK Controller CLI Tool
- Configuration Files
- Test Scripts
- Documentation
- Demo Scripts

Files in this release:
- MANUAL.md - 工程師使用手冊 (詳細操作指南)
- bin/ - 跨平台可執行檔案
- configs/ - 配置檔案範本
- docs/ - 技術文檔
- test/scripts/ - 測試和驗證腳本
- demo_cli.sh - 功能示範腳本

For detailed usage instructions, please refer to MANUAL.md
EOF
    
    log_success "目錄結構建立完成"
}

# 函數：設定檔案權限
set_permissions() {
    local temp_dir="$1"
    
    log_info "設定檔案權限..."
    
    # 設定可執行檔案權限
    chmod +x "$temp_dir/bin/rtk_controller-linux-"*
    chmod +x "$temp_dir/bin/rtk_controller-darwin-"*
    
    # 設定腳本檔案權限
    [ -f "$temp_dir/demo_cli.sh" ] && chmod +x "$temp_dir/demo_cli.sh"
    find "$temp_dir/test/scripts" -name "*.sh" -exec chmod +x {} \; 2>/dev/null || true
    
    # 設定配置檔案權限
    chmod 644 "$temp_dir/configs/controller.yaml"
    
    # 設定文檔權限
    find "$temp_dir/docs" -name "*.md" -exec chmod 644 {} \; 2>/dev/null || true
    chmod 644 "$temp_dir/LICENSE"
    chmod 644 "$temp_dir/VERSION"
    
    log_success "檔案權限設定完成"
}

# 函數：驗證可執行檔案
validate_binaries() {
    local temp_dir="$1"
    
    log_info "驗證可執行檔案..."
    
    local binaries=(
        "$temp_dir/bin/rtk_controller-linux-arm64"
        "$temp_dir/bin/rtk_controller-linux-amd64"
        "$temp_dir/bin/rtk_controller-darwin-arm64"
        "$temp_dir/bin/rtk_controller-windows-amd64.exe"
    )
    
    for binary in "${binaries[@]}"; do
        if [ ! -f "$binary" ]; then
            log_error "可執行檔案不存在: $binary"
            return 1
        fi
        
        # 檢查檔案大小 (應該 > 1MB)
        local size=$(stat -f%z "$binary" 2>/dev/null || stat -c%s "$binary" 2>/dev/null || echo "0")
        if [ "$size" -lt 1048576 ]; then
            log_warning "可執行檔案可能異常 (大小 < 1MB): $binary"
        fi
    done
    
    log_success "可執行檔案驗證完成"
}

# 函數：建立壓縮檔
create_archive() {
    local temp_dir="$1"
    local version="$2"
    local release_dir="$3"
    local archive_name="release_${version}.tgz"
    local archive_path="$(pwd)/$release_dir/$archive_name"
    
    log_info "建立壓縮檔: $archive_name"
    
    # 確保 release 目錄存在
    mkdir -p "$release_dir"
    
    # 進入臨時目錄的父目錄進行打包
    local temp_parent=$(dirname "$temp_dir")
    local temp_name=$(basename "$temp_dir")
    local current_dir=$(pwd)
    
    cd "$temp_parent"
    tar -czf "$archive_path" "$temp_name"
    local tar_result=$?
    cd "$current_dir"
    
    if [ $tar_result -ne 0 ]; then
        log_error "tar 命令執行失敗"
        return 1
    fi
    
    # 驗證壓縮檔
    if [ ! -f "$archive_path" ]; then
        log_error "壓縮檔建立失敗: $archive_path"
        return 1
    fi
    
    local archive_size=$(stat -f%z "$archive_path" 2>/dev/null || stat -c%s "$archive_path" 2>/dev/null || echo "0")
    local archive_size_mb=$((archive_size / 1024 / 1024))
    
    log_success "壓縮檔建立完成: $archive_path (${archive_size_mb}MB)"
    
    # 顯示壓縮檔內容
    log_info "壓縮檔內容預覽:"
    tar -tzf "$archive_path" | head -20
    if [ $(tar -tzf "$archive_path" | wc -l) -gt 20 ]; then
        echo "... (更多檔案)"
    fi
}

# 主要函數
main() {
    # 檢查參數
    if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
        show_usage
        exit 0
    fi
    
    # 取得版本號
    local version=$(get_version "$1")
    log_info "發行版本: $version"
    
    # 驗證必要檔案
    validate_files
    
    # 建立 release 目錄
    local release_dir="release"
    mkdir -p "$release_dir"
    
    # 建立臨時目錄
    local temp_dir=$(mktemp -d)
    local temp_release_dir="$temp_dir/rtk_controller_release"
    
    # 清理函數
    cleanup() {
        if [ -d "$temp_dir" ]; then
            rm -rf "$temp_dir"
        fi
    }
    trap cleanup EXIT
    
    log_info "開始建立發行版本..."
    
    # 建立發行版本結構
    create_release_structure "$temp_release_dir" "$version"
    
    # 設定權限
    set_permissions "$temp_release_dir"
    
    # 驗證可執行檔案
    validate_binaries "$temp_release_dir"
    
    # 建立壓縮檔
    create_archive "$temp_release_dir" "$version" "$release_dir"
    
    log_success "發行版本打包完成！"
    echo ""
    echo "==================================="
    echo "    RTK Controller Release"
    echo "==================================="
    echo "版本號: $version"
    echo "輸出檔案: release/release_${version}.tgz"
    echo "檔案大小: $(ls -lh release/release_${version}.tgz | awk '{print $5}')"
    echo ""
    echo "使用說明："
    echo "1. 將 release_${version}.tgz 傳送給客戶"
    echo "2. 客戶解壓縮後請參考 docs/QUICKSTART.md 進行部署"
    echo "3. 可執行檔案位於 bin/ 目錄"
    echo "4. 測試腳本位於 test/scripts/ 目錄"
    echo "==================================="
}

# 檢查是否在正確的目錄中執行
if [ ! -f "go.mod" ] || [ ! -f "cmd/controller/main.go" ]; then
    log_error "請在 rtk_controller 專案根目錄中執行此腳本"
    exit 1
fi

# 執行主函數
main "$@"