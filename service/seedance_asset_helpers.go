package service

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var seedanceAllowedMIMETypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"video/mp4":       true,
	"video/quicktime": true,
	"video/webm":      true,
	"audio/mpeg":      true,
	"audio/mp4":       true,
	"audio/wav":       true,
	"audio/x-wav":     true,
	"audio/ogg":       true,
}

var ErrSeedanceUnsupportedMIME = errors.New("unsupported upload mime type")
var ErrSeedanceURLUnreachable  = errors.New("asset url is not reachable")
var ErrSeedanceInvalidAssetType = errors.New("invalid asset type")

var seedanceAssetTypeAliases = map[string]string{
	"image": "Image",
	"video": "Video",
	"audio": "Audio",
}

func NormalizeSeedanceAssetType(assetType string) (string, error) {
	assetType = strings.TrimSpace(assetType)
	if assetType == "" {
		return "", ErrSeedanceInvalidAssetType
	}
	if canonical, ok := seedanceAssetTypeAliases[strings.ToLower(assetType)]; ok {
		return canonical, nil
	}
	switch assetType {
	case "Image", "Video", "Audio":
		return assetType, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrSeedanceInvalidAssetType, assetType)
	}
}

func InferSeedanceAssetTypeFromMIME(mimeType string) (string, error) {
	normalized := normalizeSeedanceMIME(mimeType)
	switch {
	case strings.HasPrefix(normalized, "image/"):
		return "Image", nil
	case strings.HasPrefix(normalized, "video/"):
		return "Video", nil
	case strings.HasPrefix(normalized, "audio/"):
		return "Audio", nil
	default:
		return "", fmt.Errorf("%w: unsupported mime %s", ErrSeedanceInvalidAssetType, normalized)
	}
}

func normalizeSeedanceMIME(mimeType string) string {
	mimeType = strings.TrimSpace(strings.ToLower(mimeType))
	if mimeType == "" {
		return ""
	}
	if idx := strings.Index(mimeType, ";"); idx >= 0 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	return mimeType
}

func validateSeedanceUploadMIME(mimeType string) error {
	normalized := normalizeSeedanceMIME(mimeType)
	if normalized == "" {
		return ErrSeedanceUnsupportedMIME
	}
	if !seedanceAllowedMIMETypes[normalized] {
		return fmt.Errorf("%w: %s", ErrSeedanceUnsupportedMIME, normalized)
	}
	return nil
}

func detectSeedanceFileMIME(path string, headerMIME string) (string, error) {
	normalizedHeader := normalizeSeedanceMIME(headerMIME)
	if normalizedHeader != "" {
		if err := validateSeedanceUploadMIME(normalizedHeader); err != nil {
			return "", err
		}
		return normalizedHeader, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return "", err
	}
	detected := normalizeSeedanceMIME(http.DetectContentType(buf[:n]))
	if detected == "application/octet-stream" {
		if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
			if byExt := normalizeSeedanceMIME(mime.TypeByExtension(ext)); byExt != "" {
				detected = byExt
			}
		}
	}
	if err := validateSeedanceUploadMIME(detected); err != nil {
		return "", err
	}
	return detected, nil
}

func validateSeedanceAssetURLReachable(assetURL string) error {
	assetURL = strings.TrimSpace(assetURL)
	if assetURL == "" {
		return errors.New("asset url is required")
	}
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest(http.MethodHead, assetURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		getReq, getErr := http.NewRequest(http.MethodGet, assetURL, nil)
		if getErr != nil {
			return fmt.Errorf("%w: %v", ErrSeedanceURLUnreachable, err)
		}
		getReq.Header.Set("Range", "bytes=0-0")
		resp, err = client.Do(getReq)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSeedanceURLUnreachable, err)
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: status %d", ErrSeedanceURLUnreachable, resp.StatusCode)
	}
	return nil
}
