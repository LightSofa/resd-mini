package core

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"resd-mini/core/shared"
	"strings"
	"sync"
)

type WxFileDecodeResult struct {
	SavePath string
	Message  string
}

type Resource struct {
	mediaMark    sync.Map
	tasks        sync.Map
	mediaByID    sync.Map
	mediaSignIdx sync.Map
	resType      map[string]bool
	resTypeMux   sync.RWMutex
}

func initResource() *Resource {
	if resourceOnce == nil {
		resourceOnce = &Resource{}
		resourceOnce.resType = resourceOnce.buildResType(globalConfig.MimeMap)
	}
	return resourceOnce
}

func (r *Resource) buildResType(mime map[string]MimeInfo) map[string]bool {
	t := map[string]bool{
		"all": true,
	}

	for _, item := range mime {
		if _, ok := t[item.Type]; !ok {
			t[item.Type] = true
		}
	}

	return t
}

func (r *Resource) mediaIsMarked(key string) bool {
	_, loaded := r.mediaMark.Load(key)
	return loaded
}

func (r *Resource) markMedia(key string) {
	r.mediaMark.Store(key, true)
}

func (r *Resource) getResType(key string) (bool, bool) {
	r.resTypeMux.RLock()
	value, ok := r.resType[key]
	r.resTypeMux.RUnlock()
	return value, ok
}

func (r *Resource) setResType(n []string) {
	r.resTypeMux.Lock()
	for key := range r.resType {
		r.resType[key] = false
	}

	for _, value := range n {
		if _, ok := r.resType[value]; ok {
			r.resType[value] = true
		}
	}
	r.resTypeMux.Unlock()
}

func (r *Resource) clear() {
	clearSyncMap(&r.mediaMark)
	clearSyncMap(&r.mediaByID)
	clearSyncMap(&r.mediaSignIdx)
}

func (r *Resource) delete(sign string) {
	r.mediaMark.Delete(sign)
	if id, ok := r.mediaSignIdx.Load(sign); ok {
		r.mediaByID.Delete(id.(string))
		r.mediaSignIdx.Delete(sign)
	}
}

func (r *Resource) cancel(id string) error {
	if d, ok := r.tasks.Load(id); ok {
		d.(*FileDownloader).Cancel()
		r.tasks.Delete(id)
		r.updateDownloadStatus(id, shared.DownloadStatusError, "cancelled", "")
		return nil
	}
	return errors.New("task not found")
}

func (r *Resource) rememberMedia(media shared.MediaInfo) {
	if media.Id == "" {
		return
	}
	if media.Status == "" {
		media.Status = shared.DownloadStatusReady
	}
	if media.OtherData == nil {
		media.OtherData = map[string]string{}
	}
	r.mediaByID.Store(media.Id, media)
	if media.UrlSign != "" {
		r.mediaMark.Store(media.UrlSign, true)
		r.mediaSignIdx.Store(media.UrlSign, media.Id)
	}
}

func (r *Resource) updateDownloadStatus(id, status, message, savePath string) {
	if id == "" {
		return
	}
	value, ok := r.mediaByID.Load(id)
	if !ok {
		return
	}
	media := value.(shared.MediaInfo)
	if status != "" {
		media.Status = status
	}
	if savePath != "" {
		media.SavePath = savePath
	}
	if message != "" {
		if media.OtherData == nil {
			media.OtherData = map[string]string{}
		}
		media.OtherData["last_message"] = message
	}
	r.mediaByID.Store(id, media)
}

func (r *Resource) listMedia() []shared.MediaInfo {
	result := make([]shared.MediaInfo, 0, 128)
	r.mediaByID.Range(func(key, value interface{}) bool {
		result = append(result, value.(shared.MediaInfo))
		return true
	})
	return result
}

func (r *Resource) getMediaByID(id string) (shared.MediaInfo, bool) {
	value, ok := r.mediaByID.Load(id)
	if !ok {
		return shared.MediaInfo{}, false
	}
	return value.(shared.MediaInfo), true
}

func (r *Resource) download(mediaInfo shared.MediaInfo, decodeStr string) {
	if globalConfig.SaveDirectory == "" {
		return
	}
	go func(mediaInfo shared.MediaInfo) {
		rawUrl := mediaInfo.Url
		fileName := shared.Md5(rawUrl)

		if v := shared.GetFileNameFromURL(rawUrl); v != "" {
			fileName = v
		}

		if mediaInfo.Description != "" {
			fileName = regexp.MustCompile(`[^\w\p{Han}]`).ReplaceAllString(mediaInfo.Description, "")
			fileLen := globalConfig.FilenameLen
			if fileLen <= 0 {
				fileLen = 50
			}

			runes := []rune(fileName)
			if len(runes) > fileLen {
				fileName = string(runes[:fileLen])
			}
		}

		if globalConfig.FilenameTime {
			mediaInfo.SavePath = filepath.Join(globalConfig.SaveDirectory, fileName+"_"+shared.GetCurrentDateTimeFormatted())
		} else {
			mediaInfo.SavePath = filepath.Join(globalConfig.SaveDirectory, fileName)
		}

		if !strings.HasSuffix(mediaInfo.SavePath, mediaInfo.Suffix) {
			mediaInfo.SavePath = mediaInfo.SavePath + mediaInfo.Suffix
		}

		if strings.Contains(rawUrl, "qq.com") {
			if globalConfig.Quality == 1 &&
				strings.Contains(rawUrl, "encfilekey=") &&
				strings.Contains(rawUrl, "token=") {
				parseUrl, err := url.Parse(rawUrl)
				queryParams := parseUrl.Query()
				if err == nil && queryParams.Has("encfilekey") && queryParams.Has("token") {
					rawUrl = parseUrl.Scheme + "://" + parseUrl.Host + "/" + parseUrl.Path +
						"?encfilekey=" + queryParams.Get("encfilekey") +
						"&token=" + queryParams.Get("token")
				}
			} else if globalConfig.Quality > 1 && mediaInfo.OtherData["wx_file_formats"] != "" {
				format := strings.Split(mediaInfo.OtherData["wx_file_formats"], "#")
				qualityMap := []string{
					format[0],
					format[len(format)/2],
					format[len(format)-1],
				}
				rawUrl += "&X-snsvideoflag=" + qualityMap[globalConfig.Quality-2]
			}
		}

		headers, _ := r.parseHeaders(mediaInfo)

		downloader := NewFileDownloader(rawUrl, mediaInfo.SavePath, globalConfig.TaskNumber, headers)
		downloader.progressCallback = func(totalDownloaded, totalSize float64, taskID int, taskProgress float64) {
			if totalSize > 0 {
				percent := int(totalDownloaded * 100 / totalSize)
				if percent < 0 {
					percent = 0
				} else if percent > 100 {
					percent = 100
				}
				r.progressEventsEmit(mediaInfo, fmt.Sprintf("%d%%", percent), shared.DownloadStatusRunning)
				return
			}
			r.progressEventsEmit(mediaInfo, fmt.Sprintf("%.0f bytes", totalDownloaded), shared.DownloadStatusRunning)
		}
		r.tasks.Store(mediaInfo.Id, downloader)
		defer r.tasks.Delete(mediaInfo.Id)
		err := downloader.Start()
		mediaInfo.SavePath = downloader.FileName
		if err != nil {
			if !strings.Contains(err.Error(), "cancelled") {
				r.progressEventsEmit(mediaInfo, err.Error())
			}
			return
		}
		if decodeStr != "" {
			r.progressEventsEmit(mediaInfo, "decrypting in progress", shared.DownloadStatusRunning)
			if err := r.decodeWxFile(mediaInfo.SavePath, decodeStr); err != nil {
				r.progressEventsEmit(mediaInfo, "decryption error: "+err.Error())
				return
			}
		}
		r.progressEventsEmit(mediaInfo, "complete", shared.DownloadStatusDone)
	}(mediaInfo)
}

func (r *Resource) parseHeaders(mediaInfo shared.MediaInfo) (map[string]string, error) {
	headers := make(map[string]string)

	if hh, ok := mediaInfo.OtherData["headers"]; ok {
		var tempHeaders map[string][]string
		if err := json.Unmarshal([]byte(hh), &tempHeaders); err != nil {
			return headers, fmt.Errorf("parse headers JSON err: %v", err)
		}

		for key, values := range tempHeaders {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
	}

	return headers, nil
}

func (r *Resource) wxFileDecode(mediaInfo shared.MediaInfo, fileName, decodeStr string) (string, error) {
	sourceFile, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer sourceFile.Close()
	mediaInfo.SavePath = strings.ReplaceAll(fileName, ".mp4", "_decrypt.mp4")

	destinationFile, err := os.Create(mediaInfo.SavePath)
	if err != nil {
		return "", err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return "", err
	}
	err = r.decodeWxFile(mediaInfo.SavePath, decodeStr)
	if err != nil {
		return "", err
	}
	return mediaInfo.SavePath, nil
}

func (r *Resource) progressEventsEmit(mediaInfo shared.MediaInfo, args ...string) {
	Status := shared.DownloadStatusError
	Message := "ok"

	if len(args) > 0 {
		Message = args[0]
	}
	if len(args) > 1 {
		Status = args[1]
	}
	r.updateDownloadStatus(mediaInfo.Id, Status, Message, mediaInfo.SavePath)

	httpServerOnce.send("downloadProgress", map[string]interface{}{
		"Id":       mediaInfo.Id,
		"Status":   Status,
		"SavePath": mediaInfo.SavePath,
		"Message":  Message,
	})
	return
}

func clearSyncMap(m *sync.Map) {
	m.Range(func(key, value interface{}) bool {
		m.Delete(key)
		return true
	})
}

func (r *Resource) decodeWxFile(fileName, decodeStr string) error {
	decodedBytes, err := base64.StdEncoding.DecodeString(decodeStr)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	byteCount := len(decodedBytes)
	fileBytes := make([]byte, byteCount)
	n, err := file.Read(fileBytes)
	if err != nil && err != io.EOF {
		return err
	}

	if n < byteCount {
		byteCount = n
	}

	xorResult := make([]byte, byteCount)
	for i := 0; i < byteCount; i++ {
		xorResult[i] = decodedBytes[i] ^ fileBytes[i]
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = file.Write(xorResult)
	if err != nil {
		return err
	}
	return nil
}
