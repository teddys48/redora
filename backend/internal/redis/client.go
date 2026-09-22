package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redora/redora/backend/internal/models"
)

type ConnectionManager interface {
	GetClient(ctx context.Context, conn *models.Connection, rawPassword string) (*redis.Client, error)
	TestConnection(ctx context.Context, conn *models.Connection, rawPassword string) error
	Close(id string) error
	CloseAll()
}

type connectionManager struct {
	mu      sync.RWMutex
	clients map[string]*redis.Client
}

func NewConnectionManager() ConnectionManager {
	return &connectionManager{
		clients: make(map[string]*redis.Client),
	}
}

func maskPassword(pass string) string {
	if pass == "" {
		return "[NONE]"
	}
	return "***"
}

func (m *connectionManager) GetClient(ctx context.Context, conn *models.Connection, rawPassword string) (*redis.Client, error) {
	m.mu.RLock()
	client, ok := m.clients[conn.ID]
	m.mu.RUnlock()

	if ok && client != nil {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := client.Ping(pingCtx).Err(); err == nil {
			return client, nil
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.clients[conn.ID]; ok && client != nil {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := client.Ping(pingCtx).Err(); err == nil {
			return client, nil
		}
		_ = client.Close()
	}

	var tlsConfig *tls.Config
	if conn.TLSEnabled {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	slog.Info("Creating pooled Redis client connection",
		"connection_id", conn.ID,
		"connection_name", conn.Name,
		"host", conn.Host,
		"port", conn.Port,
		"username", conn.Username,
		"db", conn.DB,
		"tls_enabled", conn.TLSEnabled,
		"password", maskPassword(rawPassword),
	)

	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", conn.Host, conn.Port),
		Username:     conn.Username,
		Password:     rawPassword,
		DB:           conn.DB,
		TLSConfig:    tlsConfig,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	newClient := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := newClient.Ping(pingCtx).Err(); err != nil {
		slog.Error("Failed to connect pooled Redis client",
			"connection_id", conn.ID,
			"host", conn.Host,
			"port", conn.Port,
			"error", err,
		)
		_ = newClient.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	m.clients[conn.ID] = newClient
	return newClient, nil
}

func (m *connectionManager) TestConnection(ctx context.Context, conn *models.Connection, rawPassword string) error {
	slog.Info("Executing Redis connection test",
		"host", conn.Host,
		"port", conn.Port,
		"username", conn.Username,
		"db", conn.DB,
		"tls_enabled", conn.TLSEnabled,
		"password", maskPassword(rawPassword),
	)

	var tlsConfig *tls.Config
	if conn.TLSEnabled {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	opts := &redis.Options{
		Addr:        fmt.Sprintf("%s:%d", conn.Host, conn.Port),
		Username:    conn.Username,
		Password:    rawPassword,
		DB:          conn.DB,
		TLSConfig:   tlsConfig,
		DialTimeout: 3 * time.Second,
	}

	client := redis.NewClient(opts)
	defer client.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := client.Ping(pingCtx).Err()
	if err != nil {
		slog.Error("Redis connection test failed",
			"host", conn.Host,
			"port", conn.Port,
			"username", conn.Username,
			"db", conn.DB,
			"tls_enabled", conn.TLSEnabled,
			"error", err,
		)
	} else {
		slog.Info("Redis connection test succeeded (PONG)",
			"host", conn.Host,
			"port", conn.Port,
			"username", conn.Username,
			"db", conn.DB,
		)
	}

	return err
}

func (m *connectionManager) Close(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.clients[id]; ok {
		delete(m.clients, id)
		return client.Close()
	}
	return nil
}

func (m *connectionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, client := range m.clients {
		_ = client.Close()
		delete(m.clients, id)
	}
}
