package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

var seedanceAssetService = service.NewSeedanceAssetService()

func seedanceAssetError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrSeedanceAssetNotFound),
		errors.Is(err, model.ErrSeedanceAssetGroupNotFound),
		errors.Is(err, model.ErrSeedanceUploadNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, service.ErrSeedanceForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, service.ErrSeedanceChannelNotFound):
		common.ApiErrorMsg(c, "No enabled JDSeedance channel configured for your group")
	case errors.Is(err, service.ErrSeedanceUnsupportedMIME):
		common.ApiErrorMsg(c, "Unsupported upload file type")
	case errors.Is(err, service.ErrSeedanceURLUnreachable):
		common.ApiErrorMsg(c, "Asset URL is not reachable")
	case errors.Is(err, service.ErrSeedanceUpstreamSyncFailed):
		common.ApiErrorMsg(c, "Failed to sync asset status from upstream")
	default:
		common.ApiError(c, err)
	}
}

func ListSeedanceAssetGroups(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	groups, total, err := seedanceAssetService.ListGroups(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		seedanceAssetError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(groups)
	common.ApiSuccess(c, pageInfo)
}

type createSeedanceGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	GroupType   string `json:"group_type"`
}

func CreateSeedanceAssetGroup(c *gin.Context) {
	userId := c.GetInt("id")
	var req createSeedanceGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	group, err := seedanceAssetService.CreateGroup(userId, req.Name, req.Description, req.GroupType, false)
	if err != nil {
		seedanceAssetError(c, err)
		return
	}
	common.ApiSuccess(c, group)
}

type updateSeedanceGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func UpdateSeedanceAssetGroup(c *gin.Context) {
	userId := c.GetInt("id")
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID <= 0 {
		common.ApiErrorMsg(c, "invalid group id")
		return
	}
	var req updateSeedanceGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	group, err := seedanceAssetService.UpdateGroup(userId, groupID, req.Name, req.Description)
	if err != nil {
		seedanceAssetError(c, err)
		return
	}
	common.ApiSuccess(c, group)
}

func ListSeedanceAssets(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	groupID, _ := strconv.ParseInt(c.Query("group_id"), 10, 64)
	query := model.SeedanceAssetQuery{
		GroupId: groupID,
		Keyword: c.Query("keyword"),
	}
	assets, total, err := seedanceAssetService.ListAssets(userId, query, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		seedanceAssetError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(assets)
	common.ApiSuccess(c, pageInfo)
}

type createSeedanceAssetRequest struct {
	GroupID   int64  `json:"group_id"`
	URL       string `json:"url"`
	UploadID  int64  `json:"upload_id"`
	AssetType string `json:"asset_type"`
	Name      string `json:"name"`
}

func CreateSeedanceAsset(c *gin.Context) {
	userId := c.GetInt("id")
	var req createSeedanceAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.GroupID <= 0 {
		common.ApiErrorMsg(c, "group_id is required")
		return
	}
	publicBaseURL := service.SeedancePublicBaseURLFromRequest(requestScheme(c), c.Request.Host)
	asset, err := seedanceAssetService.CreateAsset(userId, service.CreateSeedanceAssetInput{
		GroupID:   req.GroupID,
		URL:       req.URL,
		UploadID:  req.UploadID,
		AssetType: req.AssetType,
		Name:      req.Name,
	}, publicBaseURL)
	if err != nil {
		seedanceAssetError(c, err)
		return
	}
	common.ApiSuccess(c, asset)
}

func GetSeedanceAsset(c *gin.Context) {
	userId := c.GetInt("id")
	assetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || assetID <= 0 {
		common.ApiErrorMsg(c, "invalid asset id")
		return
	}
	sync := strings.EqualFold(c.Query("sync"), "true") || c.Query("sync") == "1"
	asset, err := seedanceAssetService.GetAsset(userId, assetID, sync)
	if err != nil {
		seedanceAssetError(c, err)
		return
	}
	common.ApiSuccess(c, asset)
}

type updateSeedanceAssetRequest struct {
	Name string `json:"name"`
}

func UpdateSeedanceAsset(c *gin.Context) {
	userId := c.GetInt("id")
	assetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || assetID <= 0 {
		common.ApiErrorMsg(c, "invalid asset id")
		return
	}
	var req updateSeedanceAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	asset, err := seedanceAssetService.UpdateAsset(userId, assetID, req.Name)
	if err != nil {
		seedanceAssetError(c, err)
		return
	}
	common.ApiSuccess(c, asset)
}

func DeleteSeedanceAsset(c *gin.Context) {
	userId := c.GetInt("id")
	assetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || assetID <= 0 {
		common.ApiErrorMsg(c, "invalid asset id")
		return
	}
	if err := seedanceAssetService.DeleteAsset(userId, assetID); err != nil {
		seedanceAssetError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func UploadSeedanceAssetFile(c *gin.Context) {
	userId := c.GetInt("id")
	file, err := c.FormFile("file")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	publicBaseURL := service.SeedancePublicBaseURLFromRequest(requestScheme(c), c.Request.Host)
	upload, publicURL, err := seedanceAssetService.UploadFile(userId, file, publicBaseURL)
	if err != nil {
		seedanceAssetError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"id":          upload.Id,
		"file_name":   upload.FileName,
		"mime_type":   upload.MimeType,
		"size":        upload.Size,
		"public_url":  publicURL,
		"expires_at":  upload.ExpiresAt,
		"signed_token": upload.SignedToken,
	})
}

func ServeSeedancePublicFile(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	path, mimeType, err := seedanceAssetService.ServePublicFile(token)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if mimeType != "" {
		c.Header("Content-Type", mimeType)
	}
	c.File(path)
}

func requestScheme(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return strings.TrimSpace(strings.Split(proto, ",")[0])
	}
	return "http"
}
