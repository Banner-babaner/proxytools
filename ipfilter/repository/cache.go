package repository

import (

	"ipfilter/entity"
)

type IPCache interface {
	Get(ip string) (entity.ListType, bool, bool)
	Set(ip string, listType entity.ListType, hasRule bool)
	Remove(ip string)
}

