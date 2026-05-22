package repository

import "github.com/Banner-babaner/proxytools/ipfilter/entity"

type ConfigWatcher interface {
	Watch(callback func(entity.ListsConfig))
	Stop()
}