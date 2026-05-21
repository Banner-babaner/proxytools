package ipfilter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewIPCache(t *testing.T) {
	cache, err := NewIPCache(100, 60*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, cache)
}

func TestNewIPCache_InvalidSize(t *testing.T) {
	_, err := NewIPCache(0, 60*time.Second)
	assert.Error(t, err)
}

func TestIPCache_SetAndGet(t *testing.T) {
	cache, _ := NewIPCache(100, 60*time.Second)

	cache.Set("192.168.1.1", Whitelist, true)

	listType, hasRule, found := cache.Get("192.168.1.1")
	assert.True(t, found)
	assert.True(t, hasRule)
	assert.Equal(t, Whitelist, listType)
}

func TestIPCache_SetAndGet_Blacklist(t *testing.T) {
	cache, _ := NewIPCache(100, 60*time.Second)

	cache.Set("10.0.0.1", Blacklist, true)

	listType, hasRule, found := cache.Get("10.0.0.1")
	assert.True(t, found)
	assert.True(t, hasRule)
	assert.Equal(t, Blacklist, listType)
}

func TestIPCache_SetAndGet_NoRule(t *testing.T) {
	cache, _ := NewIPCache(100, 60*time.Second)

	cache.Set("1.1.1.1", 0, false)

	listType, hasRule, found := cache.Get("1.1.1.1")
	assert.True(t, found)
	assert.False(t, hasRule)
	assert.Equal(t, ListType(0), listType)
}

func TestIPCache_Expired(t *testing.T) {
	cache, _ := NewIPCache(100, 1*time.Millisecond)

	cache.Set("192.168.1.1", Blacklist, true)
	time.Sleep(10 * time.Millisecond)

	_, _, found := cache.Get("192.168.1.1")
	assert.False(t, found)
}

func TestIPCache_NotFound(t *testing.T) {
	cache, _ := NewIPCache(100, 60*time.Second)

	_, _, found := cache.Get("10.0.0.1")
	assert.False(t, found)
}

func TestIPCache_Overwrite(t *testing.T) {
	cache, _ := NewIPCache(100, 60*time.Second)

	cache.Set("1.1.1.1", Whitelist, true)
	cache.Set("1.1.1.1", Blacklist, true)

	listType, _, found := cache.Get("1.1.1.1")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)
}

func TestIPCache_MultipleEntries(t *testing.T) {
	cache, _ := NewIPCache(100, 60*time.Second)

	cache.Set("1.1.1.1", Whitelist, true)
	cache.Set("2.2.2.2", Blacklist, true)
	cache.Set("3.3.3.3", Graylist, true)

	listType, _, found := cache.Get("1.1.1.1")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType)

	listType, _, found = cache.Get("2.2.2.2")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)

	listType, _, found = cache.Get("3.3.3.3")
	assert.True(t, found)
	assert.Equal(t, Graylist, listType)
}

func TestIPCache_Remove(t *testing.T) {
	cache, _ := NewIPCache(100, 60*time.Second)

	cache.Set("192.168.1.1", Whitelist, true)
	cache.cache.Remove("192.168.1.1")

	_, _, found := cache.Get("192.168.1.1")
	assert.False(t, found)
}

func TestIPCache_TTL(t *testing.T) {
	cache, _ := NewIPCache(100, 500*time.Millisecond)

	cache.Set("ip", Whitelist, true)

	// Ещё не истекло
	_, _, found := cache.Get("ip")
	assert.True(t, found)

	// Ждём половину TTL
	time.Sleep(250 * time.Millisecond)
	_, _, found = cache.Get("ip")
	assert.True(t, found)

	// Ждём истечения
	time.Sleep(300 * time.Millisecond)
	_, _, found = cache.Get("ip")
	assert.False(t, found)
}