// internal/proxy/proxy.go
package proxy

import (
    "net/http"
    "net/http/httputil"
    "net/url"
    "time"

    "github.com/Banner-babaner/proxytools/cache"
    "github.com/Banner-babaner/proxytools/ipfilter"
    "github.com/Banner-babaner/proxytools/logger"
    "github.com/Banner-babaner/proxytools/monitor"
    "github.com/Banner-babaner/proxytools/ratelimit"
)

// ProxyHandler основной обработчик прокси
type ProxyHandler struct {
    reverseProxy *httputil.ReverseProxy
    ipFilter     *ipfilter.FilterService
    rateLimiter  *ratelimit.LimiterService
    cacheService *cache.CacheService
    metrics      *monitor.MetricsService
}

func NewProxyHandler(
    upstreamURL string,
    ipFilter *ipfilter.FilterService,
    rateLimiter *ratelimit.LimiterService,
    cacheService *cache.CacheService,
    metrics *monitor.MetricsService,
) (*ProxyHandler, error) {
    target, err := url.Parse(upstreamURL)
    if err != nil {
        return nil, err
    }

    proxy := httputil.NewSingleHostReverseProxy(target)
    
    // Кастомный транспорт
    proxy.Transport = &http.Transport{
        MaxIdleConns:        100,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  false,
    }

    return &ProxyHandler{
        reverseProxy: proxy,
        ipFilter:     ipFilter,
        rateLimiter:  rateLimiter,
        cacheService: cacheService,
        metrics:      metrics,
    }, nil
}

// ServeHTTP обрабатывает запрос
func (ph *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()
    clientIP := r.RemoteAddr

    // 1. IP Filter
    access := ph.ipFilter.CheckAccess(clientIP)
    if access == ipfilter.Denied {
        ph.logDenied(clientIP, r.URL.String(), "blacklist")
        ph.metrics.RecordRequest(false, 0, 0, 0)
        http.Error(w, "Access denied", http.StatusForbidden)
        return
    }
    if access == ipfilter.CaptchaRequired {
        // TODO: реализовать CAPTCHA
        logger.Warn().Str("ip", clientIP).Msg("Captcha required")
    }

    // 2. Rate Limiting
    if !ph.rateLimiter.Allow(clientIP) {
        ph.metrics.RateLimitedCount.Add(1)
        ph.metrics.RecordRequest(false, time.Since(startTime).Seconds(), 0, 0)
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    // 3. Проверяем кэш
    cacheKey := ph.cacheService.GenerateKey(r.Method, r.URL.String())
    ttl := ph.cacheService.GetTTLForPath(r.Method, r.URL.Path, r.Host)

    if ttl > 0 {
        if entry, ok := ph.cacheService.Get(cacheKey); ok {
            ph.metrics.RecordCacheHit()
            ph.metrics.RecordRequest(true, time.Since(startTime).Seconds(), 0, int64(len(entry.Body)))
            
            for k, v := range entry.Headers {
                for _, val := range v {
                    w.Header().Add(k, val)
                }
            }
            w.Header().Set("X-Cache", "HIT")
            w.WriteHeader(entry.StatusCode)
            w.Write(entry.Body)
            
            logger.Debug().
                Str("ip", clientIP).
                Str("url", r.URL.String()).
                Msg("Served from cache")
            return
        }
        ph.metrics.RecordCacheMiss()
    }

    // 4. Проксируем запрос
    crw := cache.NewCacheResponseWriter(w)
    
    ph.reverseProxy.ServeHTTP(crw, r)

    duration := time.Since(startTime).Seconds()
    ph.metrics.RecordRequest(true, duration, r.ContentLength, int64(crw.Buffer.Len()))

    // 5. Кэшируем ответ (только успешные)
    if ttl > 0 && crw.StatusCode >= 200 && crw.StatusCode < 300 {
        ph.cacheService.Set(cacheKey, crw.StatusCode, w.Header(), crw.BodyBytes(), ttl, nil)
    }

    logger.Info().
        Str("ip", clientIP).
        Str("method", r.Method).
        Str("url", r.URL.String()).
        Int("status", crw.StatusCode).
        Dur("duration", time.Duration(duration*float64(time.Second))).
        Msg("Request processed")
}

func (ph *ProxyHandler) logDenied(ip, url, reason string) {
    logger.Warn().
        Str("ip", ip).
        Str("url", url).
        Str("reason", reason).
        Msg("Access denied")
}