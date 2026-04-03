package plugins

import (
	"encoding/json"
	"github.com/elazarl/goproxy"
	"net/http"
	"path/filepath"
	"resd-mini/core/shared"
	"strconv"
	"strings"
)

type DefaultPlugin struct {
	bridge *shared.Bridge
}

var keptHeaderKeys = []string{
	"Accept",
	"Accept-Language",
	"Authorization",
	"Cache-Control",
	"Cookie",
	"Dnt",
	"Origin",
	"Pragma",
	"Range",
	"Referer",
	"Sec-Ch-Ua",
	"Sec-Ch-Ua-Mobile",
	"Sec-Ch-Ua-Platform",
	"Sec-Fetch-Dest",
	"Sec-Fetch-Mode",
	"Sec-Fetch-Site",
	"User-Agent",
}

var ignoredStreamSuffix = map[string]struct{}{
	".css":   {},
	".js":    {},
	".json":  {},
	".map":   {},
	".ttf":   {},
	".woff":  {},
	".woff2": {},
}

func (p *DefaultPlugin) SetBridge(bridge *shared.Bridge) {
	p.bridge = bridge
}

func (p *DefaultPlugin) Domains() []string {
	return []string{"default"}
}

func (p *DefaultPlugin) OnRequest(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return r, nil
}

func compactHeaders(h http.Header) map[string][]string {
	out := make(map[string][]string, len(keptHeaderKeys))
	for _, key := range keptHeaderKeys {
		if values := h.Values(key); len(values) > 0 {
			out[key] = values
		}
	}
	return out
}

func (p *DefaultPlugin) OnResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil || resp.Request == nil || (resp.StatusCode != 200 && resp.StatusCode != 206 && resp.StatusCode != 304) {
		return resp
	}

	classify, suffix := p.bridge.TypeSuffix(resp.Header.Get("Content-Type"))
	if classify == "" {
		return resp
	}

	rawUrl := resp.Request.URL.String()
	isAll, _ := p.bridge.GetResType("all")
	isClassify, _ := p.bridge.GetResType(classify)

	if suffix == "default" {
		ext := filepath.Ext(filepath.Base(strings.Split(strings.Split(rawUrl, "?")[0], "#")[0]))
		if ext != "" {
			suffix = ext
		}
	}

	if classify == "stream" {
		if _, skip := ignoredStreamSuffix[strings.ToLower(suffix)]; skip {
			return resp
		}
	}

	urlSign := shared.Md5(rawUrl)
	if ok := p.bridge.MediaIsMarked(urlSign); !ok && (isAll || isClassify) {
		value, _ := strconv.ParseFloat(resp.Header.Get("content-length"), 64)
		res := shared.MediaInfo{
			Id:          urlSign,
			Url:         rawUrl,
			UrlSign:     urlSign,
			CoverUrl:    "",
			Size:        value,
			Domain:      shared.GetTopLevelDomain(rawUrl),
			Classify:    classify,
			Suffix:      suffix,
			Status:      shared.DownloadStatusReady,
			SavePath:    "",
			DecodeKey:   "",
			OtherData:   map[string]string{},
			Description: "",
			ContentType: resp.Header.Get("Content-Type"),
		}

		// Keep only headers useful for replay/downloading to cut CPU and memory pressure.
		if headers, err := json.Marshal(compactHeaders(resp.Request.Header)); err == nil {
			res.OtherData["headers"] = string(headers)
		}

		p.bridge.MarkMedia(urlSign)
		p.bridge.Send("newResources", res)
	}

	return resp
}
