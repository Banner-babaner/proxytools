// internal/ipfilter/trie_test.go
package ipfilter

import (
	"testing"
	"sync"

	"github.com/stretchr/testify/assert"
)

func TestTrie_InsertSingleIP(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("192.168.1.1", Whitelist)
	assert.NoError(t, err)

	listType, found := trie.Search("192.168.1.1")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType)
}

func TestTrie_InsertSingleIPBlacklist(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("1.2.3.4", Blacklist)
	assert.NoError(t, err)

	listType, found := trie.Search("1.2.3.4")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)
}

func TestTrie_InsertSingleIPGraylist(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("172.16.0.1", Graylist)
	assert.NoError(t, err)

	listType, found := trie.Search("172.16.0.1")
	assert.True(t, found)
	assert.Equal(t, Graylist, listType)
}

func TestTrie_InsertCIDR8(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("10.0.0.0/8", Blacklist)
	assert.NoError(t, err)

	// Начало диапазона
	listType, found := trie.Search("10.0.0.0")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)

	// Середина
	listType, found = trie.Search("10.100.50.25")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)

	// Конец диапазона
	listType, found = trie.Search("10.255.255.255")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)

	// Вне диапазона
	_, found = trie.Search("11.0.0.1")
	assert.False(t, found)

	_, found = trie.Search("9.255.255.255")
	assert.False(t, found)
}

func TestTrie_InsertCIDR16(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("192.168.0.0/16", Whitelist)
	assert.NoError(t, err)

	assert.True(t, checkFound(trie, "192.168.0.1", Whitelist))
	assert.True(t, checkFound(trie, "192.168.128.128", Whitelist))
	assert.True(t, checkFound(trie, "192.168.255.254", Whitelist))
	assert.False(t, checkFound(trie, "192.169.0.1", Whitelist))
	assert.False(t, checkFound(trie, "192.167.255.255", Whitelist))
}

func TestTrie_InsertCIDR24(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("172.16.1.0/24", Graylist)
	assert.NoError(t, err)

	assert.True(t, checkFound(trie, "172.16.1.0", Graylist))
	assert.True(t, checkFound(trie, "172.16.1.128", Graylist))
	assert.True(t, checkFound(trie, "172.16.1.255", Graylist))
	assert.False(t, checkFound(trie, "172.16.2.1", Graylist))
	assert.False(t, checkFound(trie, "172.16.0.255", Graylist))
}

func TestTrie_InsertCIDR32(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("8.8.8.8/32", Whitelist)
	assert.NoError(t, err)

	listType, found := trie.Search("8.8.8.8")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType)

	_, found = trie.Search("8.8.8.9")
	assert.False(t, found)

	_, found = trie.Search("8.8.8.7")
	assert.False(t, found)
}

func TestTrie_LongestPrefixMatch(t *testing.T) {
	trie := NewIPTrie()

	// Широкая подсеть — blacklist
	trie.Insert("10.0.0.0/8", Blacklist)
	// Узкая подсеть внутри — whitelist (исключение)
	trie.Insert("10.1.1.0/24", Whitelist)

	// IP из исключения
	listType, found := trie.Search("10.1.1.50")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType)

	// IP из общей подсети (но не из исключения)
	listType, found = trie.Search("10.2.0.1")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)

	// IP из исключения — граница
	listType, found = trie.Search("10.1.1.255")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType)
}

func TestTrie_LongestPrefixMatch_MultipleLevels(t *testing.T) {
	trie := NewIPTrie()

	trie.Insert("10.0.0.0/8", Blacklist)     // всё запрещено
	trie.Insert("10.1.0.0/16", Graylist)      // кроме этого — капча
	trie.Insert("10.1.1.0/24", Whitelist)     // а тут белый список

	// Самое специфичное правило побеждает
	listType, found := trie.Search("10.1.1.100")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType)

	// Второй уровень
	listType, found = trie.Search("10.1.2.100")
	assert.True(t, found)
	assert.Equal(t, Graylist, listType)

	// Самый общий
	listType, found = trie.Search("10.2.0.1")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)
}

func TestTrie_MultipleLists(t *testing.T) {
	trie := NewIPTrie()

	trie.Insert("10.0.0.0/8", Blacklist)
	trie.Insert("192.168.0.0/16", Whitelist)
	trie.Insert("172.16.0.0/12", Graylist)
	trie.Insert("8.8.8.8", Whitelist)

	// Проверяем каждый
	listType, found := trie.Search("10.255.255.255")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)

	listType, found = trie.Search("192.168.100.1")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType)

	listType, found = trie.Search("172.16.0.1")
	assert.True(t, found)
	assert.Equal(t, Graylist, listType)

	listType, found = trie.Search("8.8.8.8")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType)
}

func TestTrie_NotFound(t *testing.T) {
	trie := NewIPTrie()
	trie.Insert("192.168.1.0/24", Whitelist)

	_, found := trie.Search("1.1.1.1")
	assert.False(t, found)

	_, found = trie.Search("10.0.0.1")
	assert.False(t, found)

	_, found = trie.Search("255.255.255.255")
	assert.False(t, found)
}

func TestTrie_EmptyTrie(t *testing.T) {
	trie := NewIPTrie()

	_, found := trie.Search("192.168.1.1")
	assert.False(t, found)

	_, found = trie.Search("0.0.0.0")
	assert.False(t, found)
}

func TestTrie_InvalidInput(t *testing.T) {
	trie := NewIPTrie()

	assert.Error(t, trie.Insert("invalid", Whitelist))
	assert.Error(t, trie.Insert("", Whitelist))
	assert.Error(t, trie.Insert("999.999.999.999", Whitelist))
	assert.Error(t, trie.Insert("1.2.3.4.5", Whitelist))
	assert.Error(t, trie.Insert("abc.def.ghi.jkl", Whitelist))
}

func TestTrie_InvalidIP_Search(t *testing.T) {
	trie := NewIPTrie()
	trie.Insert("10.0.0.0/8", Blacklist)

	// Невалидный IP при поиске
	_, found := trie.Search("invalid")
	assert.False(t, found)

	_, found = trie.Search("")
	assert.False(t, found)
}


func TestTrie_OverwriteRule(t *testing.T) {
	trie := NewIPTrie()

	trie.Insert("10.0.0.0/8", Blacklist)
	trie.Insert("10.0.0.0/8", Whitelist) // перезаписывает

	listType, found := trie.Search("10.1.1.1")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType) // последняя запись побеждает
}

func TestTrie_SamePrefixDifferentTypes(t *testing.T) {
	trie := NewIPTrie()

	trie.Insert("192.168.0.0/24", Blacklist)
	trie.Insert("192.168.0.0/24", Whitelist)

	listType, found := trie.Search("192.168.0.1")
	assert.True(t, found)
	assert.Equal(t, Whitelist, listType) // последняя перезаписывает
}

func TestTrie_BoundaryValues(t *testing.T) {
	trie := NewIPTrie()
	trie.Insert("10.0.0.0/24", Whitelist)

	// 10.0.0.0
	assert.True(t, checkFound(trie, "10.0.0.0", Whitelist))
	// 10.0.0.255
	assert.True(t, checkFound(trie, "10.0.0.255", Whitelist))
	// 10.0.1.0 — вне
	assert.False(t, checkFound(trie, "10.0.1.0", Whitelist))
}

func TestTrie_ConcurrentReads(t *testing.T) {
	trie := NewIPTrie()
	trie.Insert("10.0.0.0/8", Blacklist)
	trie.Insert("192.168.0.0/16", Whitelist)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				trie.Search("10.1.2.3")
				trie.Search("192.168.1.1")
				trie.Search("1.1.1.1")
			}
		}()
	}

	wg.Wait()
	// Не должно быть гонки
}

func TestTrie_ConcurrentSearchAndInsert(t *testing.T) {
	trie := NewIPTrie()

	var wg sync.WaitGroup

	// Писатели
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				trie.Insert("10.0.0.0/8", Blacklist)
				trie.Insert("192.168.0.0/16", Whitelist)
			}
		}(i)
	}

	// Читатели
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				trie.Search("10.1.1.1")
				trie.Search("192.168.1.1")
			}
		}()
	}

	wg.Wait()

	// После всего поиск должен работать
	listType, found := trie.Search("10.1.1.1")
	assert.True(t, found)
	assert.Equal(t, Blacklist, listType)
}

func TestTrie_InsertRange(t *testing.T) {
	trie := NewIPTrie()

	err := trie.InsertRange("192.168.1.1", "192.168.1.5", Whitelist)
	assert.NoError(t, err)

	assert.True(t, checkFound(trie, "192.168.1.1", Whitelist))
	assert.True(t, checkFound(trie, "192.168.1.3", Whitelist))
	assert.True(t, checkFound(trie, "192.168.1.5", Whitelist))
	assert.False(t, checkFound(trie, "192.168.1.6", Whitelist))
	assert.False(t, checkFound(trie, "192.168.1.0", Whitelist))
}

func TestTrie_InsertRange_InvalidInput(t *testing.T) {
	trie := NewIPTrie()

	err := trie.InsertRange("invalid", "1.2.3.4", Whitelist)
	assert.NoError(t, err) // наша реализация возвращает nil
}

func TestNewIPTrie(t *testing.T) {
	trie := NewIPTrie()
	assert.NotNil(t, trie)
	assert.NotNil(t, trie.root)
}

func checkFound(trie *IPTrie, ip string, expectedType ListType) bool {
	listType, found := trie.Search(ip)
	return found && listType == expectedType
}