package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/webp"
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
var ErrSeedanceImageDimensions = errors.New("seedance image dimensions invalid")

const (
	seedanceImageMinPx     = 300
	seedanceImageMaxPx     = 6000
	seedanceImageMinAspect = 0.4
	seedanceImageMaxAspect = 2.5
)

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

func decodeSeedanceImageDimensions(reader io.Reader) (width, height int, err error) {
	data, err := io.ReadAll(io.LimitReader(reader, 4<<20))
	if err != nil {
		return 0, 0, err
	}
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("empty image data")
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err == nil {
		return config.Width, config.Height, nil
	}
	config, err = webp.DecodeConfig(bytes.NewReader(data))
	if err == nil {
		return config.Width, config.Height, nil
	}
	return 0, 0, fmt.Errorf("failed to decode image dimensions: %w", err)
}

func validateSeedanceImageDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("%w: unable to read image size; ensure the public URL serves the original image file", ErrSeedanceImageDimensions)
	}
	if width < seedanceImageMinPx || height < seedanceImageMinPx {
		return fmt.Errorf("%w: width and height must be at least %dpx (got %dx%d)", ErrSeedanceImageDimensions, seedanceImageMinPx, width, height)
	}
	if width > seedanceImageMaxPx || height > seedanceImageMaxPx {
		return fmt.Errorf("%w: width and height must be at most %dpx (got %dx%d)", ErrSeedanceImageDimensions, seedanceImageMaxPx, width, height)
	}
	aspect := float64(width) / float64(height)
	if aspect < seedanceImageMinAspect || aspect > seedanceImageMaxAspect {
		return fmt.Errorf("%w: aspect ratio must be between %.1f and %.1f (got %.2f for %dx%d)", ErrSeedanceImageDimensions, seedanceImageMinAspect, seedanceImageMaxAspect, aspect, width, height)
	}
	return nil
}

func validateSeedanceImageFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	width, height, err := decodeSeedanceImageDimensions(file)
	if err != nil {
		return err
	}
	return validateSeedanceImageDimensions(width, height)
}

func validateSeedanceImageURL(assetURL string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, assetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-2097151")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSeedanceURLUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: status %d", ErrSeedanceURLUnreachable, resp.StatusCode)
	}
	width, height, err := decodeSeedanceImageDimensions(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSeedanceImageDimensions, err)
	}
	return validateSeedanceImageDimensions(width, height)
}

func FormatSeedanceUpstreamError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "WidthTooSmall"), strings.Contains(msg, "WidthTooLarge"),
		strings.Contains(msg, "HeightTooSmall"), strings.Contains(msg, "HeightTooLarge"):
		return fmt.Sprintf("Image size does not meet Seedance requirements: each side must be %d-%d px with aspect ratio %.1f-%.1f", seedanceImageMinPx, seedanceImageMaxPx, seedanceImageMinAspect, seedanceImageMaxAspect)
	case strings.Contains(msg, "InvalidParameter.AssetType"):
		return "Invalid asset type. Use Image, Video, or Audio."
	default:
		if idx := strings.Index(msg, "seedance upstream error: "); idx >= 0 {
			return strings.TrimSpace(msg[idx+len("seedance upstream error: "):])
		}
		return msg
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
