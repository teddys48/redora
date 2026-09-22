package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redora/redora/backend/internal/crypto"
	"github.com/redora/redora/backend/internal/models"
	"github.com/redora/redora/backend/internal/redis"
	"github.com/redora/redora/backend/internal/repository"
)

type ConnectionService interface {
	GetAll(ctx context.Context) ([]*models.Connection, error)
	GetByID(ctx context.Context, id string) (*models.Connection, error)
	Create(ctx context.Context, input *models.ConnectionCreateInput) (*models.Connection, error)
	Update(ctx context.Context, id string, input *models.ConnectionUpdateInput) (*models.Connection, error)
	Delete(ctx context.Context, id string) error
	TestConnection(ctx context.Context, id string) error
	TestInput(ctx context.Context, input *models.ConnectionCreateInput) error
}

type connectionService struct {
	repo      repository.ConnectionRepository
	encryptor *crypto.Encryptor
	redisMgr  redis.ConnectionManager
}

func NewConnectionService(repo repository.ConnectionRepository, encryptor *crypto.Encryptor, redisMgr redis.ConnectionManager) ConnectionService {
	return &connectionService{
		repo:      repo,
		encryptor: encryptor,
		redisMgr:  redisMgr,
	}
}

func (s *connectionService) GetAll(ctx context.Context) ([]*models.Connection, error) {
	return s.repo.GetAll(ctx)
}

func (s *connectionService) GetByID(ctx context.Context, id string) (*models.Connection, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *connectionService) Create(ctx context.Context, input *models.ConnectionCreateInput) (*models.Connection, error) {
	if input.Name == "" || input.Host == "" {
		return nil, fmt.Errorf("name and host are required")
	}

	if input.Port <= 0 {
		input.Port = 6379
	}

	encryptedPass, err := s.encryptor.Encrypt(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	now := time.Now().UTC()
	conn := &models.Connection{
		ID:                uuid.New().String(),
		Name:              input.Name,
		Host:              input.Host,
		Port:              input.Port,
		Username:          input.Username,
		PasswordEncrypted: encryptedPass,
		DB:                input.DB,
		TLSEnabled:        input.TLSEnabled,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.Create(ctx, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *connectionService) Update(ctx context.Context, id string, input *models.ConnectionUpdateInput) (*models.Connection, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil || existing == nil {
		return nil, fmt.Errorf("connection not found")
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	if input.Host != "" {
		existing.Host = input.Host
	}
	if input.Port > 0 {
		existing.Port = input.Port
	}
	if input.Username != nil {
		existing.Username = *input.Username
	}
	if input.Password != nil && *input.Password != "" {
		enc, err := s.encryptor.Encrypt(*input.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt updated password: %w", err)
		}
		existing.PasswordEncrypted = enc
	}
	existing.DB = input.DB
	existing.TLSEnabled = input.TLSEnabled
	existing.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	// Close old cached client so it reconnects with new settings
	_ = s.redisMgr.Close(id)

	return existing, nil
}

func (s *connectionService) Delete(ctx context.Context, id string) error {
	_ = s.redisMgr.Close(id)
	return s.repo.Delete(ctx, id)
}

func (s *connectionService) TestConnection(ctx context.Context, id string) error {
	conn, err := s.repo.GetByID(ctx, id)
	if err != nil || conn == nil {
		return fmt.Errorf("connection not found")
	}

	rawPass, err := s.encryptor.Decrypt(conn.PasswordEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	return s.redisMgr.TestConnection(ctx, conn, rawPass)
}

func (s *connectionService) TestInput(ctx context.Context, input *models.ConnectionCreateInput) error {
	if input.Host == "" {
		return fmt.Errorf("host is required")
	}
	port := input.Port
	if port <= 0 {
		port = 6379
	}

	rawPass := input.Password
	if rawPass == "" && input.ID != "" {
		if conn, err := s.repo.GetByID(ctx, input.ID); err == nil && conn != nil {
			if decrypted, err := s.encryptor.Decrypt(conn.PasswordEncrypted); err == nil {
				rawPass = decrypted
			}
		}
	}

	dummyConn := &models.Connection{
		ID:         "test",
		Host:       input.Host,
		Port:       port,
		Username:   input.Username,
		DB:         input.DB,
		TLSEnabled: input.TLSEnabled,
	}

	return s.redisMgr.TestConnection(ctx, dummyConn, rawPass)
}
