package infra

import (
	"context"
	"testing"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
)

const (
	// Test configuration - adjust these to match your test environment
	testDragonflyAddr = "localhost:6379"
	testDragonflyDB   = 15 // Use separate DB for testing
)

func TestNewDragonflyClient_Success(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     testDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testDragonflyDB,
	}

	client := NewDragonflyClient(cfg)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	defer Close(client)

	// Try to ping to verify connection works
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}
}

func TestNewDragonflyClient_ConnectionFails(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     "localhost:9999", // Wrong port
		DragonflyPassword: "",
		DragonflyDB:       0,
	}

	client := NewDragonflyClient(cfg)
	if client == nil {
		t.Fatal("Expected non-nil client even with invalid config")
	}
	defer Close(client)

	// Ping should fail
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := Ping(ctx, client); err == nil {
		t.Error("Expected ping to fail with invalid config, got nil")
	}
}

func TestDragonflyClient_PingSuccess(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     testDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testDragonflyDB,
	}

	client := NewDragonflyClient(cfg)
	defer Close(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}
}

func TestDragonflyClient_JSONSetGet(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     testDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testDragonflyDB,
	}

	client := NewDragonflyClient(cfg)
	defer Close(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connection
	if err := Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}

	// Test data structure
	type TestStruct struct {
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
	}

	testData := TestStruct{
		Name:      "Test User",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
	}

	key := "test:json:set:get:" + time.Now().Format("20060102150405")

	// Test Set
	if err := JSONSet(ctx, client, key, testData); err != nil {
		t.Fatalf("JSONSet failed: %v", err)
	}

	// Test Get
	var retrieved TestStruct
	if err := JSONGet(ctx, client, key, &retrieved); err != nil {
		t.Fatalf("JSONGet failed: %v", err)
	}

	// Verify data
	if retrieved.Name != testData.Name {
		t.Errorf("Expected Name %s, got %s", testData.Name, retrieved.Name)
	}
	if retrieved.Email != testData.Email {
		t.Errorf("Expected Email %s, got %s", testData.Email, retrieved.Email)
	}

	// Clean up
	if err := JSONDelete(ctx, client, key); err != nil {
		t.Errorf("Failed to clean up test key: %v", err)
	}
}

func TestDragonflyClient_JSONSetWithTTL(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     testDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testDragonflyDB,
	}

	client := NewDragonflyClient(cfg)
	defer Close(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connection
	if err := Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}

	testData := map[string]string{"key": "value"}
	key := "test:json:set:ttl:" + time.Now().Format("20060102150405")
	ttl := 2 * time.Second

	// Set with TTL
	if err := JSONSetWithTTL(ctx, client, key, testData, ttl); err != nil {
		t.Fatalf("JSONSetWithTTL failed: %v", err)
	}

	// Should exist immediately
	exists, err := JSONExists(ctx, client, key)
	if err != nil {
		t.Fatalf("JSONExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(ttl + 500*time.Millisecond)

	// Should no longer exist
	exists, err = JSONExists(ctx, client, key)
	if err != nil {
		t.Fatalf("JSONExists failed: %v", err)
	}
	if exists {
		t.Error("Expected key to expire after TTL")
	}
}

func TestDragonflyClient_JSONSetNX(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     testDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testDragonflyDB,
	}

	client := NewDragonflyClient(cfg)
	defer Close(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connection
	if err := Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}

	testData := map[string]string{"key": "value"}
	key := "test:json:set:nx:" + time.Now().Format("20060102150405")

	// First set should succeed
	created, err := JSONSetNX(ctx, client, key, testData)
	if err != nil {
		t.Fatalf("JSONSetNX failed: %v", err)
	}
	if !created {
		t.Error("Expected first JSONSetNX to return true")
	}

	// Second set should fail (key already exists)
	created, err = JSONSetNX(ctx, client, key, testData)
	if err != nil {
		t.Fatalf("JSONSetNX failed: %v", err)
	}
	if created {
		t.Error("Expected second JSONSetNX to return false")
	}

	// Clean up
	if err := JSONDelete(ctx, client, key); err != nil {
		t.Errorf("Failed to clean up test key: %v", err)
	}
}

func TestDragonflyClient_JSONDelete(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     testDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testDragonflyDB,
	}

	client := NewDragonflyClient(cfg)
	defer Close(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connection
	if err := Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}

	testData := map[string]string{"key": "value"}
	key := "test:json:delete:" + time.Now().Format("20060102150405")

	// Set the key
	if err := JSONSet(ctx, client, key, testData); err != nil {
		t.Fatalf("JSONSet failed: %v", err)
	}

	// Verify it exists
	exists, err := JSONExists(ctx, client, key)
	if err != nil {
		t.Fatalf("JSONExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist before delete")
	}

	// Delete the key
	if err := JSONDelete(ctx, client, key); err != nil {
		t.Fatalf("JSONDelete failed: %v", err)
	}

	// Verify it's gone
	exists, err = JSONExists(ctx, client, key)
	if err != nil {
		t.Fatalf("JSONExists failed: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist after delete")
	}
}

func TestDragonflyClient_JSONGetMultiple(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     testDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testDragonflyDB,
	}

	client := NewDragonflyClient(cfg)
	defer Close(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connection
	if err := Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}

	prefix := "test:json:multiple:" + time.Now().Format("20060102150405")
	keys := []string{
		prefix + ":1",
		prefix + ":2",
		prefix + ":3",
	}

	testData := map[string]string{"key": "value"}

	// Set all keys
	for _, key := range keys {
		if err := JSONSet(ctx, client, key, testData); err != nil {
			t.Fatalf("JSONSet failed for key %s: %v", key, err)
		}
	}

	// Clean up deferred
	defer func() {
		for _, key := range keys {
			JSONDelete(ctx, client, key)
		}
	}()

	// Get multiple keys
	result, err := JSONGetMultiple(ctx, client, keys)
	if err != nil {
		t.Fatalf("JSONGetMultiple failed: %v", err)
	}

	// Verify we got all keys
	if len(result) != len(keys) {
		t.Errorf("Expected %d keys, got %d", len(keys), len(result))
	}

	for _, key := range keys {
		if _, exists := result[key]; !exists {
			t.Errorf("Expected key %s to exist in result", key)
		}
	}
}

func TestDragonflyClient_JSONExists(t *testing.T) {
	cfg := &domain.Config{
		DragonflyAddr:     testDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testDragonflyDB,
	}

	client := NewDragonflyClient(cfg)
	defer Close(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connection
	if err := Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}

	testData := map[string]string{"key": "value"}
	key := "test:json:exists:" + time.Now().Format("20060102150405")

	// Should not exist initially
	exists, err := JSONExists(ctx, client, key)
	if err != nil {
		t.Fatalf("JSONExists failed: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist initially")
	}

	// Set the key
	if err := JSONSet(ctx, client, key, testData); err != nil {
		t.Fatalf("JSONSet failed: %v", err)
	}

	// Should exist now
	exists, err = JSONExists(ctx, client, key)
	if err != nil {
		t.Fatalf("JSONExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist after set")
	}

	// Clean up
	if err := JSONDelete(ctx, client, key); err != nil {
		t.Errorf("Failed to clean up test key: %v", err)
	}
}
