package infrastructure

import (
	"github.com/Banner-babaner/proxytools/config"
	"github.com/Banner-babaner/proxytools/ipfilter/entity"
	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	path    string
	watcher *fsnotify.Watcher
	done chan struct{}
}

func NewFileWatcher(path string) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &FileWatcher{path: path, watcher: w, done: make(chan struct{})}, nil
}

func (fw *FileWatcher) Watch(callback func(entity.ListsConfig)) {
	fw.watcher.Add(fw.path)

	go func() {
		for {
			select {
			case event := <-fw.watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					cfg, _ := config.Load(fw.path)
					lists := entity.ListsConfig{
						Whitelist: cfg.IPFilter.Lists.Whitelist,
						Blacklist: cfg.IPFilter.Lists.Blacklist,
						Graylist:  cfg.IPFilter.Lists.Graylist,
					}
					callback(lists)
				}
			case <-fw.done:
			return
			}
		}
	}()
}

func (fw *FileWatcher) Stop() {
	close(fw.done)
	fw.watcher.Close()
}