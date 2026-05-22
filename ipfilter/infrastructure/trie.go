package infrastructure

import (
	"fmt"
	"net"
	"sync"
    "github.com/Banner-babaner/proxytools/ipfilter/entity"
)

type TrieNode struct {
    Left   *TrieNode
    Right  *TrieNode
    Type   entity.ListType 
    HasRule bool
}


type IPTrie struct {
    mu   sync.RWMutex
    root *TrieNode
}

func NewIPTrie() *IPTrie {
    return &IPTrie{
        root: &TrieNode{},
    }
}




func (t *IPTrie) Insert(cidr string, listType entity.ListType) error {
    _, ipNet, err := net.ParseCIDR(cidr)
    if err != nil {
        // пробуем как одиночный IP
        ip := net.ParseIP(cidr)
        if ip == nil {
            return err
        }
        ipNet = &net.IPNet{
            IP:   ip,
            Mask: net.CIDRMask(32, 32),
        }
        if ip.To4() == nil {
            ipNet.Mask = net.CIDRMask(128, 128)
        }
    }
    
    t.mu.Lock()
    defer t.mu.Unlock()
    
    ones, _ := ipNet.Mask.Size()
    ip := ipNet.IP.To4()
    if ip == nil {
        ip = ipNet.IP.To16()
    }
    
    node := t.root
    for i := 0; i < ones; i++ {
        byteIdx := i / 8
        bitIdx := 7 - (i % 8)
        
        if ip[byteIdx]&(1<<bitIdx) != 0 {
            if node.Right == nil {
                node.Right = &TrieNode{}
            }
            node = node.Right
        } else {
            if node.Left == nil {
                node.Left = &TrieNode{}
            }
            node = node.Left
        }
    }
    
    node.HasRule = true
    node.Type = listType
    return nil
}


func (t *IPTrie) Search(ipStr string) (entity.ListType, bool) {
    ip := net.ParseIP(ipStr)
    if ip == nil {
        return 0, false
    }
    
    ip4 := ip.To4()
    if ip4 == nil {
        return 0, false
    }
    
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    node := t.root
    var lastMatch entity.ListType
    found := false
    
    for i := 0; i < 32; i++ {
        if node.HasRule {
            lastMatch = node.Type
            found = true
        }
        
        byteIdx := i / 8
        bitIdx := 7 - (i % 8)
        
        if ip4[byteIdx]&(1<<bitIdx) != 0 {
            if node.Right == nil {
                break
            }
            node = node.Right
        } else {
            if node.Left == nil {
                break
            }
            node = node.Left
        }
    }
    
    if node.HasRule {
        lastMatch = node.Type
        found = true
    }
    
    return lastMatch, found
}


func (t *IPTrie) InsertRange(startIP, endIP string, listType entity.ListType) error {
    start := net.ParseIP(startIP)
    end := net.ParseIP(endIP)
    if start == nil || end == nil {
        return fmt.Errorf("Invalid IP-range")
    }
    

    start4 := start.To4()
    end4 := end.To4()
    
    for ip := start4; !ip.Equal(end4); incrementIP(ip) {
        t.Insert(ip.String()+"/32", listType)
    }
    t.Insert(end4.String()+"/32", listType)
    
    return nil
}

func incrementIP(ip net.IP) {
    for j := len(ip) - 1; j >= 0; j-- {
        ip[j]++
        if ip[j] > 0 {
            break
        }
    }
}