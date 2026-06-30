package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type mockSeedanceUpstream struct {
	groups  map[string]string
	assets  map[string]*SeedanceGetAssetResult
	nextGID int
	nextAID int
}

func newMockSeedanceUpstream() *mockSeedanceUpstream {
	return &mockSeedanceUpstream{
		groups: make(map[string]string),
		assets: make(map[string]*SeedanceGetAssetResult),
	}
}

func (m *mockSeedanceUpstream) CreateAssetGroup(name, description, groupType string) (string, error) {
	m.nextGID++
	id := fmt.Sprintf("group-test-%d", m.nextGID)
	m.groups[id] = name
	return id, nil
}

func (m *mockSeedanceUpstream) UpdateAssetGroup(id, name, description string) error {
	m.groups[id] = name
	return nil
}

func (m *mockSeedanceUpstream) CreateAsset(groupId, assetURL, assetType, name string) (string, error) {
	m.nextAID++
	id := fmt.Sprintf("asset-test-%d", m.nextAID)
	m.assets[id] = &SeedanceGetAssetResult{
		Id:        id,
		GroupId:   groupId,
		Name:      name,
		AssetType: assetType,
		URL:       assetURL,
		Status:    "ready",
	}
	return id, nil
}

func (m *mockSeedanceUpstream) GetAsset(id string) (*SeedanceGetAssetResult, error) {
	asset, ok := m.assets[id]
	if !ok {
		return nil, model.ErrSeedanceAssetNotFound
	}
	return asset, nil
}

func (m *mockSeedanceUpstream) UpdateAsset(id, name string) error {
	if asset, ok := m.assets[id]; ok {
		asset.Name = name
	}
	return nil
}

func (m *mockSeedanceUpstream) DeleteAsset(id string) error {
	delete(m.assets, id)
	return nil
}

func setupSeedanceServiceTestDB(t *testing.T) {
	t.Helper()
	common.UsingSQLite = true
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
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
		Id:     1,
		Type:   constant.ChannelTypeJDSeedance,
		Key:    "test-key",
		Status: common.ChannelStatusEnabled,
		Name:   "jd-seedance",
		Group:  "default",
		BaseURL: &baseURL,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
}

func newTestSeedanceService(mock *mockSeedanceUpstream) *SeedanceAssetService {
	svc := NewSeedanceAssetService()
	svc.NewClient = func(baseURL, apiKey string) SeedanceAssetUpstream {
		return mock
	}
	return svc
}

func TestSeedanceAssetIsolationBetweenUsers(t *testing.T) {
	setupSeedanceServiceTestDB(t)
	mock := newMockSeedanceUpstream()
	svc := newTestSeedanceService(mock)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	groupA, err := svc.CreateGroup(1, "group-a", "", "AIGC", true)
	if err != nil {
		t.Fatalf("create group A: %v", err)
	}
	assetA, err := svc.CreateAsset(1, CreateSeedanceAssetInput{
		GroupID:   groupA.Id,
		URL:       server.URL + "/a.jpg",
		AssetType: "image",
		Name:      "asset-a",
	}, "https://api.example.com")
	if err != nil {
		t.Fatalf("create asset A: %v", err)
	}

	if _, err := svc.GetAsset(2, assetA.Id, false); err == nil {
		t.Fatal("user B should not read user A asset")
	}
	if _, err := svc.UpdateAsset(2, assetA.Id, "hijack"); err == nil {
		t.Fatal("user B should not update user A asset")
	}
	if err := svc.DeleteAsset(2, assetA.Id); err == nil {
		t.Fatal("user B should not delete user A asset")
	}

	assetsB, totalB, err := svc.ListAssets(2, model.SeedanceAssetQuery{}, 0, 20)
	if err != nil {
		t.Fatalf("list assets B: %v", err)
	}
	if totalB != 0 || len(assetsB) != 0 {
		t.Fatalf("user B list leaked assets: %+v", assetsB)
	}

	groupsB, totalGroupsB, err := svc.ListGroups(2, 0, 20)
	if err != nil {
		t.Fatalf("list groups B: %v", err)
	}
	if totalGroupsB != 1 {
		t.Fatalf("user B should only see own default group, got %d", totalGroupsB)
	}
	if len(groupsB) != 1 || groupsB[0].UserId != 2 {
		t.Fatalf("unexpected groups for user B: %+v", groupsB)
	}

	if _, err := svc.UpdateGroup(2, groupA.Id, "stolen", ""); err == nil {
		t.Fatal("user B should not update user A group")
	}
}

func TestServePublicFileExpiredUpload(t *testing.T) {
	setupSeedanceServiceTestDB(t)
	svc := NewSeedanceAssetService()
	upload := &model.SeedanceUpload{
		UserId: 1, FileName: "a.png", MimeType: "image/png", Size: 1,
		StoragePath: "/tmp/missing-seedance-file", SignedToken: "expired-token",
		ExpiresAt: time.Now().Unix() - 3600,
	}
	if err := model.CreateSeedanceUpload(upload); err != nil {
		t.Fatalf("create upload: %v", err)
	}
	if _, _, err := svc.ServePublicFile("expired-token"); err == nil {
		t.Fatal("expected expired upload error")
	}
}

func TestSeedanceListAssetsScopedByUser(t *testing.T) {
	setupSeedanceServiceTestDB(t)
	mock := newMockSeedanceUpstream()
	svc := newTestSeedanceService(mock)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	groupA, _ := svc.CreateGroup(1, "group-a", "", "AIGC", true)
	groupB, _ := svc.CreateGroup(2, "group-b", "", "AIGC", true)
	_, _ = svc.CreateAsset(1, CreateSeedanceAssetInput{
		GroupID: groupA.Id, URL: server.URL + "/1.jpg", AssetType: "image", Name: "one",
	}, "https://api.example.com")
	_, _ = svc.CreateAsset(2, CreateSeedanceAssetInput{
		GroupID: groupB.Id, URL: server.URL + "/2.jpg", AssetType: "image", Name: "two",
	}, "https://api.example.com")

	assetsA, totalA, err := svc.ListAssets(1, model.SeedanceAssetQuery{}, 0, 20)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if totalA != 1 || len(assetsA) != 1 || assetsA[0].UserId != 1 {
		t.Fatalf("unexpected assets for user A: %+v", assetsA)
	}
}
