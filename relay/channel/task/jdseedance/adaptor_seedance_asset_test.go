package jdseedance

import (
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestResolveSeedanceMediaURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	db, err := gorm.Open(sqlite.Open("file:jdseedance-resolve?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	model.DB = db
	if err := db.AutoMigrate(&model.SeedanceAsset{}); err != nil {
		t.Fatal(err)
	}
	asset := &model.SeedanceAsset{
		UserId:     7,
		GroupId:    1,
		UpstreamId: "asset-test-1",
		AssetType:  "image",
		SourceUrl:  "https://cdn.example.com/a.jpg",
		PublicUrl:  "https://cdn.example.com/a.jpg",
	}
	if err := model.CreateSeedanceAsset(asset); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", 7)

	resolved, err := resolveSeedanceMediaURL(c, "seedance_asset://"+strconv.FormatInt(asset.Id, 10))
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved != "https://cdn.example.com/a.jpg" {
		t.Fatalf("resolved = %q", resolved)
	}

	c.Set("id", 8)
	if _, err := resolveSeedanceMediaURL(c, "seedance_asset://"+strconv.FormatInt(asset.Id, 10)); err == nil {
		t.Fatal("other user should not resolve asset")
	}
}
