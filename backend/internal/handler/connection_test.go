package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/redora/redora/backend/internal/crypto"
	"github.com/redora/redora/backend/internal/database"
	"github.com/redora/redora/backend/internal/handler"
	"github.com/redora/redora/backend/internal/models"
	"github.com/redora/redora/backend/internal/redis"
	"github.com/redora/redora/backend/internal/repository"
	"github.com/redora/redora/backend/internal/service"
)

func TestConnectionCRUD(t *testing.T) {
	// Setup temporary SQLite database
	dbFile := "./test_redora.db"
	defer os.Remove(dbFile)

	db, err := database.InitDB(dbFile)
	if err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	encryptor := crypto.NewEncryptor("test-secret-key-32-bytes-long!")
	redisMgr := redis.NewConnectionManager()
	defer redisMgr.CloseAll()

	connRepo := repository.NewConnectionRepository(db)
	connSvc := service.NewConnectionService(connRepo, encryptor, redisMgr)
	connHandler := handler.NewConnectionHandler(connSvc)

	app := fiber.New()
	apiGroup := app.Group("/api")
	connHandler.RegisterRoutes(apiGroup)

	// 1. Create Connection
	createPayload := models.ConnectionCreateInput{
		Name:       "Test Redis Instance",
		Host:       "127.0.0.1",
		Port:       6379,
		Username:   "default",
		Password:   "MySecretPassword",
		DB:         0,
		TLSEnabled: false,
	}

	bodyBytes, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/connections", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute POST /api/connections: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var res map[string]*models.Connection
	if err := json.Unmarshal(respBody, &res); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	createdConn := res["data"]
	if createdConn == nil || createdConn.ID == "" {
		t.Fatal("Expected created connection with non-empty ID")
	}

	if createdConn.Name != "Test Redis Instance" {
		t.Errorf("Expected name 'Test Redis Instance', got '%s'", createdConn.Name)
	}

	// 2. Get All Connections
	reqGet := httptest.NewRequest(http.MethodGet, "/api/connections", nil)
	respGet, err := app.Test(reqGet, -1)
	if err != nil {
		t.Fatalf("Failed to execute GET /api/connections: %v", err)
	}

	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", respGet.StatusCode)
	}

	respGetBody, _ := io.ReadAll(respGet.Body)
	var getRes map[string][]*models.Connection
	json.Unmarshal(respGetBody, &getRes)

	if len(getRes["data"]) != 1 {
		t.Fatalf("Expected 1 connection, got %d", len(getRes["data"]))
	}

	// 3. Update Connection without password (verify password preservation)
	updatePayload := models.ConnectionUpdateInput{
		Name: "Updated Redis Name",
		Host: "127.0.0.1",
		Port: 6379,
		DB:   1,
	}
	upBytes, _ := json.Marshal(updatePayload)
	reqUp := httptest.NewRequest(http.MethodPut, "/api/connections/"+createdConn.ID, bytes.NewReader(upBytes))
	reqUp.Header.Set("Content-Type", "application/json")
	respUp, err := app.Test(reqUp, -1)
	if err != nil {
		t.Fatalf("Failed to execute PUT /api/connections: %v", err)
	}
	if respUp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", respUp.StatusCode)
	}

	// Verify database password is still intact and decodable
	storedConn, err := connRepo.GetByID(reqUp.Context(), createdConn.ID)
	if err != nil || storedConn == nil {
		t.Fatalf("Failed to retrieve stored connection: %v", err)
	}
	decryptedPass, err := encryptor.Decrypt(storedConn.PasswordEncrypted)
	if err != nil || decryptedPass != "MySecretPassword" {
		t.Fatalf("Expected password to remain 'MySecretPassword', got '%s' (err: %v)", decryptedPass, err)
	}

	// 4. Delete Connection
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/connections/"+createdConn.ID, nil)
	respDel, err := app.Test(reqDel, -1)
	if err != nil {
		t.Fatalf("Failed to execute DELETE /api/connections: %v", err)
	}

	if respDel.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", respDel.StatusCode)
	}
}
