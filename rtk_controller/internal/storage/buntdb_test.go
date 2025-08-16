package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuntDB(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "buntdb_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path",
			path:    dbPath,
			wantErr: false,
		},
		{
			name:    "memory database",
			path:    ":memory:",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewBuntDB(tt.path)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				
				// Clean up
				if db != nil {
					db.Close()
				}
			}
		})
	}
}

func TestBuntDB_SetAndGet(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{
			name:  "string value",
			key:   "test:string",
			value: "hello world",
		},
		{
			name:  "integer value",
			key:   "test:int",
			value: 42,
		},
		{
			name: "struct value",
			key:  "test:struct",
			value: TestStruct{
				Name:  "test",
				Value: 123,
			},
		},
		{
			name: "map value",
			key:  "test:map",
			value: map[string]interface{}{
				"key1": "value1",
				"key2": 456,
			},
		},
		{
			name:  "slice value",
			key:   "test:slice",
			value: []string{"item1", "item2", "item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Set
			err := db.Set(tt.key, tt.value)
			assert.NoError(t, err)

			// Test Get based on value type
			switch tt.value.(type) {
			case string:
				var result string
				err = db.Get(tt.key, &result)
				assert.NoError(t, err)
				assert.Equal(t, tt.value, result)

			case int:
				var result int
				err = db.Get(tt.key, &result)
				assert.NoError(t, err)
				assert.Equal(t, tt.value, result)

			case TestStruct:
				var result TestStruct
				err = db.Get(tt.key, &result)
				assert.NoError(t, err)
				assert.Equal(t, tt.value, result)

			case map[string]interface{}:
				var result map[string]interface{}
				err = db.Get(tt.key, &result)
				assert.NoError(t, err)
				assert.Equal(t, tt.value, result)

			case []string:
				var result []string
				err = db.Get(tt.key, &result)
				assert.NoError(t, err)
				assert.Equal(t, tt.value, result)
			}
		})
	}
}

func TestBuntDB_GetNonExistentKey(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	var result string
	err = db.Get("non:existent:key", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBuntDB_Delete(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	key := "test:delete"
	value := "delete me"

	// Set a value
	err = db.Set(key, value)
	assert.NoError(t, err)

	// Verify it exists
	var result string
	err = db.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	// Delete it
	err = db.Delete(key)
	assert.NoError(t, err)

	// Verify it's gone
	err = db.Get(key, &result)
	assert.Error(t, err)
}

func TestBuntDB_List(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Set up test data
	testData := map[string]string{
		"devices:device1": "data1",
		"devices:device2": "data2",
		"devices:device3": "data3",
		"commands:cmd1":   "cmd_data1",
		"commands:cmd2":   "cmd_data2",
		"other:data":      "other_data",
	}

	for key, value := range testData {
		err = db.Set(key, value)
		assert.NoError(t, err)
	}

	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "list devices",
			pattern:  "devices:*",
			expected: []string{"devices:device1", "devices:device2", "devices:device3"},
		},
		{
			name:     "list commands",
			pattern:  "commands:*",
			expected: []string{"commands:cmd1", "commands:cmd2"},
		},
		{
			name:     "list all",
			pattern:  "*",
			expected: []string{"commands:cmd1", "commands:cmd2", "devices:device1", "devices:device2", "devices:device3", "other:data"},
		},
		{
			name:     "no matches",
			pattern:  "nonexistent:*",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, err := db.List(tt.pattern)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, keys)
		})
	}
}

func TestBuntDB_CreateIndex(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Test creating a string index
	err = db.CreateIndex("test_index", "test:*", nil)
	assert.NoError(t, err)

	// Test creating the same index again (should not error)
	err = db.CreateIndex("test_index", "test:*", nil)
	assert.NoError(t, err)

	// Add some test data
	testData := []struct {
		key   string
		value string
	}{
		{"test:c", "charlie"},
		{"test:a", "alpha"},
		{"test:b", "beta"},
	}

	for _, data := range testData {
		err = db.Set(data.key, data.value)
		assert.NoError(t, err)
	}

	// Test that we can query the indexed data
	keys, err := db.List("test:*")
	assert.NoError(t, err)
	assert.Len(t, keys, 3)
}

func TestBuntDB_ViewAndUpdate(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Test Update
	err = db.Update(func(tx Transaction) error {
		return tx.Set("test:key", "test_value")
	})
	assert.NoError(t, err)

	// Test View
	err = db.View(func(tx Transaction) error {
		var value string
		err := tx.Get("test:key", &value)
		assert.NoError(t, err)
		assert.Equal(t, "test_value", value)
		return nil
	})
	assert.NoError(t, err)

	// Test transaction rollback on error
	err = db.Update(func(tx Transaction) error {
		err := tx.Set("test:key2", "will_be_rolled_back")
		assert.NoError(t, err)
		return fmt.Errorf("intentional error")
	})
	assert.Error(t, err)

	// Verify rollback worked
	var value string
	err = db.Get("test:key2", &value)
	assert.Error(t, err) // Should not exist due to rollback
}

func TestBuntDB_ConcurrentAccess(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	const numGoroutines = 10
	const numOperations = 100

	// Test concurrent writes
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent:goroutine%d:item%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				
				err := db.Set(key, value)
				assert.NoError(t, err)
				
				var retrieved string
				err = db.Get(key, &retrieved)
				assert.NoError(t, err)
				assert.Equal(t, value, retrieved)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all data was written
	keys, err := db.List("concurrent:*")
	assert.NoError(t, err)
	assert.Len(t, keys, numGoroutines*numOperations)
}

func TestBuntDB_JSONSerialization(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	type ComplexStruct struct {
		ID        string                 `json:"id"`
		Timestamp time.Time              `json:"timestamp"`
		Data      map[string]interface{} `json:"data"`
		Tags      []string               `json:"tags"`
		Nested    struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		} `json:"nested"`
	}

	now := time.Now().UTC().Truncate(time.Second)
	
	original := ComplexStruct{
		ID:        "test-123",
		Timestamp: now,
		Data: map[string]interface{}{
			"metric1": 42.5,
			"metric2": "test_string",
			"metric3": true,
		},
		Tags: []string{"tag1", "tag2", "tag3"},
		Nested: struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}{
			Name:  "nested_test",
			Value: 789,
		},
	}

	// Test setting complex struct
	err = db.Set("complex:test", original)
	assert.NoError(t, err)

	// Test getting complex struct
	var retrieved ComplexStruct
	err = db.Get("complex:test", &retrieved)
	assert.NoError(t, err)

	assert.Equal(t, original.ID, retrieved.ID)
	assert.Equal(t, original.Timestamp, retrieved.Timestamp)
	assert.Equal(t, original.Data, retrieved.Data)
	assert.Equal(t, original.Tags, retrieved.Tags)
	assert.Equal(t, original.Nested, retrieved.Nested)
}

func TestBuntDB_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	const numOperations = 10000
	
	// Test write performance
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("perf:item:%d", i)
		value := map[string]interface{}{
			"id":    i,
			"name":  fmt.Sprintf("item_%d", i),
			"value": i * 2,
		}
		
		err := db.Set(key, value)
		assert.NoError(t, err)
	}
	writeTime := time.Since(start)
	
	t.Logf("Write performance: %d operations in %v (%.2f ops/sec)", 
		numOperations, writeTime, float64(numOperations)/writeTime.Seconds())

	// Test read performance
	start = time.Now()
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("perf:item:%d", i)
		var value map[string]interface{}
		
		err := db.Get(key, &value)
		assert.NoError(t, err)
		assert.Equal(t, i, int(value["id"].(float64)))
	}
	readTime := time.Since(start)
	
	t.Logf("Read performance: %d operations in %v (%.2f ops/sec)", 
		numOperations, readTime, float64(numOperations)/readTime.Seconds())

	// Test list performance
	start = time.Now()
	keys, err := db.List("perf:*")
	assert.NoError(t, err)
	assert.Len(t, keys, numOperations)
	listTime := time.Since(start)
	
	t.Logf("List performance: %d keys in %v", len(keys), listTime)
}

func TestBuntDB_ErrorHandling(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Test invalid JSON in Get
	err = db.Update(func(tx Transaction) error {
		return tx.SetRaw("invalid:json", "invalid json content")
	})
	assert.NoError(t, err)

	var result map[string]interface{}
	err = db.Get("invalid:json", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")

	// Test setting nil value
	err = db.Set("nil:test", nil)
	assert.NoError(t, err)

	var nilResult interface{}
	err = db.Get("nil:test", &nilResult)
	assert.NoError(t, err)
	assert.Nil(t, nilResult)
}

func TestBuntDB_Close(t *testing.T) {
	db, err := NewBuntDB(":memory:")
	require.NoError(t, err)

	// Add some data
	err = db.Set("test:key", "test_value")
	assert.NoError(t, err)

	// Close the database
	err = db.Close()
	assert.NoError(t, err)

	// Operations after close should fail gracefully
	err = db.Set("test:key2", "test_value2")
	assert.Error(t, err)

	var value string
	err = db.Get("test:key", &value)
	assert.Error(t, err)
}