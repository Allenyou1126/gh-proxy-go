package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"io"
	"log/slog"
	"net/http"
	"os"
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
	slog.Debug("Loading static files.")
	var err error

	slog.Debug("Loading index.html")
	if indexHTML, err = fs.ReadFile("static/index.html"); err != nil {
		slog.Error("Error loading static files.")
		os.Exit(1)
	}
	slog.Debug("index.html loaded.")

	slog.Debug("Loading favicon.ico")
	if iconData, err = fs.ReadFile("static/favicon.ico"); err != nil {
		slog.Error("Error loading static files.")
		os.Exit(1)
	}
	slog.Debug("favicon.ico loaded.")
}

func main() {
	slog.Debug("Debug mode enabled.")

	slog.Debug("Creating routes.")
	if DEBUG_MODE {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	slog.Debug("Routes created.")

	slog.Debug("Registering middlewares.")
	slogGinConfig := sloggin.Config{
		WithUserAgent: true,
		WithRequestID: true,
		WithTraceID:   true,
		Filters:       nil,
	}
	r.Use(sloggin.NewWithConfig(slog.Default(), slogGinConfig))
	r.Use(gin.Recovery())
	slog.Debug("Middlewares registered.")

	slog.Debug("Registering route handler.")
	r.Any("/*path", handler)
	slog.Debug("Route handler registered.")

	slog.Debug("Launching server.")
	err := r.Run(fmt.Sprintf("%s:%d", HOST, PORT))
	if err != nil {
		slog.Error("Error launching server.")
		os.Exit(1)
	}
}

func handler(c *gin.Context) {
	path := c.Param("path")
	slog.Debug("Handling request.", "path", path)

	if path == "/favicon.ico" {
		slog.Debug("Getting favicon.ico")
		c.Data(http.StatusOK, "image/vnd.microsoft.icon", iconData)
		return
	}

	if path == "/" {
		if q := c.Query("q"); q != "" {
			slog.Debug("Redirecting arg q.", "q", q)
			c.Redirect(http.StatusFound, "/"+q)
			return
		}

		slog.Debug("Getting index.html")
		c.Data(http.StatusOK, "text/html", indexHTML)
		return
	}

	u := strings.TrimPrefix(path, "/")
	slog.Debug("Preprocessing url.", "origin", u)

	if !strings.HasPrefix(u, "http") {
		u = "https://" + u
		slog.Debug("Add https:// prefix.", "u", u)
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = strings.Replace(u, "s:/", "s://", 1)
		slog.Debug("Fix scheme prefix.", "u", u)
	}
	slog.Debug("Preprocessing finished.", "url", u)

	if m := checkURL(u); m != nil {
		slog.Debug("Found repo in url.", "m", m)
		if !checkWhiteList(m) {
			c.String(http.StatusForbidden, "Forbidden by white list.")
			slog.Debug("Forbidden by white list.")
			return
		}
		if checkBlackList(m) {
			c.String(http.StatusForbidden, "Forbidden by black list.")
			slog.Debug("Forbidden by black list.")
			return
		}
		if checkPassList(m) {
			slog.Debug("Use pass list.")
			handlePassList(c, u)
			return
		}
	}

	if USE_JSDELIVR_AS_MIRROR_FOR_BRANCHES {
		if newURL := processJsDelivr(u); newURL != "" {
			slog.Debug("Use jsdelivr as mirror.", "new_url", newURL)
			c.Redirect(http.StatusFound, newURL)
			return
		}
	}

	slog.Debug("Use proxy.")
	proxyHandler(c, u)
}

func checkURL(u string) []string {
	slog.Debug("Matching repos in URL.", "url", u)
	for _, re := range []*regexp.Regexp{exp1, exp2, exp3, exp4, exp5} {
		if matches := re.FindStringSubmatch(u); matches != nil {
			slog.Debug("Matched in URL.", "url", u, "matches", matches)
			return matches[1:]
		}
	}
	slog.Debug("Nothing matched in URL.", "url", u)
	return nil
}

func processJsDelivr(u string) string {
	slog.Debug("Processing JsDelivr.", "u", u)
	if matches := exp2.FindStringSubmatch(u); matches != nil {
		ret := strings.Replace(u, "/blob/", "@", 1)
		slog.Debug("Matched blobs.", "matches", matches, "result", ret)
		return ret
	}
	if matches := exp4.FindStringSubmatch(u); matches != nil {
		ret := regexp.MustCompile(`(.+?/.+?)/(.+?/)`).ReplaceAllString(u, "$1@$2")
		slog.Debug("Matched repo rules.", "matches", matches, "result", ret)
		return ret
	}
	slog.Debug("Not matched.")
	return ""
}

func handlePassList(c *gin.Context, u string) {
	targetURL := u + c.Request.URL.RawQuery
	slog.Debug("Processing pass list.", "u", u, "target", targetURL)
	if strings.HasPrefix(targetURL, "https:/") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL[7:]
		slog.Debug("Fix scheme prefix.", "target", targetURL)
	}
	slog.Debug("Redirecting.", "target", targetURL)
	c.Redirect(http.StatusFound, targetURL)
}

func proxyHandler(c *gin.Context, targetURL string) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: SKIP_TLS_VERIFYING},
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
	slog.Debug("Copying headers.", "src", src)
	for k, vv := range src {
		if k == "Host" {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
	slog.Debug("Headers copied.", "dst", dst)
}
