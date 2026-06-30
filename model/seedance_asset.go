package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

var (
	ErrSeedanceAssetNotFound      = errors.New("seedance asset not found")
	ErrSeedanceAssetGroupNotFound = errors.New("seedance asset group not found")
	ErrSeedanceUploadNotFound     = errors.New("seedance upload not found")
)

type SeedanceAssetGroup struct {
	Id          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index;not null"`
	UpstreamId  string `json:"upstream_id" gorm:"type:varchar(128);uniqueIndex;not null"`
	Name        string `json:"name" gorm:"type:varchar(64);not null"`
	Description string `json:"description" gorm:"type:varchar(300)"`
	GroupType   string `json:"group_type" gorm:"type:varchar(32);not null;default:AIGC"`
	IsDefault   bool   `json:"is_default" gorm:"not null;default:false"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;not null"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;not null"`
}

type SeedanceAsset struct {
	Id         int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId     int            `json:"user_id" gorm:"index;not null"`
	GroupId    int64          `json:"group_id" gorm:"index;not null"`
	UpstreamId string         `json:"upstream_id" gorm:"type:varchar(128);uniqueIndex;not null"`
	Name       string         `json:"name" gorm:"type:varchar(64)"`
	AssetType  string         `json:"asset_type" gorm:"type:varchar(64);not null"`
	SourceUrl  string         `json:"source_url" gorm:"type:text"`
	PublicUrl  string         `json:"public_url" gorm:"type:text"`
	Status     string         `json:"status" gorm:"type:varchar(32)"`
	CreatedAt  int64          `json:"created_at" gorm:"bigint;not null"`
	UpdatedAt  int64          `json:"updated_at" gorm:"bigint;not null"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

type SeedanceUpload struct {
	Id          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index;not null"`
	FileName    string `json:"file_name" gorm:"type:varchar(255);not null"`
	MimeType    string `json:"mime_type" gorm:"type:varchar(128)"`
	Size        int64  `json:"size" gorm:"not null"`
	StoragePath string `json:"-" gorm:"type:varchar(512);not null"`
	SignedToken string `json:"signed_token" gorm:"type:varchar(128);uniqueIndex;not null"`
	ExpiresAt   int64  `json:"expires_at" gorm:"bigint;index;not null"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;not null"`
}

type SeedanceAssetQuery struct {
	GroupId int64
	Keyword string
}

func nowUnix() int64 {
	return time.Now().Unix()
}

func CreateSeedanceAssetGroup(group *SeedanceAssetGroup) error {
	if group == nil {
		return errors.New("group is nil")
	}
	ts := nowUnix()
	if group.CreatedAt == 0 {
		group.CreatedAt = ts
	}
	group.UpdatedAt = ts
	return DB.Create(group).Error
}

func GetSeedanceAssetGroupByID(userId int, id int64) (*SeedanceAssetGroup, error) {
	var group SeedanceAssetGroup
	err := DB.Where("user_id = ? AND id = ?", userId, id).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeedanceAssetGroupNotFound
		}
		return nil, err
	}
	return &group, nil
}

func GetSeedanceAssetGroupByUpstreamID(userId int, upstreamId string) (*SeedanceAssetGroup, error) {
	var group SeedanceAssetGroup
	err := DB.Where("user_id = ? AND upstream_id = ?", userId, upstreamId).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeedanceAssetGroupNotFound
		}
		return nil, err
	}
	return &group, nil
}

func ListSeedanceAssetGroups(userId int, offset, limit int) ([]SeedanceAssetGroup, int64, error) {
	var groups []SeedanceAssetGroup
	var total int64
	query := DB.Model(&SeedanceAssetGroup{}).Where("user_id = ?", userId)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("is_default DESC, id DESC").Offset(offset).Limit(limit).Find(&groups).Error
	return groups, total, err
}

func CountSeedanceAssetGroups(userId int) (int64, error) {
	var total int64
	err := DB.Model(&SeedanceAssetGroup{}).Where("user_id = ?", userId).Count(&total).Error
	return total, err
}

func UpdateSeedanceAssetGroup(userId int, id int64, name, description string) error {
	result := DB.Model(&SeedanceAssetGroup{}).
		Where("user_id = ? AND id = ?", userId, id).
		Updates(map[string]any{
			"name":        name,
			"description": description,
			"updated_at":  nowUnix(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSeedanceAssetGroupNotFound
	}
	return nil
}

func CreateSeedanceAsset(asset *SeedanceAsset) error {
	if asset == nil {
		return errors.New("asset is nil")
	}
	ts := nowUnix()
	if asset.CreatedAt == 0 {
		asset.CreatedAt = ts
	}
	asset.UpdatedAt = ts
	return DB.Create(asset).Error
}

func GetSeedanceAssetByID(userId int, id int64) (*SeedanceAsset, error) {
	var asset SeedanceAsset
	err := DB.Where("user_id = ? AND id = ?", userId, id).First(&asset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeedanceAssetNotFound
		}
		return nil, err
	}
	return &asset, nil
}

func GetSeedanceAssetByUpstreamID(userId int, upstreamId string) (*SeedanceAsset, error) {
	var asset SeedanceAsset
	err := DB.Where("user_id = ? AND upstream_id = ?", userId, upstreamId).First(&asset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeedanceAssetNotFound
		}
		return nil, err
	}
	return &asset, nil
}

func ListSeedanceAssets(userId int, query SeedanceAssetQuery, offset, limit int) ([]SeedanceAsset, int64, error) {
	var assets []SeedanceAsset
	var total int64
	db := DB.Model(&SeedanceAsset{}).Where("user_id = ?", userId)
	if query.GroupId > 0 {
		db = db.Where("group_id = ?", query.GroupId)
	}
	keyword := strings.TrimSpace(query.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ?", like)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&assets).Error
	return assets, total, err
}

func UpdateSeedanceAsset(userId int, id int64, name string) error {
	result := DB.Model(&SeedanceAsset{}).
		Where("user_id = ? AND id = ?", userId, id).
		Updates(map[string]any{
			"name":       name,
			"updated_at": nowUnix(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSeedanceAssetNotFound
	}
	return nil
}

func UpdateSeedanceAssetSyncFields(userId int, id int64, publicUrl, status string) error {
	result := DB.Model(&SeedanceAsset{}).
		Where("user_id = ? AND id = ?", userId, id).
		Updates(map[string]any{
			"public_url": publicUrl,
			"status":     status,
			"updated_at": nowUnix(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSeedanceAssetNotFound
	}
	return nil
}

func DeleteSeedanceAsset(userId int, id int64) error {
	result := DB.Where("user_id = ? AND id = ?", userId, id).Delete(&SeedanceAsset{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSeedanceAssetNotFound
	}
	return nil
}

func CreateSeedanceUpload(upload *SeedanceUpload) error {
	if upload == nil {
		return errors.New("upload is nil")
	}
	if upload.CreatedAt == 0 {
		upload.CreatedAt = nowUnix()
	}
	return DB.Create(upload).Error
}

func GetSeedanceUploadByID(userId int, id int64) (*SeedanceUpload, error) {
	var upload SeedanceUpload
	err := DB.Where("user_id = ? AND id = ?", userId, id).First(&upload).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeedanceUploadNotFound
		}
		return nil, err
	}
	return &upload, nil
}

func GetSeedanceUploadByToken(token string) (*SeedanceUpload, error) {
	var upload SeedanceUpload
	err := DB.Where("signed_token = ?", token).First(&upload).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeedanceUploadNotFound
		}
		return nil, err
	}
	return &upload, nil
}

func ListExpiredSeedanceUploads(beforeUnix int64, limit int) ([]SeedanceUpload, error) {
	if limit <= 0 {
		limit = 100
	}
	var uploads []SeedanceUpload
	err := DB.Where("expires_at < ?", beforeUnix).
		Order("expires_at ASC").
		Limit(limit).
		Find(&uploads).Error
	return uploads, err
}

func DeleteSeedanceUploadByID(id int64) error {
	return DB.Where("id = ?", id).Delete(&SeedanceUpload{}).Error
}

func ListSeedanceAssetsPendingSync(limit int) ([]SeedanceAsset, error) {
	if limit <= 0 {
		limit = 50
	}
	var assets []SeedanceAsset
	err := DB.Where("status = ? OR public_url = '' OR public_url IS NULL", "pending").
		Order("updated_at ASC").
		Limit(limit).
		Find(&assets).Error
	return assets, err
}

func GetJDSeedanceChannel(userGroup string) (*Channel, error) {
	groups := []string{userGroup, "default"}
	seen := make(map[string]bool)
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" || seen[group] {
			continue
		}
		seen[group] = true
		var channel Channel
		err := DB.Where("type = ? AND `group` = ? AND status = ?",
			constant.ChannelTypeJDSeedance, group, common.ChannelStatusEnabled).
			Order("priority DESC, id ASC").
			First(&channel).Error
		if err == nil {
			return &channel, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("no enabled JDSeedance channel found for group %q", userGroup)
}
