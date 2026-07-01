package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var (
	ErrSeedanceForbidden          = errors.New("forbidden")
	ErrSeedanceChannelNotFound    = errors.New("no enabled JDSeedance channel configured")
	ErrSeedanceUpstreamSyncFailed = errors.New("failed to sync asset from upstream")
)

type SeedanceAssetUpstream interface {
	CreateAssetGroup(name, description, groupType string) (string, error)
	UpdateAssetGroup(id, name, description string) error
	CreateAsset(groupId, assetURL, assetType, name string) (string, error)
	GetAsset(id string) (*SeedanceGetAssetResult, error)
	UpdateAsset(id, name string) error
	DeleteAsset(id string) error
}

type SeedanceAssetService struct {
	NewClient func(baseURL, apiKey string) SeedanceAssetUpstream
}

func NewSeedanceAssetService() *SeedanceAssetService {
	return &SeedanceAssetService{
		NewClient: func(baseURL, apiKey string) SeedanceAssetUpstream {
			return NewSeedanceAssetClient(baseURL, apiKey)
		},
	}
}

func seedanceStorageRoot() string {
	if path := strings.TrimSpace(os.Getenv("SEEDANCE_ASSET_STORAGE_PATH")); path != "" {
		return path
	}
	if common.SQLitePath != "" {
		return filepath.Join(filepath.Dir(common.SQLitePath), "seedance")
	}
	return filepath.Join(".", "data", "seedance")
}

func seedanceFileTokenTTL() int {
	ttl := common.GetEnvOrDefault("SEEDANCE_FILE_TOKEN_TTL", 3600)
	if ttl <= 0 {
		return 3600
	}
	return ttl
}

func seedanceMaxUploadBytes() int64 {
	mb := common.GetEnvOrDefault("SEEDANCE_MAX_UPLOAD_MB", 100)
	if mb <= 0 {
		mb = 100
	}
	return int64(mb) * 1024 * 1024
}

func (s *SeedanceAssetService) resolveClient(userId int) (SeedanceAssetUpstream, error) {
	user, err := model.GetUserById(userId, false)
	if err != nil {
		return nil, err
	}
	channel, err := model.GetJDSeedanceChannel(user.Group)
	if err != nil {
		return nil, ErrSeedanceChannelNotFound
	}
	baseURL := "https://agentrs.jd.com"
	if channel.BaseURL != nil && strings.TrimSpace(*channel.BaseURL) != "" {
		baseURL = strings.TrimSpace(*channel.BaseURL)
	}
	return s.NewClient(baseURL, channel.Key), nil
}

func (s *SeedanceAssetService) EnsureDefaultGroup(userId int) (*model.SeedanceAssetGroup, error) {
	count, err := model.CountSeedanceAssetGroups(userId)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		groups, _, err := model.ListSeedanceAssetGroups(userId, 0, 1)
		if err != nil {
			return nil, err
		}
		for _, group := range groups {
			if group.IsDefault {
				return &group, nil
			}
		}
		if len(groups) > 0 {
			return &groups[0], nil
		}
	}
	return s.CreateGroup(userId, fmt.Sprintf("user-%d-default", userId), "Default Seedance asset group", "AIGC", true)
}

func (s *SeedanceAssetService) CreateGroup(userId int, name, description, groupType string, isDefault bool) (*model.SeedanceAssetGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("group name is required")
	}
	if groupType == "" {
		groupType = "AIGC"
	}
	client, err := s.resolveClient(userId)
	if err != nil {
		return nil, err
	}
	upstreamID, err := client.CreateAssetGroup(name, description, groupType)
	if err != nil {
		return nil, err
	}
	group := &model.SeedanceAssetGroup{
		UserId:      userId,
		UpstreamId:  upstreamID,
		Name:        name,
		Description: description,
		GroupType:   groupType,
		IsDefault:   isDefault,
	}
	if err := model.CreateSeedanceAssetGroup(group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *SeedanceAssetService) ListGroups(userId, offset, limit int) ([]model.SeedanceAssetGroup, int64, error) {
	count, err := model.CountSeedanceAssetGroups(userId)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		if _, err := s.EnsureDefaultGroup(userId); err != nil {
			return nil, 0, err
		}
	}
	return model.ListSeedanceAssetGroups(userId, offset, limit)
}

func (s *SeedanceAssetService) UpdateGroup(userId int, groupID int64, name, description string) (*model.SeedanceAssetGroup, error) {
	group, err := model.GetSeedanceAssetGroupByID(userId, groupID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("group name is required")
	}
	client, err := s.resolveClient(userId)
	if err != nil {
		return nil, err
	}
	if err := client.UpdateAssetGroup(group.UpstreamId, name, description); err != nil {
		return nil, err
	}
	if err := model.UpdateSeedanceAssetGroup(userId, groupID, name, description); err != nil {
		return nil, err
	}
	group.Name = name
	group.Description = description
	return group, nil
}

func (s *SeedanceAssetService) ListAssets(userId int, query model.SeedanceAssetQuery, offset, limit int) ([]model.SeedanceAsset, int64, error) {
	return model.ListSeedanceAssets(userId, query, offset, limit)
}

func (s *SeedanceAssetService) GetAsset(userId int, assetID int64, sync bool) (*model.SeedanceAsset, error) {
	asset, err := model.GetSeedanceAssetByID(userId, assetID)
	if err != nil {
		return nil, err
	}
	if !sync {
		return asset, nil
	}
	client, err := s.resolveClient(userId)
	if err != nil {
		return nil, err
	}
	upstream, err := client.GetAsset(asset.UpstreamId)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSeedanceUpstreamSyncFailed, err)
	}
	_ = model.UpdateSeedanceAssetSyncFields(userId, assetID, upstream.URL, upstream.Status)
	asset.PublicUrl = upstream.URL
	asset.Status = upstream.Status
	if upstream.Name != "" {
		asset.Name = upstream.Name
	}
	return asset, nil
}

type CreateSeedanceAssetInput struct {
	GroupID   int64
	URL       string
	UploadID  int64
	AssetType string
	Name      string
}

func (s *SeedanceAssetService) CreateAsset(userId int, input CreateSeedanceAssetInput, publicBaseURL string) (*model.SeedanceAsset, error) {
	group, err := model.GetSeedanceAssetGroupByID(userId, input.GroupID)
	if err != nil {
		return nil, err
	}
	assetURL := strings.TrimSpace(input.URL)
	var upload *model.SeedanceUpload
	if input.UploadID > 0 {
		var uploadErr error
		upload, uploadErr = model.GetSeedanceUploadByID(userId, input.UploadID)
		if uploadErr != nil {
			return nil, uploadErr
		}
		if upload.ExpiresAt < time.Now().Unix() {
			return nil, errors.New("upload expired")
		}
		assetURL, err = buildSeedancePublicURL(publicBaseURL, upload.SignedToken)
		if err != nil {
			return nil, err
		}
	}
	if assetURL == "" {
		return nil, errors.New("asset url or upload_id is required")
	}
	if input.UploadID <= 0 {
		if err := validateSeedanceAssetURLReachable(assetURL); err != nil {
			return nil, err
		}
	}
	assetType := strings.TrimSpace(input.AssetType)
	if assetType == "" && upload != nil {
		inferred, inferErr := InferSeedanceAssetTypeFromMIME(upload.MimeType)
		if inferErr != nil {
			return nil, inferErr
		}
		assetType = inferred
	}
	if assetType == "" {
		return nil, errors.New("asset_type is required")
	}
	normalizedType, err := NormalizeSeedanceAssetType(assetType)
	if err != nil {
		return nil, err
	}
	assetType = normalizedType
	client, err := s.resolveClient(userId)
	if err != nil {
		return nil, err
	}
	upstreamID, err := client.CreateAsset(group.UpstreamId, assetURL, assetType, strings.TrimSpace(input.Name))
	if err != nil {
		return nil, err
	}
	asset := &model.SeedanceAsset{
		UserId:     userId,
		GroupId:    group.Id,
		UpstreamId: upstreamID,
		Name:       strings.TrimSpace(input.Name),
		AssetType:  assetType,
		SourceUrl:  assetURL,
		Status:     "pending",
	}
	if err := model.CreateSeedanceAsset(asset); err != nil {
		return nil, err
	}
	if upstream, getErr := client.GetAsset(upstreamID); getErr == nil {
		_ = model.UpdateSeedanceAssetSyncFields(userId, asset.Id, upstream.URL, upstream.Status)
		asset.PublicUrl = upstream.URL
		asset.Status = upstream.Status
	}
	return asset, nil
}

func (s *SeedanceAssetService) UpdateAsset(userId int, assetID int64, name string) (*model.SeedanceAsset, error) {
	asset, err := model.GetSeedanceAssetByID(userId, assetID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("asset name is required")
	}
	client, err := s.resolveClient(userId)
	if err != nil {
		return nil, err
	}
	if err := client.UpdateAsset(asset.UpstreamId, name); err != nil {
		return nil, err
	}
	if err := model.UpdateSeedanceAsset(userId, assetID, name); err != nil {
		return nil, err
	}
	asset.Name = name
	return asset, nil
}

func (s *SeedanceAssetService) DeleteAsset(userId int, assetID int64) error {
	asset, err := model.GetSeedanceAssetByID(userId, assetID)
	if err != nil {
		return err
	}
	client, err := s.resolveClient(userId)
	if err != nil {
		return err
	}
	if err := client.DeleteAsset(asset.UpstreamId); err != nil {
		return err
	}
	return model.DeleteSeedanceAsset(userId, assetID)
}

func buildSeedancePublicURL(publicBaseURL, token string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if base == "" {
		return "", errors.New("public base url is not configured")
	}
	return fmt.Sprintf("%s/api/seedance/files/public/%s", base, token), nil
}

func (s *SeedanceAssetService) UploadFile(userId int, header *multipart.FileHeader, publicBaseURL string) (*model.SeedanceUpload, string, error) {
	if header == nil {
		return nil, "", errors.New("file is required")
	}
	if header.Size > seedanceMaxUploadBytes() {
		return nil, "", fmt.Errorf("file exceeds max upload size of %d MB", common.GetEnvOrDefault("SEEDANCE_MAX_UPLOAD_MB", 100))
	}
	file, err := header.Open()
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", err
	}
	token := hex.EncodeToString(tokenBytes)

	userDir := filepath.Join(seedanceStorageRoot(), fmt.Sprintf("%d", userId))
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return nil, "", err
	}
	safeName := filepath.Base(header.Filename)
	storagePath := filepath.Join(userDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), safeName))
	out, err := os.Create(storagePath)
	if err != nil {
		return nil, "", err
	}
	written, err := io.Copy(out, file)
	out.Close()
	if err != nil {
		_ = os.Remove(storagePath)
		return nil, "", err
	}

	mimeType, err := detectSeedanceFileMIME(storagePath, header.Header.Get("Content-Type"))
	if err != nil {
		_ = os.Remove(storagePath)
		return nil, "", err
	}

	expiresAt := time.Now().Unix() + int64(seedanceFileTokenTTL())
	upload := &model.SeedanceUpload{
		UserId:      userId,
		FileName:    safeName,
		MimeType:    mimeType,
		Size:        written,
		StoragePath: storagePath,
		SignedToken: token,
		ExpiresAt:   expiresAt,
	}
	if err := model.CreateSeedanceUpload(upload); err != nil {
		_ = os.Remove(storagePath)
		return nil, "", err
	}
	publicURL, err := buildSeedancePublicURL(publicBaseURL, token)
	if err != nil {
		return nil, "", err
	}
	return upload, publicURL, nil
}

func (s *SeedanceAssetService) ServePublicFile(token string) (string, string, error) {
	upload, err := model.GetSeedanceUploadByToken(token)
	if err != nil {
		return "", "", err
	}
	if upload.ExpiresAt < time.Now().Unix() {
		return "", "", model.ErrSeedanceUploadNotFound
	}
	if _, err := os.Stat(upload.StoragePath); err != nil {
		return "", "", model.ErrSeedanceUploadNotFound
	}
	mimeType, err := detectSeedanceFileMIME(upload.StoragePath, upload.MimeType)
	if err != nil {
		mimeType = normalizeSeedanceMIME(upload.MimeType)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}
	return upload.StoragePath, mimeType, nil
}

func (s *SeedanceAssetService) ResolveAssetURLForRelay(userId int, localAssetID int64) (string, error) {
	asset, err := model.GetSeedanceAssetByID(userId, localAssetID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(asset.PublicUrl) != "" {
		return asset.PublicUrl, nil
	}
	if strings.TrimSpace(asset.SourceUrl) != "" {
		return asset.SourceUrl, nil
	}
	client, err := s.resolveClient(userId)
	if err != nil {
		return "", err
	}
	upstream, err := client.GetAsset(asset.UpstreamId)
	if err != nil {
		return "", err
	}
	_ = model.UpdateSeedanceAssetSyncFields(userId, asset.Id, upstream.URL, upstream.Status)
	return upstream.URL, nil
}

func SeedancePublicBaseURLFromRequest(scheme, host string) string {
	if configured := strings.TrimSpace(os.Getenv("SEEDANCE_PUBLIC_BASE_URL")); configured != "" {
		return strings.TrimRight(configured, "/")
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}
