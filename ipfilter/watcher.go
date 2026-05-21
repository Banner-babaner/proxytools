
package ipfilter

import (
    "github.com/fsnotify/fsnotify"
    "github.com/Banner-babaner/proxytools/logger"
    "github.com/Banner-babaner/proxytools/config"
    "github.com/spf13/viper"
)


func (fs *FilterService) StartWatcher(configPath string) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        logger.Error().Err(err).Msg("Failed to create file watcher")
        return
    }
    
    go func() {
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                if event.Op&fsnotify.Write == fsnotify.Write {
                    logger.Info().Str("file", event.Name).Msg("Config file changed, reloading")
                    
                    v := viper.New()
                    v.SetConfigFile(configPath)
                    if err := v.ReadInConfig(); err != nil {
                        logger.Error().Err(err).Msg("Failed to reload config")
                        continue
                    }
                    
                    var cfg config.Config
                    if err := v.Unmarshal(&cfg); err != nil {
                        logger.Error().Err(err).Msg("Failed to unmarshal config")
                        continue
                    }
                    
                    fs.loadLists(cfg.IPFilter.Lists)
                }
            case err, ok := <-watcher.Errors:
                if !ok {
                    return
                }
                logger.Error().Err(err).Msg("File watcher error")
            }
        }
    }()
    
    watcher.Add(configPath)
}