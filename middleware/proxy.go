package custommiddleware

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-server-timing"
	"github.com/vicanso/pike/cache"
	"github.com/vicanso/pike/proxy"
	"github.com/vicanso/pike/util"
	"github.com/vicanso/pike/vars"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// 根据echo proxy middleware 调整而来

type (
	// ProxyConfig defines the config for Proxy middleware.
	ProxyConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper middleware.Skipper

		// Rewrites defines URL path rewrite rules. The values captured in asterisk can be
		// retrieved by index e.g. $1, $2 and so on.
		// Examples:
		// "/old":              "/new",
		// "/api/*":            "/$1",
		// "/js/*":             "/public/javascripts/$1",
		// "/users/*/orders/*": "/user/$1/order/$2",
		Rewrites []string

		// Timeout 超时间隔
		Timeout time.Duration
		// ETag 是否生成ETag
		ETag bool

		rewriteRegex map[*regexp.Regexp]string
	}

	// ProxyTarget defines the upstream target.
	ProxyTarget struct {
		Name string
		URL  *url.URL
	}
	bodyDumpResponseWriter struct {
		body    *bytes.Buffer
		headers http.Header
		code    int
		http.ResponseWriter
	}
)

const (
	defaultTimeout = 10 * time.Second
)

// genETag 获取数据对应的ETag
func genETag(buf []byte) string {
	size := len(buf)
	if size == 0 {
		return "\"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk=\""
	}
	h := sha1.New()
	h.Write(buf)
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("\"%x-%s\"", size, hash)
}

func captureTokens(pattern *regexp.Regexp, input string) *strings.Replacer {
	groups := pattern.FindAllStringSubmatch(input, -1)
	if groups == nil {
		return nil
	}
	values := groups[0][1:]
	replace := make([]string, 2*len(values))
	for i, v := range values {
		j := 2 * i
		replace[j] = "$" + strconv.Itoa(i+1)
		replace[j+1] = v
	}
	return strings.NewReplacer(replace...)
}

func proxyHTTP(t *ProxyTarget) http.Handler {
	return httputil.NewSingleHostReverseProxy(t.URL)
}

// 根据Cache-Control的信息，获取s-maxage或者max-age的值
func getCacheAge(cacheControl []byte) uint16 {
	// cacheControl := header.PeekBytes(vars.CacheControl)
	if len(cacheControl) == 0 {
		return 0
	}
	// 如果设置不可缓存，返回0
	reg, _ := regexp.Compile(`no-cache|no-store|private`)
	match := reg.Match(cacheControl)
	if match {
		return 0
	}

	// 优先从s-maxage中获取
	reg, _ = regexp.Compile(`s-maxage=(\d+)`)
	result := reg.FindSubmatch(cacheControl)
	if len(result) == 2 {
		maxAge, _ := strconv.Atoi(string(result[1]))
		return uint16(maxAge)
	}

	// 从max-age中获取缓存时间
	reg, _ = regexp.Compile(`max-age=(\d+)`)
	result = reg.FindSubmatch(cacheControl)
	if len(result) != 2 {
		return 0
	}
	maxAge, _ := strconv.Atoi(string(result[1]))
	return uint16(maxAge)
}

// Proxy returns a Proxy middleware with config.
func Proxy(config ProxyConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = middleware.DefaultSkipper
	}
	config.rewriteRegex = map[*regexp.Regexp]string{}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	// Initialize
	for _, value := range config.Rewrites {
		arr := strings.Split(value, ":")
		if len(arr) != 2 {
			continue
		}
		k := arr[0]
		v := arr[1]
		k = strings.Replace(k, "*", "(\\S*)", -1)
		config.rewriteRegex[regexp.MustCompile(k)] = v
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			// 如果已获取到数据，则不需要proxy获取(已从cache中获取)
			if c.Get(vars.Response) != nil {
				return next(c)
			}
			// 获取director
			director, ok := c.Get(vars.Director).(*proxy.Director)
			if !ok {
				return vars.ErrDirectorNotFound
			}
			// 从director中选择可用的backend
			backend := director.Select(c)
			if len(backend) == 0 {
				return vars.ErrNoBackendAvaliable
			}

			timing, _ := c.Get(vars.Timing).(*servertiming.Header)
			var m *servertiming.Metric
			if timing != nil {
				m = timing.NewMetric(vars.GetResponseFromProxyMetric)
				m.WithDesc("get response from proxy").Start()
			}

			req := c.Request()
			reqHeader := req.Header
			targetURL, _ := url.Parse(backend)
			tgt := &ProxyTarget{
				Name: director.Name,
				URL:  targetURL,
			}

			// Rewrite
			for k, v := range config.rewriteRegex {
				replacer := captureTokens(k, req.URL.Path)
				if replacer != nil {
					req.URL.Path = replacer.Replace(v)
				}
			}

			// Fix header
			if reqHeader.Get(echo.HeaderXRealIP) == "" {
				reqHeader.Set(echo.HeaderXRealIP, c.RealIP())
			}
			if reqHeader.Get(echo.HeaderXForwardedProto) == "" {
				reqHeader.Set(echo.HeaderXForwardedProto, c.Scheme())
			}
			// Proxy
			writer := &bodyDumpResponseWriter{
				body:    new(bytes.Buffer),
				headers: make(http.Header),
			}
			// proxy时为了避免304的出现，因此调用时临时删除header
			ifModifiedSince := reqHeader.Get(echo.HeaderIfModifiedSince)
			ifNoneMatch := reqHeader.Get(vars.IfNoneMatch)
			if len(ifModifiedSince) != 0 {
				reqHeader.Del(echo.HeaderIfModifiedSince)
			}
			if len(ifNoneMatch) != 0 {
				reqHeader.Del(vars.IfNoneMatch)
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			done := make(chan bool)
			go func() {
				proxyHTTP(tgt).ServeHTTP(writer, req)
				done <- true
			}()
			select {
			case <-done:
			case <-ctx.Done():
				return vars.ErrGatewayTimeout
			}
			if len(ifModifiedSince) != 0 {
				reqHeader.Set(echo.HeaderIfModifiedSince, ifModifiedSince)
			}
			if len(ifNoneMatch) != 0 {
				reqHeader.Set(vars.IfNoneMatch, ifNoneMatch)
			}
			headers := writer.headers
			cacheControl := headers[vars.CacheControl]
			var ttl uint16
			if len(cacheControl) != 0 {
				// cache control 只会有一个http header
				ttl = getCacheAge([]byte(cacheControl[0]))
			}
			body := writer.body.Bytes()
			if config.ETag {
				eTagValue := util.GetHeaderValue(headers, vars.ETag)
				if len(eTagValue) == 0 {
					headers[vars.ETag] = []string{
						genETag(body),
					}
				}
			}
			cr := &cache.Response{
				CreatedAt:  uint32(time.Now().Unix()),
				TTL:        ttl,
				StatusCode: uint16(writer.code),
				Header:     headers,
			}
			contentEncoding := headers[echo.HeaderContentEncoding]
			if len(contentEncoding) == 0 {
				cr.Body = body
			} else {
				switch contentEncoding[0] {
				case vars.GzipEncoding:
					cr.GzipBody = body
				case vars.BrEncoding:
					cr.BrBody = body
				default:
					return vars.ErrContentEncodingNotSupport
				}
			}
			c.Set(vars.Response, cr)
			if m != nil {
				m.Stop()
			}
			return next(c)
		}
	}
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.code = code
}

func (w *bodyDumpResponseWriter) Header() http.Header {
	return w.headers
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}