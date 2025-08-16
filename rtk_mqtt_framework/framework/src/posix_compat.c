#include "rtk_platform_compat.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/time.h>
#include <stdarg.h>
#include <stdio.h>

/**
 * @file posix_compat.c
 * @brief POSIX 平台兼容性實現
 */

// === 記憶體管理 ===

void* rtk_malloc(size_t size) {
    return malloc(size);
}

void* rtk_calloc(size_t count, size_t size) {
    return calloc(count, size);
}

void* rtk_realloc(void* ptr, size_t size) {
    return realloc(ptr, size);
}

void rtk_free(void* ptr) {
    free(ptr);
}

// === 字串操作 ===

char* rtk_strdup(const char* str) {
    if (!str) return NULL;
    
    size_t len = strlen(str) + 1;
    char* copy = (char*)malloc(len);
    if (copy) {
        memcpy(copy, str, len);
    }
    return copy;
}

int rtk_snprintf(char* buffer, size_t size, const char* format, ...) {
    va_list args;
    va_start(args, format);
    int result = vsnprintf(buffer, size, format, args);
    va_end(args);
    return result;
}

// === 時間函數 ===

uint64_t rtk_get_time_ms(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return (uint64_t)(tv.tv_sec) * 1000 + (uint64_t)(tv.tv_usec) / 1000;
}

void rtk_sleep_ms(uint32_t ms) {
    usleep(ms * 1000);
}

// === 執行緒操作 ===

int rtk_mutex_init(rtk_mutex_t* mutex) {
    pthread_mutex_t* pmutex = (pthread_mutex_t*)malloc(sizeof(pthread_mutex_t));
    if (!pmutex) {
        return -1;
    }
    
    int result = pthread_mutex_init(pmutex, NULL);
    if (result != 0) {
        free(pmutex);
        return -1;
    }
    
    *mutex = (rtk_mutex_t)pmutex;
    return 0;
}

int rtk_mutex_lock(rtk_mutex_t mutex) {
    if (!mutex) return -1;
    return pthread_mutex_lock((pthread_mutex_t*)mutex);
}

int rtk_mutex_unlock(rtk_mutex_t mutex) {
    if (!mutex) return -1;
    return pthread_mutex_unlock((pthread_mutex_t*)mutex);
}

int rtk_mutex_destroy(rtk_mutex_t* mutex) {
    if (!mutex || !*mutex) return -1;
    
    int result = pthread_mutex_destroy((pthread_mutex_t*)*mutex);
    free(*mutex);
    *mutex = NULL;
    return result;
}

int rtk_thread_create(rtk_thread_t** thread, rtk_thread_func_t func, void* arg) {
    pthread_t* pthread = (pthread_t*)malloc(sizeof(pthread_t));
    if (!pthread) {
        return -1;
    }
    
    int result = pthread_create(pthread, NULL, func, arg);
    if (result != 0) {
        free(pthread);
        return -1;
    }
    
    *thread = (rtk_thread_t*)pthread;
    return 0;
}

int rtk_thread_join(rtk_thread_t* thread, void** retval) {
    if (!thread) return -1;
    
    pthread_t* pthread = (pthread_t*)thread;
    int result = pthread_join(*pthread, retval);
    free(pthread);
    return result;
}

int rtk_thread_detach(rtk_thread_t* thread) {
    if (!thread) return -1;
    
    pthread_t* pthread = (pthread_t*)thread;
    int result = pthread_detach(*pthread);
    free(pthread);
    return result;
}

// === 訊號量操作 ===

int rtk_sem_init(rtk_sem_t** sem, uint32_t initial_count) {
    // 簡化實現
    (void)sem;
    (void)initial_count;
    return 0;
}

int rtk_sem_wait(rtk_sem_t* sem) {
    (void)sem;
    return 0;
}

int rtk_sem_post(rtk_sem_t* sem) {
    (void)sem;
    return 0;
}

int rtk_sem_destroy(rtk_sem_t** sem) {
    (void)sem;
    return 0;
}

// === 原子操作 ===

int32_t rtk_atomic_add(volatile int32_t* ptr, int32_t value) {
    return __sync_fetch_and_add(ptr, value);
}

int32_t rtk_atomic_sub(volatile int32_t* ptr, int32_t value) {
    return __sync_fetch_and_sub(ptr, value);
}

int32_t rtk_atomic_inc(volatile int32_t* ptr) {
    return __sync_fetch_and_add(ptr, 1);
}

int32_t rtk_atomic_dec(volatile int32_t* ptr) {
    return __sync_fetch_and_sub(ptr, 1);
}

int32_t rtk_atomic_cas(volatile int32_t* ptr, int32_t expected, int32_t desired) {
    return __sync_val_compare_and_swap(ptr, expected, desired);
}