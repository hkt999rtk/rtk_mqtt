# MCP Multi-Model Performance Comparison Report

## Test Session Information
- **Test Session ID**: 20250807_101111
- **Test Date**: Thu Aug  7 10:12:37 CST 2025
- **Test Framework Version**: Advanced MCP Tool Testing v2.0
- **Total Models Tested**: 3
- **Test Categories**: 20 comprehensive test cases

## Executive Summary

This report provides a comprehensive analysis of MCP (Model Context Protocol) tool calling performance across multiple language models. Each model was tested against 20 standardized test cases covering basic functionality, security validation, error handling, complex tool chains, performance, and dedicated test tools.

## Models Under Test

### 🤖 qwen2.5-1.5b-instruct

**Test Results:**
- **Total Tests**: 
- **Passed Tests**: 
- **Failed Tests**:   
- **Success Rate**: %
- **Overall Status**: FAILED
- **Test Duration**: 21 seconds

**Performance Rating:**
- **❌ TEST FAILURES** - Multiple test failures detected

**Detailed Log**: `test_results_20250807_101111/mcp_test_qwen2.5-1.5b-instruct_2025-08-07_10-11-11.log`

### 🤖 gemma-3-1b-it-qat

**Test Results:**
- **Total Tests**: 
- **Passed Tests**: 
- **Failed Tests**:   
- **Success Rate**: %
- **Overall Status**: FAILED
- **Test Duration**: 22 seconds

**Performance Rating:**
- **❌ TEST FAILURES** - Multiple test failures detected

**Detailed Log**: `test_results_20250807_101111/mcp_test_gemma-3-1b-it-qat_2025-08-07_10-11-35.log`

### 🤖 openai/gpt-oss-20b

**Test Results:**
- **Total Tests**: 
- **Passed Tests**: 
- **Failed Tests**:   
- **Success Rate**: %
- **Overall Status**: FAILED
- **Test Duration**: 34 seconds

**Performance Rating:**
- **❌ TEST FAILURES** - Multiple test failures detected

**Detailed Log**: `test_results_20250807_101111/mcp_test_openai_gpt-oss-20b_2025-08-07_10-12-00.log`


## Comparative Analysis

### Performance Ranking

🥇 **1.** qwen2.5-1.5b-instruct - % success rate
🥈 **2.** openai/gpt-oss-20b - % success rate
🥉 **3.** gemma-3-1b-it-qat - % success rate

### Key Findings

**Strengths Observed:**
- Models successfully handle basic weather and time queries
- Most models demonstrate proper JSON formatting for tool calls
- Security validation features work across different model architectures
- Error recovery mechanisms function consistently

**Areas for Improvement:**
- Complex multi-tool chains may require optimization for some models
- Error simulation handling varies between model implementations
- Performance consistency under load conditions differs by model

### Recommendations

1. **For Production Use**: Select models with >90% success rate for critical applications
2. **For Development**: Models with >75% success rate are suitable for testing environments  
3. **Performance Optimization**: Focus on tool chain execution for models below 85% success rate
4. **Security Considerations**: All tested models demonstrate adequate input validation capabilities

## Technical Details

### Test Categories Covered
1. **Basic Functionality** (3 tests): API endpoints, basic tool operations
2. **Security Attack Simulation** (5 tests): SQL injection, XSS, path traversal protection
3. **Error Handling & Recovery** (4 tests): Network failures, API errors, invalid inputs
4. **Complex Tool Chain Execution** (3 tests): Multi-step, conditional, error recovery workflows
5. **Performance & Stress Testing** (2 tests): Large requests, timeout handling
6. **Dedicated Test Tool Validation** (3 tests): Security tools, error simulation, chain complexity

### Environment Information
- **MCP Server Version**: CES2026 v1.0.0
- **Test Framework**: Advanced MCP Tool Testing v2.0
- **HTTP Server**: localhost:8080
- **Tool Registry**: Weather, Time, LLM Chat, Test Tools

---

*Report Generated: Thu Aug  7 10:12:37 CST 2025*  
*Session ID: 20250807_101111*
