package proxy

import (
    "net/http"
    "net/http/httputil"
    "net/url"
    "time"

    cache "github.com/Banner-babaner/proxytools/cache/usecase"
    cacheInfra "github.com/Banner-babaner/proxytools/cache/infrastructure"
    filter "github.com/Banner-babaner/proxytools/ipfilter/usecase"
    filterEnt "github.com/Banner-babaner/proxytools/ipfilter/entity"
    "github.com/Banner-babaner/proxytools/logger"
    monitor "github.com/Banner-babaner/proxytools/monitor/usecase"
    ratelimit "github.com/Banner-babaner/proxytools/ratelimit/usecase"
)


type ProxyHandler struct {
    reverseProxy *httputil.ReverseProxy
    ipFilter     *filter.FilterService
    rateLimiter  *ratelimit.LimiterService
    cacheService *cache.CacheService
    metrics      *monitor.MetricsService
}

func NewProxyHandler(
    upstreamURL string,
    ipFilter *filter.FilterService,
    rateLimiter *ratelimit.LimiterService,
    cacheService *cache.CacheService,
    metrics *monitor.MetricsService,
) (*ProxyHandler, error) {
    target, err := url.Parse(upstreamURL)
    if err != nil {
        return nil, err
    }

    proxy := httputil.NewSingleHostReverseProxy(target)
    
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


func (ph *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()
    clientIP := r.RemoteAddr

    access := ph.ipFilter.CheckAccess(clientIP)
    if access == filterEnt.Denied {
        ph.logDenied(clientIP, r.URL.String(), "blacklist")
        ph.metrics.RecordRequest(false, 0, 0, 0)
        http.Error(w, "Access denied", http.StatusForbidden)
        return
    }
    if access == filterEnt.CaptchaRequired {
        logger.Warn().Str("ip", clientIP).Msg("Captcha required")
    }

    if !ph.rateLimiter.Allow(clientIP) {
        ph.metrics.RecordRateLimit()
        ph.metrics.RecordRequest(false, time.Since(startTime).Seconds(), 0, 0)
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    cacheKey := ph.cacheService.GenerateKey(r.Method, r.URL.String())
    ttl := ph.cacheService.GetTTL(r.Method, r.URL.Path, r.Host)

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

    crw := cacheInfra.NewCacheResponseWriter(w)
    
    ph.reverseProxy.ServeHTTP(crw, r)

    duration := time.Since(startTime).Seconds()
    ph.metrics.RecordRequest(true, duration, r.ContentLength, int64(len(crw.Body())))

    if ttl > 0 && crw.StatusCode() >= 200 && crw.StatusCode() < 300 {
        ph.cacheService.Set(cacheKey, crw.StatusCode(), w.Header(), crw.Body(), ttl, nil)
    }

    logger.Info().
        Str("ip", clientIP).
        Str("method", r.Method).
        Str("url", r.URL.String()).
        Int("status", crw.StatusCode()).
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