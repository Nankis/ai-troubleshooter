package lark

import (
	"context"
	"fmt"
	"strings"

	"github.com/ginseng/ai-troubleshooter/internal/vision"
)

type DownloadedImage struct {
	ImageKey  string
	MediaType string
	Data      []byte
}

type ImageDownloader interface {
	DownloadImage(ctx context.Context, messageID string, imageKey string) (DownloadedImage, error)
}

type ImageOptions struct {
	MaxImages     int
	MaxImageBytes int
}

func (h *Handler) SetImageProcessor(downloader ImageDownloader, client vision.Client, options ImageOptions) {
	h.imageDownloader = downloader
	h.vision = client
	h.imageOptions = options
}

func (h *Handler) enrichOCRWithImages(ctx context.Context, event Event) string {
	parts := []string{}
	if strings.TrimSpace(event.OCRText) != "" {
		parts = append(parts, strings.TrimSpace(event.OCRText))
	}
	imageKeys := boundedImageKeys(event.ImageKeys, h.imageOptions.MaxImages)
	if len(imageKeys) == 0 {
		return strings.Join(parts, "\n")
	}
	if h.imageDownloader == nil {
		parts = append(parts, "图片未下载：未配置 Lark 图片下载器；image_keys="+strings.Join(imageKeys, ","))
		return strings.Join(parts, "\n")
	}
	images := []vision.ImageInput{}
	for _, imageKey := range imageKeys {
		downloaded, err := h.imageDownloader.DownloadImage(ctx, event.MessageID, imageKey)
		if err != nil {
			parts = append(parts, fmt.Sprintf("图片下载失败 image_key=%s error=%s", imageKey, err.Error()))
			continue
		}
		maxBytes := h.imageOptions.MaxImageBytes
		if maxBytes <= 0 {
			maxBytes = 10 * 1024 * 1024
		}
		if len(downloaded.Data) > maxBytes {
			parts = append(parts, fmt.Sprintf("图片跳过 image_key=%s reason=超过大小限制 %d bytes", imageKey, maxBytes))
			continue
		}
		images = append(images, vision.ImageInput{
			ImageKey:  downloaded.ImageKey,
			MediaType: downloaded.MediaType,
			Data:      downloaded.Data,
		})
	}
	if len(images) == 0 {
		return strings.Join(parts, "\n")
	}
	if h.vision == nil {
		parts = append(parts, fmt.Sprintf("图片已下载 %d 张，但未配置视觉识别模型。", len(images)))
		return strings.Join(parts, "\n")
	}
	analysis, err := h.vision.AnalyzeImages(ctx, event.Text, images)
	if err != nil {
		parts = append(parts, "图片识别失败："+err.Error())
		return strings.Join(parts, "\n")
	}
	parts = append(parts, fmt.Sprintf("图片识别结果 provider=%s model=%s:\n%s",
		fallback(analysis.ModelProvider, "unknown"),
		fallback(analysis.ModelName, "unknown"),
		strings.TrimSpace(analysis.OCRText)))
	return strings.Join(parts, "\n")
}

func boundedImageKeys(values []string, max int) []string {
	if max <= 0 {
		max = 3
	}
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if len(out) >= max {
			break
		}
	}
	return out
}
