package repository

import (
	"github.com/Banner-babaner/proxytools/ipfilter/entity"
)

type IPListRepository interface {
	Search(ip string) (entity.ListType, bool)
	Insert(cidr string, listType entity.ListType) error
	InsertRange(startIP string, endIP string, listType entity.ListType) error
	
}
