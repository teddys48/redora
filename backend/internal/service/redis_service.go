package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redora/redora/backend/internal/crypto"
	"github.com/redora/redora/backend/internal/models"
	redisClient "github.com/redora/redora/backend/internal/redis"
	"github.com/redora/redora/backend/internal/repository"
)

type RedisService interface {
	ScanKeys(ctx context.Context, connID, pattern, typeFilter string, cursor uint64, limit int64) (*models.ScanKeysResponse, error)
	GetKeyDetail(ctx context.Context, connID, key string) (*models.KeyDetailResponse, error)
	CreateKey(ctx context.Context, connID string, input *models.CreateKeyInput) error
	UpdateKey(ctx context.Context, connID, key string, input *models.CreateKeyInput) error
	DeleteKey(ctx context.Context, connID, key string) error
	RenameKey(ctx context.Context, connID, oldKey, newKey string) error
	SetTTL(ctx context.Context, connID, key string, ttl int64) error
	ExecuteCommand(ctx context.Context, connID, commandStr string) (*models.ExecuteCommandResponse, error)
	ListPubSubChannels(ctx context.Context, connID, pattern string) ([]string, error)
	PublishMessage(ctx context.Context, connID, channel, message string) (int64, error)
}

type redisService struct {
	repo      repository.ConnectionRepository
	encryptor *crypto.Encryptor
	redisMgr  redisClient.ConnectionManager
}

func NewRedisService(repo repository.ConnectionRepository, encryptor *crypto.Encryptor, redisMgr redisClient.ConnectionManager) RedisService {
	return &redisService{
		repo:      repo,
		encryptor: encryptor,
		redisMgr:  redisMgr,
	}
}

func (s *redisService) getClient(ctx context.Context, connID string) (*redis.Client, error) {
	conn, err := s.repo.GetByID(ctx, connID)
	if err != nil || conn == nil {
		return nil, fmt.Errorf("connection not found")
	}

	rawPass, err := s.encryptor.Decrypt(conn.PasswordEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	return s.redisMgr.GetClient(ctx, conn, rawPass)
}

func (s *redisService) ScanKeys(ctx context.Context, connID, pattern, typeFilter string, cursor uint64, limit int64) (*models.ScanKeysResponse, error) {
	client, err := s.getClient(ctx, connID)
	if err != nil {
		return nil, err
	}

	if pattern == "" {
		pattern = "*"
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	typeFilter = strings.ToLower(strings.TrimSpace(typeFilter))

	var keys []string
	var nextCursor uint64
	var scanErr error

	// Use ScanType if supported and typeFilter is specified
	if typeFilter != "" {
		keys, nextCursor, scanErr = client.ScanType(ctx, cursor, pattern, limit, typeFilter).Result()
	} else {
		keys, nextCursor, scanErr = client.Scan(ctx, cursor, pattern, limit).Result()
	}

	if scanErr != nil {
		return nil, fmt.Errorf("redis SCAN failed: %w", scanErr)
	}

	// Fetch metadata (Type & TTL) via Redis Pipeline
	items := make([]models.KeyItem, 0, len(keys))
	if len(keys) > 0 {
		pipe := client.Pipeline()
		typeCmds := make([]*redis.StatusCmd, len(keys))
		ttlCmds := make([]*redis.DurationCmd, len(keys))

		for i, k := range keys {
			typeCmds[i] = pipe.Type(ctx, k)
			ttlCmds[i] = pipe.TTL(ctx, k)
		}

		_, _ = pipe.Exec(ctx)

		for i, k := range keys {
			keyType := typeCmds[i].Val()
			ttlVal := ttlCmds[i].Val()

			// Additional filter check if ScanType didn't filter
			if typeFilter != "" && strings.ToLower(keyType) != typeFilter {
				continue
			}

			ttlSec := int64(-1)
			if ttlVal > 0 {
				ttlSec = int64(ttlVal.Seconds())
			} else if ttlVal == -2*time.Second {
				ttlSec = -2
			}

			items = append(items, models.KeyItem{
				Key:  k,
				Type: keyType,
				TTL:  ttlSec,
			})
		}
	}

	return &models.ScanKeysResponse{
		Keys:       items,
		NextCursor: nextCursor,
	}, nil
}

func (s *redisService) GetKeyDetail(ctx context.Context, connID, key string) (*models.KeyDetailResponse, error) {
	client, err := s.getClient(ctx, connID)
	if err != nil {
		return nil, err
	}

	keyType, err := client.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get key type: %w", err)
	}

	if keyType == "none" {
		return nil, fmt.Errorf("key '%s' not found", key)
	}

	ttlVal, err := client.TTL(ctx, key).Result()
	if err != nil {
		ttlVal = -1
	}

	ttlSec := int64(-1)
	if ttlVal > 0 {
		ttlSec = int64(ttlVal.Seconds())
	} else if ttlVal == -2*time.Second {
		ttlSec = -2
	}

	var val interface{}
	switch keyType {
	case "string":
		v, err := client.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		// Bound maximum value size to 512KB
		if len(v) > 524288 {
			v = v[:524288] + "... [TRUNCATED > 512KB]"
		}
		val = v

	case "hash":
		res, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		val = res

	case "list":
		// Paginate list elements up to 200 items
		res, err := client.LRange(ctx, key, 0, 199).Result()
		if err != nil {
			return nil, err
		}
		val = res

	case "set":
		res, err := client.SMembers(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		val = res

	case "zset":
		res, err := client.ZRangeWithScores(ctx, key, 0, 199).Result()
		if err != nil {
			return nil, err
		}
		zItems := make([]map[string]interface{}, len(res))
		for i, z := range res {
			zItems[i] = map[string]interface{}{
				"member": z.Member,
				"score":  z.Score,
			}
		}
		val = zItems

	case "stream":
		res, err := client.XRangeN(ctx, key, "-", "+", 100).Result()
		if err != nil {
			return nil, err
		}
		streamItems := make([]map[string]interface{}, len(res))
		for i, entry := range res {
			streamItems[i] = map[string]interface{}{
				"id":     entry.ID,
				"values": entry.Values,
			}
		}
		val = streamItems

	default:
		val = fmt.Sprintf("[Unsupported type: %s]", keyType)
	}

	return &models.KeyDetailResponse{
		Key:   key,
		Type:  keyType,
		TTL:   ttlSec,
		Value: val,
	}, nil
}

func (s *redisService) CreateKey(ctx context.Context, connID string, input *models.CreateKeyInput) error {
	client, err := s.getClient(ctx, connID)
	if err != nil {
		return err
	}

	if input.Key == "" {
		return fmt.Errorf("key name cannot be empty")
	}

	expiration := time.Duration(0)
	if input.TTL > 0 {
		expiration = time.Duration(input.TTL) * time.Second
	}

	keyType := strings.ToLower(input.Type)
	switch keyType {
	case "string":
		valStr := fmt.Sprintf("%v", input.Value)
		return client.Set(ctx, input.Key, valStr, expiration).Err()

	case "hash":
		if input.Field != "" {
			valStr := fmt.Sprintf("%v", input.Value)
			err = client.HSet(ctx, input.Key, input.Field, valStr).Err()
		} else if m, ok := input.Value.(map[string]interface{}); ok {
			err = client.HSet(ctx, input.Key, m).Err()
		} else {
			return fmt.Errorf("invalid hash value payload")
		}

	case "list":
		valStr := fmt.Sprintf("%v", input.Value)
		err = client.RPush(ctx, input.Key, valStr).Err()

	case "set":
		valStr := fmt.Sprintf("%v", input.Value)
		err = client.SAdd(ctx, input.Key, valStr).Err()

	case "zset":
		valStr := fmt.Sprintf("%v", input.Value)
		err = client.ZAdd(ctx, input.Key, redis.Z{
			Score:  input.Score,
			Member: valStr,
		}).Err()

	default:
		return fmt.Errorf("unsupported key type: %s", input.Type)
	}

	if err != nil {
		return err
	}

	if expiration > 0 {
		_ = client.Expire(ctx, input.Key, expiration)
	}

	return nil
}

func (s *redisService) UpdateKey(ctx context.Context, connID, key string, input *models.CreateKeyInput) error {
	client, err := s.getClient(ctx, connID)
	if err != nil {
		return err
	}

	keyType, err := client.Type(ctx, key).Result()
	if err != nil || keyType == "none" {
		return fmt.Errorf("key '%s' does not exist", key)
	}

	switch keyType {
	case "string":
		valStr := fmt.Sprintf("%v", input.Value)
		ttlVal, _ := client.TTL(ctx, key).Result()
		return client.Set(ctx, key, valStr, ttlVal).Err()

	case "hash":
		if input.Field == "" {
			return fmt.Errorf("field is required for updating hash key")
		}
		valStr := fmt.Sprintf("%v", input.Value)
		return client.HSet(ctx, key, input.Field, valStr).Err()

	case "list":
		valStr := fmt.Sprintf("%v", input.Value)
		return client.RPush(ctx, key, valStr).Err()

	case "set":
		valStr := fmt.Sprintf("%v", input.Value)
		return client.SAdd(ctx, key, valStr).Err()

	case "zset":
		valStr := fmt.Sprintf("%v", input.Value)
		return client.ZAdd(ctx, key, redis.Z{
			Score:  input.Score,
			Member: valStr,
		}).Err()

	default:
		return fmt.Errorf("cannot update key of type: %s", keyType)
	}
}

func (s *redisService) DeleteKey(ctx context.Context, connID, key string) error {
	client, err := s.getClient(ctx, connID)
	if err != nil {
		return err
	}
	return client.Del(ctx, key).Err()
}

func (s *redisService) RenameKey(ctx context.Context, connID, oldKey, newKey string) error {
	if newKey == "" || oldKey == newKey {
		return fmt.Errorf("invalid new key name")
	}
	client, err := s.getClient(ctx, connID)
	if err != nil {
		return err
	}
	return client.Rename(ctx, oldKey, newKey).Err()
}

func (s *redisService) SetTTL(ctx context.Context, connID, key string, ttl int64) error {
	client, err := s.getClient(ctx, connID)
	if err != nil {
		return err
	}

	if ttl < 0 {
		return client.Persist(ctx, key).Err()
	}

	return client.Expire(ctx, key, time.Duration(ttl)*time.Second).Err()
}

func (s *redisService) ExecuteCommand(ctx context.Context, connID, commandStr string) (*models.ExecuteCommandResponse, error) {
	commandStr = strings.TrimSpace(commandStr)
	if commandStr == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	client, err := s.getClient(ctx, connID)
	if err != nil {
		return nil, err
	}

	// Parse command line arguments
	args := strings.Fields(commandStr)
	if len(args) == 0 {
		return nil, fmt.Errorf("invalid command format")
	}

	interfaceArgs := make([]interface{}, len(args))
	for i, v := range args {
		interfaceArgs[i] = v
	}

	start := time.Now()
	resCmd := client.Do(ctx, interfaceArgs...)
	duration := time.Since(start)

	res, cmdErr := resCmd.Result()
	execTimeStr := fmt.Sprintf("%.2fms", float64(duration.Microseconds())/1000.0)

	if cmdErr != nil {
		slog.Warn("CLI command returned error", "conn_id", connID, "cmd", args[0], "error", cmdErr)
		return &models.ExecuteCommandResponse{
			Result:    fmt.Sprintf("ERR: %v", cmdErr),
			Execution: execTimeStr,
			Status:    "error",
		}, nil
	}

	return &models.ExecuteCommandResponse{
		Result:    formatRedisResult(res),
		Execution: execTimeStr,
		Status:    "ok",
	}, nil
}

func formatRedisResult(v interface{}) string {
	if v == nil {
		return "(nil)"
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case []byte:
		return fmt.Sprintf("%q", string(val))
	case int64:
		return fmt.Sprintf("(integer) %d", val)
	case []interface{}:
		if len(val) == 0 {
			return "(empty array)"
		}
		var sb strings.Builder
		for i, elem := range val {
			sb.WriteString(fmt.Sprintf("%d) %s\n", i+1, formatRedisResult(elem)))
		}
		return strings.TrimSuffix(sb.String(), "\n")
	case map[string]interface{}:
		var sb strings.Builder
		for k, elem := range val {
			sb.WriteString(fmt.Sprintf("%s -> %s\n", k, formatRedisResult(elem)))
		}
		return strings.TrimSuffix(sb.String(), "\n")
	default:
		return fmt.Sprintf("%v", val)
	}
}

func (s *redisService) ListPubSubChannels(ctx context.Context, connID, pattern string) ([]string, error) {
	client, err := s.getClient(ctx, connID)
	if err != nil {
		return nil, err
	}

	if pattern == "" {
		pattern = "*"
	}

	return client.PubSubChannels(ctx, pattern).Result()
}

func (s *redisService) PublishMessage(ctx context.Context, connID, channel, message string) (int64, error) {
	if channel == "" {
		return 0, fmt.Errorf("channel name required")
	}

	client, err := s.getClient(ctx, connID)
	if err != nil {
		return 0, err
	}

	return client.Publish(ctx, channel, message).Result()
}
