package custommiddleware

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	servertiming "github.com/mitchellh/go-server-timing"
	"github.com/vicanso/fresh"
	"github.com/vicanso/pike/cache"
	"github.com/vicanso/pike/vars"
)

type (
	// FreshCheckerConfig freshChecker配置
	FreshCheckerConfig struct {
		Skipper middleware.Skipper
	}
)

// FreshChecker 判断请求是否fresh(304)
func FreshChecker(config FreshCheckerConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = middleware.DefaultSkipper
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}
			rid := c.Get(vars.RID).(string)
			debug := c.Logger().Debug
			cr, ok := c.Get(vars.Response).(*cache.Response)
			if !ok {
				debug(rid, " response not set")
				return vars.ErrResponseNotSet
			}
			statusCode := int(cr.StatusCode)
			method := c.Request().Method
			c.Set(vars.Fresh, false)
			if method != echo.GET && method != echo.HEAD {
				debug(rid, " method no need to check fresh")
				return next(c)
			}
			if statusCode < http.StatusOK || statusCode >= http.StatusBadRequest {
				debug(rid, " status no need to check fresh")
				return next(c)
			}
			timing, _ := c.Get(vars.Timing).(*servertiming.Header)
			var m *servertiming.Metric
			if timing != nil {
				m = timing.NewMetric(vars.FreshCheckerMetric)
				m.WithDesc("fresh checker").Start()
			}

			reqHeader := c.Request().Header
			resHeader := c.Response().Header()

			ifModifiedSince := reqHeader.Get(echo.HeaderIfModifiedSince)
			ifNoneMatch := reqHeader.Get(vars.IfNoneMatch)
			cacheControl := reqHeader.Get(vars.CacheControl)
			reqHeaderData := &fresh.RequestHeader{
				IfModifiedSince: []byte(ifModifiedSince),
				IfNoneMatch:     []byte(ifNoneMatch),
				CacheControl:    []byte(cacheControl),
			}
			eTag := resHeader.Get(vars.ETag)
			lastModified := resHeader.Get(echo.HeaderLastModified)
			resHeaderData := &fresh.ResponseHeader{
				ETag:         []byte(eTag),
				LastModified: []byte(lastModified),
			}

			// 如果请求还是fresh，则后续处理可返回304
			if fresh.Fresh(reqHeaderData, resHeaderData) {
				debug(rid, " is fresh")
				c.Set(vars.Fresh, true)
			}
			if m != nil {
				m.Stop()
			}
			debug(rid, " isn't fresh")
			return next(c)
		}
	}
}
