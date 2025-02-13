package main

import (
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"regexp"
	"strings"
)

//go:embed static/*
var fs embed.FS

var (
	indexHTML []byte
	iconData  []byte

	exp1 = regexp.MustCompile(`^(?:https?://)?github\.com/(?P<author>.+?)/(?P<repo>.+?)/(?:releases|archive)/.*$`)
	exp2 = regexp.MustCompile(`^(?:https?://)?github\.com/(?P<author>.+?)/(?P<repo>.+?)/(?:blob|raw)/.*$`)
	exp3 = regexp.MustCompile(`^(?:https?://)?github\.com/(?P<author>.+?)/(?P<repo>.+?)/(?:info|git-).*$`)
	exp4 = regexp.MustCompile(`^(?:https?://)?raw\.(?:githubusercontent|github)\.com/(?P<author>.+?)/(?P<repo>.+?)/.+?/.+$`)
	exp5 = regexp.MustCompile(`^(?:https?://)?gist\.(?:githubusercontent|github)\.com/(?P<author>.+?)/.+?/.+$`)
)

func init() {
	// 预加载静态资源
	var err error
	if indexHTML, err = fs.ReadFile("static/index.html"); err != nil {
		panic(err)
	}
	if iconData, err = fs.ReadFile("static/favicon.ico"); err != nil {
		panic(err)
	}
}

func main() {
	r := gin.New()
	slogGinConfig := sloggin.Config{
		WithUserAgent: true,
		WithRequestID: true,
		WithTraceID:   true,
		Filters:       nil,
	}
	r.Use(sloggin.NewWithConfig(slog.Default(), slogGinConfig))
	r.Use(gin.Recovery())
	r.Any("/*path", handler)
	err := r.Run(fmt.Sprintf("%s:%d", HOST, PORT))
	if err != nil {
		return
	}
}

func handler(c *gin.Context) {
	path := c.Param("path")
	if path == "/favicon.ico" {
		c.Data(http.StatusOK, "image/vnd.microsoft.icon", iconData)
		return
	}
	if path == "/" {
		if q := c.Query("q"); q != "" {
			c.Redirect(http.StatusFound, "/"+q)
			return
		}
		c.Data(http.StatusOK, "text/html", indexHTML)
		return
	}
	u := strings.TrimPrefix(path, "/")
	if !strings.HasPrefix(u, "http") {
		u = "https://" + u
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = strings.Replace(u, "s:/", "s://", 1)
	}
	if m := checkURL(u); m != nil {
		if !checkWhiteList(m) {
			c.String(http.StatusForbidden, "Forbidden by white list.")
			return
		}
		if checkBlackList(m) {
			c.String(http.StatusForbidden, "Forbidden by black list.")
			return
		}
		if checkPassList(m) {
			handlePassList(c, u)
			return
		}
	}
	if USE_JSDELIVER_AS_MIRROR_FOR_BRANCHES {
		if newURL := processJsDelivr(u); newURL != "" {
			c.Redirect(http.StatusFound, newURL)
			return
		}
	}
	proxyHandler(c, u)
}

func checkURL(u string) []string {
	for _, re := range []*regexp.Regexp{exp1, exp2, exp3, exp4, exp5} {
		if matches := re.FindStringSubmatch(u); matches != nil {
			return matches[1:]
		}
	}
	return nil
}

func processJsDelivr(u string) string {
	if matches := exp2.FindStringSubmatch(u); matches != nil {
		return strings.Replace(u, "/blob/", "@", 1)
	}
	if matches := exp4.FindStringSubmatch(u); matches != nil {
		return regexp.MustCompile(`(.+?/.+?)/(.+?/)`).ReplaceAllString(u, "$1@$2")
	}
	return ""
}

func handlePassList(c *gin.Context, u string) {
	targetURL := u + c.Request.URL.RawQuery
	if strings.HasPrefix(targetURL, "https:/") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL[7:]
	}
	c.Redirect(http.StatusFound, targetURL)
}

func proxyHandler(c *gin.Context, targetURL string) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "server error "+err.Error())
		return
	}

	copyHeaders(req.Header, c.Request.Header)

	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, "server error "+err.Error())
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if location := resp.Header.Get("Location"); location != "" {
		if checkURL(location) != nil {
			resp.Header.Set("Location", "/"+location)
		}
	}

	copyHeaders(c.Writer.Header(), resp.Header)
	c.Status(resp.StatusCode)

	if resp.ContentLength > SIZE_LIMIT {
		c.Redirect(http.StatusFound, targetURL)
		return
	}

	buf := make([]byte, CHUNK_SIZE)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, err := c.Writer.Write(buf[:n])
			if err != nil {
				return
			}
			c.Writer.Flush()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		if k == "Host" {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
