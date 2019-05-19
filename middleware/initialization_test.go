package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vicanso/cod"
	"github.com/vicanso/pike/config"
	"github.com/vicanso/pike/stats"
)

func TestNewInitialization(t *testing.T) {
	cfg := config.New()
	cfg.Viper.Set("header", []string{
		"X-Response-ID:456",
	})
	cfg.Viper.Set("requestHeader", []string{
		"X-Token:ab",
		"X-Request-ID:123",
	})

	fn := NewInitialization(cfg, stats.New())
	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	c := cod.NewContext(resp, req)
	c.Next = func() error {
		if c.GetHeader("X-Response-ID") != "" {
			t.Fatalf("the response id should be set after next")
		}

		if c.GetRequestHeader("X-Token") != "ab" ||
			c.GetRequestHeader("X-Request-ID") != "123" {
			t.Fatalf("the request header should be set before next")
		}
		return nil
	}
	err := fn(c)
	if err != nil {
		t.Fatalf("init middleware fail, %v", err)
	}
	if c.GetHeader("X-Response-ID") != "456" {
		t.Fatalf("response id is not set success")
	}
}

func TestTooManyRequest(t *testing.T) {
	cfg := config.New()
	max := cfg.GetConcurrency()
	cfg.Viper.Set("concurrency", 1)
	defer cfg.Viper.Set("concurrency", max)
	fn := NewInitialization(cfg, stats.New())

	c1 := cod.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	c1.Next = func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}
	go func() {
		err := fn(c1)
		if err != nil {
			panic(err)
		}
	}()
	c2 := cod.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	time.Sleep(time.Millisecond)
	err := fn(c2)
	if err != errTooManyRequest {
		t.Fatalf("should return too many request error")
	}
}
