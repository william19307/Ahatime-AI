package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type seedanceAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupSeedanceControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	model.DB = db
	if err := db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.SeedanceAssetGroup{},
		&model.SeedanceAsset{},
		&model.SeedanceUpload{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	userA := model.User{Id: 1, Username: "user_a", Password: "password123", Role: 1, Status: 1, Group: "default", AffCode: "aff_a"}
	userB := model.User{Id: 2, Username: "user_b", Password: "password123", Role: 1, Status: 1, Group: "default", AffCode: "aff_b"}
	if err := db.Create(&userA).Error; err != nil {
		t.Fatalf("create userA: %v", err)
	}
	if err := db.Create(&userB).Error; err != nil {
		t.Fatalf("create userB: %v", err)
	}
	baseURL := "https://agentrs.jd.com"
	channel := model.Channel{
		Id:      1,
		Type:    constant.ChannelTypeJDSeedance,
		Key:     "test-key",
		Status:  common.ChannelStatusEnabled,
		Name:    "jd-seedance",
		Group:   "default",
		BaseURL: &baseURL,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func installMockSeedanceService(t *testing.T) *service.SeedanceAssetService {
	t.Helper()
	mock := newMockSeedanceControllerUpstream()
	svc := service.NewSeedanceAssetService()
	svc.NewClient = func(baseURL, apiKey string) service.SeedanceAssetUpstream {
		return mock
	}
	seedanceAssetService = svc
	t.Cleanup(func() {
		seedanceAssetService = service.NewSeedanceAssetService()
	})
	return svc
}

type mockSeedanceControllerUpstream struct {
	nextGroup int
	nextAsset int
}

func newMockSeedanceControllerUpstream() *mockSeedanceControllerUpstream {
	return &mockSeedanceControllerUpstream{}
}

func (m *mockSeedanceControllerUpstream) CreateAssetGroup(name, description, groupType string) (string, error) {
	m.nextGroup++
	return fmt.Sprintf("group-%d", m.nextGroup), nil
}

func (m *mockSeedanceControllerUpstream) UpdateAssetGroup(id, name, description string) error {
	return nil
}

func (m *mockSeedanceControllerUpstream) CreateAsset(groupId, assetURL, assetType, name string) (string, error) {
	m.nextAsset++
	return fmt.Sprintf("asset-%d", m.nextAsset), nil
}

func (m *mockSeedanceControllerUpstream) GetAsset(id string) (*service.SeedanceGetAssetResult, error) {
	return &service.SeedanceGetAssetResult{
		Id:     id,
		URL:    "https://cdn.example.com/" + id,
		Status: "ready",
		Name:   "mock",
	}, nil
}

func (m *mockSeedanceControllerUpstream) UpdateAsset(id, name string) error {
	return nil
}

func (m *mockSeedanceControllerUpstream) DeleteAsset(id string) error {
	return nil
}

func newSeedanceAuthContext(t *testing.T, method, target string, body any, userID int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		requestBody = bytes.NewReader(payload)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, requestBody)
	if body != nil {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	ctx.Set("id", userID)
	return ctx, recorder
}

func decodeSeedanceAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) seedanceAPIResponse {
	t.Helper()
	var resp seedanceAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	return resp
}

func TestGetSeedanceAssetCrossUserReturns404(t *testing.T) {
	setupSeedanceControllerTestDB(t)
	installMockSeedanceService(t)

	group := &model.SeedanceAssetGroup{
		UserId: 1, UpstreamId: "group-1", Name: "A", GroupType: "AIGC", IsDefault: true,
	}
	if err := model.CreateSeedanceAssetGroup(group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	asset := &model.SeedanceAsset{
		UserId: 1, GroupId: group.Id, UpstreamId: "asset-1", Name: "secret", AssetType: "image",
	}
	if err := model.CreateSeedanceAsset(asset); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	ctx, recorder := newSeedanceAuthContext(t, http.MethodGet, "/api/seedance/assets/1", nil, 2)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	GetSeedanceAsset(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestDeleteSeedanceAssetCrossUserReturns404(t *testing.T) {
	setupSeedanceControllerTestDB(t)
	installMockSeedanceService(t)

	group := &model.SeedanceAssetGroup{
		UserId: 1, UpstreamId: "group-1", Name: "A", GroupType: "AIGC", IsDefault: true,
	}
	if err := model.CreateSeedanceAssetGroup(group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	asset := &model.SeedanceAsset{
		UserId: 1, GroupId: group.Id, UpstreamId: "asset-1", Name: "secret", AssetType: "image",
	}
	if err := model.CreateSeedanceAsset(asset); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	ctx, recorder := newSeedanceAuthContext(t, http.MethodDelete, "/api/seedance/assets/1", nil, 2)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	DeleteSeedanceAsset(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestServeSeedancePublicFileExpiredTokenReturns404(t *testing.T) {
	setupSeedanceControllerTestDB(t)
	upload := &model.SeedanceUpload{
		UserId: 1, FileName: "a.png", MimeType: "image/png", Size: 1,
		StoragePath: "/tmp/not-used", SignedToken: "expired-token", ExpiresAt: time.Now().Unix() - 3600,
	}
	if err := model.CreateSeedanceUpload(upload); err != nil {
		t.Fatalf("create upload: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/seedance/files/public/expired-token", nil)
	ctx.Params = gin.Params{{Key: "token", Value: "expired-token"}}
	ServeSeedancePublicFile(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestListSeedanceAssetGroupsWithoutChannelReturnsError(t *testing.T) {
	db := setupSeedanceControllerTestDB(t)
	if err := db.Where("1 = 1").Delete(&model.Channel{}).Error; err != nil {
		t.Fatalf("delete channels: %v", err)
	}
	installMockSeedanceService(t)

	ctx, recorder := newSeedanceAuthContext(t, http.MethodGet, "/api/seedance/groups", nil, 1)
	ListSeedanceAssetGroups(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 with error payload, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	resp := decodeSeedanceAPIResponse(t, recorder)
	if resp.Success {
		t.Fatalf("expected failure when channel missing, got success")
	}
}
