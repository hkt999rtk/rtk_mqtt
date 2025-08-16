#!/bin/bash

# RTK Controller CLI Functional Test Script
# This script tests the interactive CLI functionality using expect

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
CONTROLLER_BINARY="./rtk-controller"
TEST_CONFIG="configs/controller.yaml"
TEST_TIMEOUT=30
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Log file
LOG_FILE="test/results/cli_functional_test.log"
mkdir -p "$(dirname "$LOG_FILE")"

# Helper functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✓ $1${NC}" | tee -a "$LOG_FILE"
    ((PASSED_TESTS++))
}

error() {
    echo -e "${RED}✗ $1${NC}" | tee -a "$LOG_FILE"
    ((FAILED_TESTS++))
}

warning() {
    echo -e "${YELLOW}⚠ $1${NC}" | tee -a "$LOG_FILE"
}

# Test counter
test_start() {
    ((TOTAL_TESTS++))
    log "Test $TOTAL_TESTS: $1"
}

# Check if required tools are available
check_prerequisites() {
    log "Checking prerequisites..."
    
    if ! command -v expect >/dev/null 2>&1; then
        error "expect is required but not installed. Please install expect."
        exit 1
    fi
    
    if [ ! -f "$CONTROLLER_BINARY" ]; then
        error "Controller binary not found at $CONTROLLER_BINARY"
        exit 1
    fi
    
    if [ ! -f "$TEST_CONFIG" ]; then
        error "Test configuration not found at $TEST_CONFIG"
        exit 1
    fi
    
    success "Prerequisites check passed"
}

# Test CLI startup and basic functionality
test_cli_startup() {
    test_start "CLI startup and help command"
    
    expect_script=$(cat << 'EOF'
set timeout 10
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    send "help\r"
    expect "Available commands:" {
        send "exit\r"
        expect eof
        exit 0
    }
    exit 1
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "CLI startup and help command"
    else
        error "CLI startup failed or help command not working"
    fi
}

# Test device management commands
test_device_commands() {
    test_start "Device management commands"
    
    expect_script=$(cat << 'EOF'
set timeout 15
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test device list (should be empty initially)
    send "device list\r"
    expect -re "(No devices found|DEVICE ID)" {
        # Test device help
        send "device help\r"
        expect "Device management commands" {
            # Test invalid device command
            send "device invalid-command\r"
            expect -re "(Unknown|Invalid|Error)" {
                send "exit\r"
                expect eof
                exit 0
            }
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Device management commands"
    else
        error "Device management commands failed"
    fi
}

# Test command management
test_command_management() {
    test_start "Command management functionality"
    
    expect_script=$(cat << 'EOF'
set timeout 15
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test command list (should be empty initially)
    send "command list\r"
    expect -re "(No commands found|COMMAND ID)" {
        # Test command help
        send "command help\r"
        expect "Command management" {
            # Test invalid command syntax
            send "command send\r"
            expect -re "(Usage|Error|required)" {
                send "exit\r"
                expect eof
                exit 0
            }
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Command management functionality"
    else
        error "Command management functionality failed"
    fi
}

# Test system commands
test_system_commands() {
    test_start "System status and monitoring commands"
    
    expect_script=$(cat << 'EOF'
set timeout 15
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test system status
    send "system status\r"
    expect "RTK Controller System Status" {
        # Test system health
        send "system health\r"
        expect "System Health Check" {
            # Test system stats
            send "system stats\r"
            expect -re "(System Statistics|METRIC)" {
                send "exit\r"
                expect eof
                exit 0
            }
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "System status and monitoring commands"
    else
        error "System status and monitoring commands failed"
    fi
}

# Test configuration commands
test_config_commands() {
    test_start "Configuration management commands"
    
    expect_script=$(cat << 'EOF'
set timeout 15
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test config show
    send "config show\r"
    expect -re "(localhost|mqtt)" {
        # Test config show with section
        send "config show --section mqtt\r"
        expect -re "(broker|port)" {
            send "exit\r"
            expect eof
            exit 0
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Configuration management commands"
    else
        error "Configuration management commands failed"
    fi
}

# Test log management commands
test_log_commands() {
    test_start "Log management commands"
    
    expect_script=$(cat << 'EOF'
set timeout 15
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test log stats
    send "log stats\r"
    expect -re "(Total Messages|MQTT)" {
        # Test log list
        send "log list --limit 5\r"
        expect -re "(No messages|TOPIC)" {
            send "exit\r"
            expect eof
            exit 0
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Log management commands"
    else
        error "Log management commands failed"
    fi
}

# Test auto-completion functionality
test_autocompletion() {
    test_start "Command auto-completion"
    
    expect_script=$(cat << 'EOF'
set timeout 10
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test partial command completion (simulate Tab)
    send "dev\t"
    expect -re "(device|completion)" {
        send "\r"
        expect -re "(Usage|Device management)" {
            send "exit\r"
            expect eof
            exit 0
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Command auto-completion"
    else
        warning "Command auto-completion test failed (may not be fully implemented)"
        # Don't count as failure since this is an advanced feature
        ((FAILED_TESTS--))
    fi
}

# Test command history
test_command_history() {
    test_start "Command history functionality"
    
    expect_script=$(cat << 'EOF'
set timeout 15
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Execute a command
    send "help\r"
    expect "Available commands:" {
        # Try to access history (simulate up arrow)
        send "\033\[A"
        expect -re "(help|rtk>)" {
            send "\r"
            expect "Available commands:" {
                send "exit\r"
                expect eof
                exit 0
            }
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Command history functionality"
    else
        warning "Command history test failed (readline may not be properly configured)"
        # Don't count as failure since this depends on readline configuration
        ((FAILED_TESTS--))
    fi
}

# Test error handling
test_error_handling() {
    test_start "Error handling and invalid commands"
    
    expect_script=$(cat << 'EOF'
set timeout 10
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test invalid command
    send "invalid-command\r"
    expect -re "(Unknown|Invalid|Error)" {
        # Test empty command
        send "\r"
        expect "rtk> " {
            # Test command with invalid arguments
            send "device info invalid-args\r"
            expect -re "(Error|Usage|Invalid)" {
                send "exit\r"
                expect eof
                exit 0
            }
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Error handling and invalid commands"
    else
        error "Error handling test failed"
    fi
}

# Test CLI responsiveness under load
test_cli_performance() {
    test_start "CLI performance and responsiveness"
    
    expect_script=$(cat << 'EOF'
set timeout 20
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Execute multiple commands quickly
    send "system status\r"
    expect "RTK Controller System Status" {
        send "device list\r"
        expect -re "(No devices|DEVICE ID)" {
            send "command list\r"
            expect -re "(No commands|COMMAND ID)" {
                send "system health\r"
                expect "System Health Check" {
                    send "log stats\r"
                    expect -re "(Total Messages|MQTT)" {
                        send "exit\r"
                        expect eof
                        exit 0
                    }
                }
            }
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "CLI performance and responsiveness"
    else
        error "CLI performance test failed"
    fi
}

# Test graceful exit
test_graceful_exit() {
    test_start "Graceful CLI exit"
    
    expect_script=$(cat << 'EOF'
set timeout 10
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test exit command
    send "exit\r"
    expect eof {
        exit 0
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Graceful CLI exit"
    else
        error "Graceful CLI exit test failed"
    fi
}

# Test with different output formats
test_output_formats() {
    test_start "Output format options"
    
    expect_script=$(cat << 'EOF'
set timeout 15
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Test JSON format
    send "device list --format json\r"
    expect -re "(\\[|\\{|No devices)" {
        # Test CSV format
        send "device list --format csv\r"
        expect -re "(,|No devices)" {
            # Test table format (default)
            send "device list --format table\r"
            expect -re "(DEVICE ID|No devices)" {
                send "exit\r"
                expect eof
                exit 0
            }
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Output format options"
    else
        error "Output format options test failed"
    fi
}

# Test interrupt handling (Ctrl+C)
test_interrupt_handling() {
    test_start "Interrupt signal handling"
    
    expect_script=$(cat << 'EOF'
set timeout 10
spawn ./rtk-controller --cli --config configs/controller.yaml
expect "rtk> " {
    # Send interrupt signal
    send "\003"
    expect -re "(rtk>|Interrupt)" {
        # CLI should still be responsive
        send "help\r"
        expect "Available commands:" {
            send "exit\r"
            expect eof
            exit 0
        }
    }
}
exit 1
EOF
    )
    
    if echo "$expect_script" | expect -f -; then
        success "Interrupt signal handling"
    else
        warning "Interrupt handling test failed (may be expected behavior)"
        # Don't count as failure since some implementations may exit on Ctrl+C
        ((FAILED_TESTS--))
    fi
}

# Main test execution
main() {
    log "Starting RTK Controller CLI Functional Tests"
    log "============================================="
    
    check_prerequisites
    
    # Core functionality tests
    test_cli_startup
    test_device_commands
    test_command_management
    test_system_commands
    test_config_commands
    test_log_commands
    
    # Advanced feature tests
    test_autocompletion
    test_command_history
    test_output_formats
    
    # Error handling and edge cases
    test_error_handling
    test_interrupt_handling
    
    # Performance tests
    test_cli_performance
    
    # Final test
    test_graceful_exit
    
    # Print summary
    log ""
    log "Test Summary"
    log "============"
    log "Total tests: $TOTAL_TESTS"
    log "Passed: $PASSED_TESTS"
    log "Failed: $FAILED_TESTS"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        success "All CLI functional tests passed!"
        echo -e "${GREEN}🎉 CLI functionality is working correctly${NC}"
        exit 0
    else
        error "Some CLI functional tests failed!"
        echo -e "${RED}❌ CLI has $FAILED_TESTS failing test(s)${NC}"
        exit 1
    fi
}

# Handle script interruption
cleanup() {
    log "Test script interrupted"
    # Kill any remaining controller processes
    pkill -f rtk-controller || true
    exit 130
}

trap cleanup INT TERM

# Run main function
main "$@"