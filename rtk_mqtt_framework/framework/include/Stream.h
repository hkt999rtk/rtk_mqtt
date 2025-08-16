#ifndef STREAM_H
#define STREAM_H

/**
 * @file Stream.h
 * @brief Arduino Stream 類兼容性定義
 */

#include "arduino_compat.h"

// Arduino Stream 類的基本實現
class Stream {
public:
    virtual ~Stream() {}
    virtual int available() = 0;
    virtual int read() = 0;
    virtual int peek() = 0;
    virtual void flush() = 0;
    virtual size_t write(uint8_t) = 0;
    virtual size_t write(const uint8_t* buffer, size_t size) = 0;
    
    // 實用方法
    size_t print(const char* str) {
        return write((const uint8_t*)str, strlen(str));
    }
    
    size_t println(const char* str) {
        size_t result = print(str);
        result += write('\n');
        return result;
    }
};

#endif // STREAM_H