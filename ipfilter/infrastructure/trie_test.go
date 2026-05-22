package infrastructure

import (
	"testing"
	"sync"

	"github.com/Banner-babaner/proxytools/ipfilter/entity"
	"github.com/stretchr/testify/assert"
)

func TestTrie_InsertSingleIP(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("192.168.1.1", entity.Whitelist)
	assert.NoError(t, err)

	listType, found := trie.Search("192.168.1.1")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType)
}

func TestTrie_InsertSingleIPBlacklist(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("1.2.3.4", entity.Blacklist)
	assert.NoError(t, err)

	listType, found := trie.Search("1.2.3.4")
	assert.True(t, found)
	assert.Equal(t, entity.Blacklist, listType)
}

func TestTrie_InsertSingleIPGraylist(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("172.16.0.1", entity.Graylist)
	assert.NoError(t, err)

	listType, found := trie.Search("172.16.0.1")
	assert.True(t, found)
	assert.Equal(t, entity.Graylist, listType)
}

func TestTrie_InsertCIDR8(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("10.0.0.0/8", entity.Blacklist)
	assert.NoError(t, err)

	// Начало диапазона
	listType, found := trie.Search("10.0.0.0")
	assert.True(t, found)
	assert.Equal(t, entity.Blacklist, listType)

	// Середина
	listType, found = trie.Search("10.100.50.25")
	assert.True(t, found)
	assert.Equal(t, entity.Blacklist, listType)

	// Конец диапазона
	listType, found = trie.Search("10.255.255.255")
	assert.True(t, found)
	assert.Equal(t, entity.Blacklist, listType)

	// Вне диапазона
	_, found = trie.Search("11.0.0.1")
	assert.False(t, found)

	_, found = trie.Search("9.255.255.255")
	assert.False(t, found)
}

func TestTrie_InsertCIDR16(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("192.168.0.0/16", entity.Whitelist)
	assert.NoError(t, err)

	assert.True(t, checkFound(trie, "192.168.0.1", entity.Whitelist))
	assert.True(t, checkFound(trie, "192.168.128.128", entity.Whitelist))
	assert.True(t, checkFound(trie, "192.168.255.254", entity.Whitelist))
	assert.False(t, checkFound(trie, "192.169.0.1", entity.Whitelist))
	assert.False(t, checkFound(trie, "192.167.255.255", entity.Whitelist))
}

func TestTrie_InsertCIDR24(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("172.16.1.0/24", entity.Graylist)
	assert.NoError(t, err)

	assert.True(t, checkFound(trie, "172.16.1.0", entity.Graylist))
	assert.True(t, checkFound(trie, "172.16.1.128", entity.Graylist))
	assert.True(t, checkFound(trie, "172.16.1.255", entity.Graylist))
	assert.False(t, checkFound(trie, "172.16.2.1", entity.Graylist))
	assert.False(t, checkFound(trie, "172.16.0.255", entity.Graylist))
}

func TestTrie_InsertCIDR32(t *testing.T) {
	trie := NewIPTrie()

	err := trie.Insert("8.8.8.8/32", entity.Whitelist)
	assert.NoError(t, err)

	listType, found := trie.Search("8.8.8.8")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType)

	_, found = trie.Search("8.8.8.9")
	assert.False(t, found)

	_, found = trie.Search("8.8.8.7")
	assert.False(t, found)
}

func TestTrie_LongestPrefixMatch(t *testing.T) {
	trie := NewIPTrie()

	// Широкая подсеть — blacklist
	trie.Insert("10.0.0.0/8", entity.Blacklist)
	// Узкая подсеть внутри — whitelist (исключение)
	trie.Insert("10.1.1.0/24", entity.Whitelist)

	// IP из исключения
	listType, found := trie.Search("10.1.1.50")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType)

	// IP из общей подсети (но не из исключения)
	listType, found = trie.Search("10.2.0.1")
	assert.True(t, found)
	assert.Equal(t, entity.Blacklist, listType)

	// IP из исключения — граница
	listType, found = trie.Search("10.1.1.255")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType)
}

func TestTrie_LongestPrefixMatch_MultipleLevels(t *testing.T) {
	trie := NewIPTrie()

	trie.Insert("10.0.0.0/8", entity.Blacklist)     // всё запрещено
	trie.Insert("10.1.0.0/16", entity.Graylist)      // кроме этого — капча
	trie.Insert("10.1.1.0/24", entity.Whitelist)     // а тут белый список

	// Самое специфичное правило побеждает
	listType, found := trie.Search("10.1.1.100")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType)

	// Второй уровень
	listType, found = trie.Search("10.1.2.100")
	assert.True(t, found)
	assert.Equal(t, entity.Graylist, listType)

	// Самый общий
	listType, found = trie.Search("10.2.0.1")
	assert.True(t, found)
	assert.Equal(t, entity.Blacklist, listType)
}

func TestTrie_MultipleLists(t *testing.T) {
	trie := NewIPTrie()

	trie.Insert("10.0.0.0/8", entity.Blacklist)
	trie.Insert("192.168.0.0/16", entity.Whitelist)
	trie.Insert("172.16.0.0/12", entity.Graylist)
	trie.Insert("8.8.8.8", entity.Whitelist)

	// Проверяем каждый
	listType, found := trie.Search("10.255.255.255")
	assert.True(t, found)
	assert.Equal(t, entity.Blacklist, listType)

	listType, found = trie.Search("192.168.100.1")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType)

	listType, found = trie.Search("172.16.0.1")
	assert.True(t, found)
	assert.Equal(t, entity.Graylist, listType)

	listType, found = trie.Search("8.8.8.8")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType)
}

func TestTrie_NotFound(t *testing.T) {
	trie := NewIPTrie()
	trie.Insert("192.168.1.0/24", entity.Whitelist)

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

	assert.Error(t, trie.Insert("invalid", entity.Whitelist))
	assert.Error(t, trie.Insert("", entity.Whitelist))
	assert.Error(t, trie.Insert("999.999.999.999", entity.Whitelist))
	assert.Error(t, trie.Insert("1.2.3.4.5", entity.Whitelist))
	assert.Error(t, trie.Insert("abc.def.ghi.jkl", entity.Whitelist))
}

func TestTrie_InvalidIP_Search(t *testing.T) {
	trie := NewIPTrie()
	trie.Insert("10.0.0.0/8", entity.Blacklist)

	// Невалидный IP при поиске
	_, found := trie.Search("invalid")
	assert.False(t, found)

	_, found = trie.Search("")
	assert.False(t, found)
}


func TestTrie_OverwriteRule(t *testing.T) {
	trie := NewIPTrie()

	trie.Insert("10.0.0.0/8", entity.Blacklist)
	trie.Insert("10.0.0.0/8", entity.Whitelist) // перезаписывает

	listType, found := trie.Search("10.1.1.1")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType) // последняя запись побеждает
}

func TestTrie_SamePrefixDifferentTypes(t *testing.T) {
	trie := NewIPTrie()

	trie.Insert("192.168.0.0/24", entity.Blacklist)
	trie.Insert("192.168.0.0/24", entity.Whitelist)

	listType, found := trie.Search("192.168.0.1")
	assert.True(t, found)
	assert.Equal(t, entity.Whitelist, listType) // последняя перезаписывает
}

func TestTrie_BoundaryValues(t *testing.T) {
	trie := NewIPTrie()
	trie.Insert("10.0.0.0/24", entity.Whitelist)

	// 10.0.0.0
	assert.True(t, checkFound(trie, "10.0.0.0", entity.Whitelist))
	// 10.0.0.255
	assert.True(t, checkFound(trie, "10.0.0.255", entity.Whitelist))
	// 10.0.1.0 — вне
	assert.False(t, checkFound(trie, "10.0.1.0", entity.Whitelist))
}

func TestTrie_ConcurrentReads(t *testing.T) {
	trie := NewIPTrie()
	trie.Insert("10.0.0.0/8", entity.Blacklist)
	trie.Insert("192.168.0.0/16", entity.Whitelist)

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
				trie.Insert("10.0.0.0/8", entity.Blacklist)
				trie.Insert("192.168.0.0/16", entity.Whitelist)
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
	assert.Equal(t, entity.Blacklist, listType)
}

func TestTrie_InsertRange(t *testing.T) {
	trie := NewIPTrie()

	err := trie.InsertRange("192.168.1.1", "192.168.1.5", entity.Whitelist)
	assert.NoError(t, err)

	assert.True(t, checkFound(trie, "192.168.1.1", entity.Whitelist))
	assert.True(t, checkFound(trie, "192.168.1.3", entity.Whitelist))
	assert.True(t, checkFound(trie, "192.168.1.5", entity.Whitelist))
	assert.False(t, checkFound(trie, "192.168.1.6", entity.Whitelist))
	assert.False(t, checkFound(trie, "192.168.1.0", entity.Whitelist))
}

func TestTrie_InsertRange_InvalidInput(t *testing.T) {
	trie := NewIPTrie()

	err := trie.InsertRange("invalid", "1.2.3.4", entity.Whitelist)
	assert.NoError(t, err)
}

func TestNewIPTrie(t *testing.T) {
	trie := NewIPTrie()
	assert.NotNil(t, trie)
	assert.NotNil(t, trie.root)
}

func checkFound(trie *IPTrie, ip string, expectedType entity.ListType) bool {
	listType, found := trie.Search(ip)
	return found && listType == expectedType
}