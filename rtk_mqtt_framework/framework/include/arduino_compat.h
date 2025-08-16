#ifndef ARDUINO_COMPAT_H
#define ARDUINO_COMPAT_H

#include <stdint.h>
#include <stddef.h>
#include <string.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file arduino_compat.h
 * @brief Arduino 兼容層，為 PubSubClient 提供必要的類型定義
 * 
 * 此文件提供了 PubSubClient 所需的基本 Arduino 類型和常數，
 * 使其能夠在非 Arduino 環境中編譯和運行。
 */

// Arduino 基本類型定義
typedef uint8_t byte;
typedef bool boolean;

// Arduino 常數
#ifndef HIGH
#define HIGH 1
#endif

#ifndef LOW
#define LOW 0
#endif

#ifndef INPUT
#define INPUT 0
#endif

#ifndef OUTPUT
#define OUTPUT 1
#endif

// 時間相關函數 (毫秒)
unsigned long millis(void);
void delay(unsigned long ms);

#ifdef __cplusplus
}

// C++ 部分：提供 Arduino 兼容的類

#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <netdb.h>

/**
 * @brief 基本的 IPAddress 類實現
 */
class IPAddress {
private:
    uint8_t _address[4];
    
public:
    IPAddress() {
        memset(_address, 0, 4);
    }
    
    IPAddress(uint8_t first_octet, uint8_t second_octet, uint8_t third_octet, uint8_t fourth_octet) {
        _address[0] = first_octet;
        _address[1] = second_octet;
        _address[2] = third_octet;
        _address[3] = fourth_octet;
    }
    
    IPAddress(uint32_t address) {
        _address[0] = address & 0xFF;
        _address[1] = (address >> 8) & 0xFF;
        _address[2] = (address >> 16) & 0xFF;
        _address[3] = (address >> 24) & 0xFF;
    }
    
    uint8_t& operator[](int index) {
        return _address[index];
    }
    
    const uint8_t& operator[](int index) const {
        return _address[index];
    }
    
    operator uint32_t() const {
        return ((uint32_t)_address[3] << 24) | 
               ((uint32_t)_address[2] << 16) | 
               ((uint32_t)_address[1] << 8) | 
               _address[0];
    }
};

/**
 * @brief 基本的 Client 抽象類 (類似 Arduino 的 Client)
 */
class Client {
public:
    virtual ~Client() {}
    virtual int connect(IPAddress ip, uint16_t port) = 0;
    virtual int connect(const char *host, uint16_t port) = 0;
    virtual size_t write(uint8_t) = 0;
    virtual size_t write(const uint8_t *buf, size_t size) = 0;
    virtual int available() = 0;
    virtual int read() = 0;
    virtual int read(uint8_t *buf, size_t size) = 0;
    virtual int peek() = 0;
    virtual void flush() = 0;
    virtual void stop() = 0;
    virtual uint8_t connected() = 0;
    virtual operator bool() = 0;
};

/**
 * @brief 基本的 TCP 客戶端實現
 */
class TCPClient : public Client {
private:
    int sockfd;
    bool _connected;
    
public:
    TCPClient() : sockfd(-1), _connected(false) {}
    virtual ~TCPClient() { stop(); }
    
    virtual int connect(IPAddress ip, uint16_t port) override;
    virtual int connect(const char *host, uint16_t port) override;
    virtual size_t write(uint8_t byte) override;
    virtual size_t write(const uint8_t *buf, size_t size) override;
    virtual int available() override;
    virtual int read() override;
    virtual int read(uint8_t *buf, size_t size) override;
    virtual int peek() override;
    virtual void flush() override;
    virtual void stop() override;
    virtual uint8_t connected() override;
    virtual operator bool() override;
};

#endif // __cplusplus

#endif // ARDUINO_COMPAT_H