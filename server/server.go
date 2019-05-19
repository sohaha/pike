package server

import (
	"strings"

	"github.com/vicanso/cod"
	"github.com/vicanso/pike/cache"
	"github.com/vicanso/pike/config"
	"github.com/vicanso/pike/df"
	"github.com/vicanso/pike/middleware"
	"github.com/vicanso/pike/stats"
	"github.com/vicanso/pike/upstream"

	compress "github.com/vicanso/cod-compress"
	etag "github.com/vicanso/cod-etag"
	fresh "github.com/vicanso/cod-fresh"
	recover "github.com/vicanso/cod-recover"
)

// New create a cod server
func New(cfg *config.Config, director *upstream.Director, dsp *cache.Dispatcher, insStats *stats.Stats) *cod.Cod {
	d := cod.New()
	d.EnableTrace = cfg.GetEnableServerTiming()
	// 如果启动 server timing
	// 则在回调中调整响应头
	if d.EnableTrace {
		prefix := df.APP + "-"
		d.OnTrace(func(c *cod.Context, traceInfos cod.TraceInfos) {
			c.SetHeader(cod.HeaderServerTiming, string(traceInfos.ServerTiming(prefix)))
		})
	}

	// 如果有配置admin，则添加管理后台处理
	adminPath := cfg.GetAdminPath()
	if adminPath != "" {
		adminServer := NewAdminServer(cfg, director, dsp, insStats)
		d.Use(func(c *cod.Context) error {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, adminPath) {
				c.Request.URL.Path = path[len(adminPath):]
				c.Pass(adminServer)
				return nil
			}
			return c.Next()
		})
	}

	d.Use(recover.New())

	fn := middleware.NewInitialization(cfg, insStats)
	d.Use(fn)
	d.SetFunctionName(fn, "Initialization")

	fn = fresh.NewDefault()
	d.Use(fn)
	d.SetFunctionName(fn, "Fresh")

	// 可缓存数据在缓存时会生成gzip 与br
	fn = compress.NewWithDefaultCompressor(compress.Config{
		MinLength: cfg.GetCompressMinLength(),
		Level:     cfg.GetCompressLevel(),
		Checker:   cfg.GetTextFilter(),
	})
	d.Use(fn)
	d.SetFunctionName(fn, "Compress")

	fn = middleware.NewResponder()
	d.Use(fn)
	d.SetFunctionName(fn, "Responder")

	fn = middleware.NewCacheIdentifier(cfg, dsp)
	d.Use(fn)
	d.SetFunctionName(fn, "CacheIdentifier")

	fn = etag.NewDefault()
	d.Use(fn)
	d.SetFunctionName(fn, "ETag")

	fn = middleware.NewProxy(director)
	d.Use(fn)
	d.SetFunctionName(fn, "Proxy")

	noop := func(c *cod.Context) (err error) {
		return
	}
	d.ALL("/*url", noop)
	d.SetFunctionName(noop, "-")
	return d
}
