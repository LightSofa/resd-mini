package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/ncruces/zenity"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"resd-mini/core/shared"
	"strings"
	"sync"
	"time"
)

type respData map[string]interface{}

type ResponseData struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type HttpServer struct {
	indexHTML []byte
	upGrader  websocket.Upgrader
	wsClients map[*websocket.Conn]bool
	broadcast chan []byte
	mutex     sync.RWMutex
}

func initHttpServer() *HttpServer {
	if httpServerOnce == nil {
		httpServerOnce = &HttpServer{
			upGrader: websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
			wsClients: make(map[*websocket.Conn]bool),
			broadcast: make(chan []byte, 1000),
		}
		file, err := appOnce.assets.ReadFile("web/dist/index.html")
		if err != nil {
			globalLogger.Error().Stack().Err(err)
		} else {
			httpServerOnce.indexHTML = file
		}
	}
	return httpServerOnce
}

func (h *HttpServer) run() {
	listener, err := net.Listen("tcp", globalConfig.Host+":"+globalConfig.Port)
	if err != nil {
		globalLogger.Err(err)
		log.Fatalf("Service cannot start: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ws", h.wsHandler)
	mux.HandleFunc("/api/install", h.install)
	mux.HandleFunc("/api/set-system-password", h.setSystemPassword)
	mux.HandleFunc("/api/preview", h.preview)
	mux.HandleFunc("/api/proxy-open", h.openSystemProxy)
	mux.HandleFunc("/api/proxy-unset", h.unsetSystemProxy)
	mux.HandleFunc("/api/open-directory", h.openDirectoryDialog)
	mux.HandleFunc("/api/open-file", h.openFileDialog)
	mux.HandleFunc("/api/open-folder", h.openFolder)
	mux.HandleFunc("/api/is-proxy", h.isProxy)
	mux.HandleFunc("/api/app-info", h.appInfo)
	mux.HandleFunc("/api/set-config", h.setConfig)
	mux.HandleFunc("/api/get-config", h.getConfig)
	mux.HandleFunc("/api/set-type", h.setType)
	mux.HandleFunc("/api/clear", h.clear)
	mux.HandleFunc("/api/delete", h.delete)
	mux.HandleFunc("/api/download", h.download)
	mux.HandleFunc("/api/cancel", h.cancel)
	mux.HandleFunc("/api/wx-file-decode", h.wxFileDecode)
	mux.HandleFunc("/api/batch-export", h.batchExport)
	mux.HandleFunc("/api/cert", h.downCert)

	// Script-friendly API
	mux.HandleFunc("/api/v1/health", h.health)
	mux.HandleFunc("/api/v1/resources", h.listResources)
	mux.HandleFunc("/api/v1/resource", h.getResource)
	mux.HandleFunc("/api/v1/proxy/open", h.openSystemProxy)
	mux.HandleFunc("/api/v1/proxy/unset", h.unsetSystemProxy)
	mux.HandleFunc("/api/v1/proxy/status", h.isProxy)
	mux.HandleFunc("/api/v1/config", h.v1Config)
	mux.HandleFunc("/api/v1/download", h.download)
	mux.HandleFunc("/api/v1/cancel", h.cancel)
	mux.HandleFunc("/api/v1/clear", h.clear)
	mux.HandleFunc("/api/v1/delete", h.delete)
	mux.HandleFunc("/api/v1/set-type", h.setType)
	mux.HandleFunc("/api/v1/wx-file-decode", h.wxFileDecode)

	// Static assets endpoint
	mux.HandleFunc("/", h.staticHandler)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panelHost := strings.HasSuffix(r.Host, ":"+globalConfig.Port)
			if panelHost {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				mux.ServeHTTP(w, r)
			} else {
				if err := normalizeTransparentRequest(r); err != nil {
					http.Error(w, "bad transparent request: "+err.Error(), http.StatusBadRequest)
					return
				}
				proxyOnce.Proxy.ServeHTTP(w, r)
			}
		}),
	}
	go h.handleMessages()
	fmt.Println("Service started, listening http://" + globalConfig.Host + ":" + globalConfig.Port)
	if err1 := server.Serve(listener); err1 != nil {
		globalLogger.Err(err1)
		fmt.Printf("Service startup exception: %v", err1)
	}
}

func normalizeTransparentRequest(r *http.Request) error {
	if r == nil || r.URL == nil {
		return fmt.Errorf("empty request")
	}
	if r.Method == http.MethodConnect {
		return nil
	}
	if r.URL.Scheme != "" && r.URL.Host != "" {
		return nil
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		return fmt.Errorf("missing host")
	}
	r.URL.Scheme = "http"
	r.URL.Host = host
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	return nil
}

func (h *HttpServer) staticHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(h.indexHTML)
		return
	}

	filePath := strings.TrimPrefix(r.URL.Path, "/")
	file, err := appOnce.assets.ReadFile("web/dist/" + filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Serve the file with correct content type
	http.ServeContent(w, r, filePath, time.Time{}, strings.NewReader(string(file)))
}

func (h *HttpServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upGrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	h.mutex.Lock()
	h.wsClients[conn] = true
	h.mutex.Unlock()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("WebSocket read error:", err)
			break
		}
		select {
		case h.broadcast <- message:
		default:
		}
	}
	h.mutex.Lock()
	delete(h.wsClients, conn)
	h.mutex.Unlock()
}

func (h *HttpServer) downCert(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-x509-ca-data")
	w.Header().Set("Content-Disposition", "attachment;filename=res-downloader-public.crt")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(appOnce.PublicCrt)))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, io.NopCloser(bytes.NewReader(appOnce.PublicCrt)))
}

func (h *HttpServer) preview(w http.ResponseWriter, r *http.Request) {
	realURL := r.URL.Query().Get("url")
	if realURL == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}
	realURL, _ = url.QueryUnescape(realURL)
	parsedURL, err := url.Parse(realURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	request, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		http.Error(w, "Failed to fetch the resource", http.StatusInternalServerError)
		return
	}

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		request.Header.Set("Range", rangeHeader)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		http.Error(w, "Failed to fetch the resource", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		if strings.ToLower(k) == "access-control-allow-origin" {
			continue
		}
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, "Failed to serve the resource", http.StatusInternalServerError)
	}
}

func (h *HttpServer) handleMessages() {
	for {
		msg := <-h.broadcast
		h.mutex.RLock()
		for client := range h.wsClients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				fmt.Printf("写入消息错误: %v", err)
				client.Close()
				h.mutex.Lock()
				delete(h.wsClients, client)
				h.mutex.Unlock()
			}
		}
		h.mutex.RUnlock()
	}
}

func (h *HttpServer) send(t string, data interface{}) {
	h.syncStateByEvent(t, data)

	h.mutex.RLock()
	hasClients := len(h.wsClients) > 0
	h.mutex.RUnlock()
	if !hasClients {
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"type": t,
		"data": data,
	})
	if err != nil {
		fmt.Println("Error converting map to JSON:", err)
		return
	}

	select {
	case h.broadcast <- jsonData:
	default:
		// Drop low-priority events under pressure to protect proxy throughput.
		if t != "downloadProgress" {
			return
		}
		h.broadcast <- jsonData
	}
}

func (h *HttpServer) syncStateByEvent(eventType string, data interface{}) {
	switch eventType {
	case "newResources":
		if media, ok := data.(shared.MediaInfo); ok {
			resourceOnce.rememberMedia(media)
		}
	case "downloadProgress":
		if payload, ok := data.(map[string]interface{}); ok {
			resourceOnce.updateDownloadStatus(
				toString(payload["Id"]),
				toString(payload["Status"]),
				toString(payload["Message"]),
				toString(payload["SavePath"]),
			)
		}
	}
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func (h *HttpServer) writeJson(w http.ResponseWriter, data *ResponseData) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		globalLogger.Err(err)
	}
}

func (h *HttpServer) error(w http.ResponseWriter, args ...interface{}) {
	message := "ok"
	var data interface{}

	if len(args) > 0 {
		message = args[0].(string)
	}
	if len(args) > 1 {
		data = args[1]
	}
	h.writeJson(w, h.buildResp(0, message, data))
}

func (h *HttpServer) success(w http.ResponseWriter, args ...interface{}) {
	message := "ok"
	var data interface{}

	if len(args) > 0 {
		data = args[0]
	}

	if len(args) > 1 {
		message = args[1].(string)
	}
	h.writeJson(w, h.buildResp(1, message, data))
}

func (h *HttpServer) buildResp(code int, message string, data interface{}) *ResponseData {
	return &ResponseData{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func (h *HttpServer) openDirectoryDialog(w http.ResponseWriter, r *http.Request) {
	folder, err := zenity.SelectFile(zenity.Filename(""), zenity.Directory())
	if err != nil {
		h.error(w, err.Error())
		return
	}
	h.success(w, respData{
		"folder": folder,
	})
}

func (h *HttpServer) openFileDialog(w http.ResponseWriter, r *http.Request) {
	filePath, err := zenity.SelectFile(
		zenity.Filename(""),
		zenity.FileFilters{
			{"Video files", []string{"*.mp4"}, false},
		})
	if err != nil {
		h.error(w, err.Error())
		return
	}
	h.success(w, respData{
		"file": filePath,
	})
}

func (h *HttpServer) openFolder(w http.ResponseWriter, r *http.Request) {
	var data struct {
		FilePath string `json:"filePath"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err == nil && data.FilePath == "" {
		return
	}

	err = shared.OpenFolder(data.FilePath)
	if err != nil {
		globalLogger.Err(err)
		h.error(w, err.Error())
		return
	}
	h.success(w)
}

func (h *HttpServer) install(w http.ResponseWriter, r *http.Request) {
	if appOnce.isInstall() {
		h.success(w, respData{
			"isPass": systemOnce.Password == "",
		})
		return
	}

	out, err := appOnce.installCert()
	if err != nil {
		h.error(w, err.Error()+"\n"+out, respData{
			"isPass": systemOnce.Password == "",
		})
		return
	}

	h.success(w, respData{
		"isPass": systemOnce.Password == "",
	})
}

func (h *HttpServer) setSystemPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Password string `json:"password"`
		IsCache  bool   `json:"isCache"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		h.error(w, err.Error())
		return
	}
	systemOnce.SetPassword(data.Password, data.IsCache)
	h.success(w)
}

func (h *HttpServer) openSystemProxy(w http.ResponseWriter, r *http.Request) {
	err := appOnce.OpenSystemProxy()
	if err != nil {
		h.error(w, err.Error(), respData{
			"value": appOnce.IsProxy,
		})
		return
	}
	h.success(w, respData{
		"value": appOnce.IsProxy,
	})
}

func (h *HttpServer) unsetSystemProxy(w http.ResponseWriter, r *http.Request) {
	err := appOnce.UnsetSystemProxy()
	if err != nil {
		h.error(w, err.Error(), respData{
			"value": appOnce.IsProxy,
		})
		return
	}
	h.success(w, respData{
		"value": appOnce.IsProxy,
	})
}

func (h *HttpServer) isProxy(w http.ResponseWriter, r *http.Request) {
	h.success(w, respData{
		"value": appOnce.IsProxy,
	})
}

func (h *HttpServer) appInfo(w http.ResponseWriter, r *http.Request) {
	h.success(w, appOnce)
}

func (h *HttpServer) health(w http.ResponseWriter, r *http.Request) {
	h.success(w, respData{
		"status":              "ok",
		"name":                appOnce.AppName,
		"port":                globalConfig.Port,
		"proxy":               appOnce.IsProxy,
		"gateway_transparent": systemOnce.GatewayTransparent,
	})
}

func (h *HttpServer) listResources(w http.ResponseWriter, r *http.Request) {
	h.success(w, respData{
		"items": resourceOnce.listMedia(),
	})
}

func (h *HttpServer) getResource(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		h.error(w, "missing id")
		return
	}
	media, ok := resourceOnce.getMediaByID(id)
	if !ok {
		h.error(w, "resource not found")
		return
	}
	h.success(w, media)
}

func (h *HttpServer) getConfig(w http.ResponseWriter, r *http.Request) {
	h.success(w, globalConfig)
}

func (h *HttpServer) v1Config(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.getConfig(w, r)
		return
	}
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		h.setConfig(w, r)
		return
	}
	h.error(w, "method not allowed")
}

func (h *HttpServer) setConfig(w http.ResponseWriter, r *http.Request) {
	var data Config
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.error(w, err.Error())
		return
	}
	globalConfig.setConfig(data)
	h.success(w)
}

func (h *HttpServer) setType(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Type string `json:"type"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err == nil {
		if data.Type != "" {
			resourceOnce.setResType(strings.Split(data.Type, ","))
		} else {
			resourceOnce.setResType([]string{})
		}
	}

	h.success(w)
}

func (h *HttpServer) clear(w http.ResponseWriter, r *http.Request) {
	resourceOnce.clear()
	h.success(w)
}

func (h *HttpServer) delete(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Sign []string `json:"sign"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err == nil && len(data.Sign) > 0 {
		for _, v := range data.Sign {
			resourceOnce.delete(v)
		}
	}
	h.success(w)
}

func (h *HttpServer) download(w http.ResponseWriter, r *http.Request) {
	var data struct {
		shared.MediaInfo
		DecodeStr string `json:"decodeStr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.error(w, err.Error())
		return
	}
	resourceOnce.rememberMedia(data.MediaInfo)
	resourceOnce.download(data.MediaInfo, data.DecodeStr)
	h.success(w)
}

func (h *HttpServer) cancel(w http.ResponseWriter, r *http.Request) {
	var data struct {
		shared.MediaInfo
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.error(w, err.Error())
		return
	}

	err := resourceOnce.cancel(data.Id)
	if err != nil {
		h.error(w, err.Error())
		return
	}
	h.success(w)
}

func (h *HttpServer) wxFileDecode(w http.ResponseWriter, r *http.Request) {
	var data struct {
		shared.MediaInfo
		Filename  string `json:"filename"`
		DecodeStr string `json:"decodeStr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.error(w, err.Error())
		return
	}
	savePath, err := resourceOnce.wxFileDecode(data.MediaInfo, data.Filename, data.DecodeStr)
	if err != nil {
		h.error(w, err.Error())
		return
	}
	h.success(w, respData{
		"save_path": savePath,
	})
}

func (h *HttpServer) batchExport(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.error(w, err.Error())
		return
	}
	saveDir := strings.TrimSpace(globalConfig.SaveDirectory)
	if saveDir == "" {
		h.error(w, "save directory is empty")
		return
	}
	if err := os.MkdirAll(saveDir, 0750); err != nil {
		h.error(w, "create save directory failed: "+err.Error())
		return
	}
	fileName := filepath.Join(saveDir, "res-downloader-"+shared.GetCurrentDateTimeFormatted()+".txt")
	err := os.WriteFile(fileName, []byte(data.Content), 0644)
	if err != nil {
		h.error(w, err.Error())
		return
	}
	h.success(w, respData{
		"file_name": fileName,
	})
}
