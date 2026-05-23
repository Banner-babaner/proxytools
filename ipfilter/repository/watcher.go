package repository

import "ipfilter/entity"

type ConfigWatcher interface {
	Watch(callback func(entity.ListsConfig))
	Stop()
}