package main

import (
	"context"
	"flag"
	"fmt"
	"media-crawler-go/internal/api"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/platform"
	_ "media-crawler-go/internal/platform/bilibili"
	_ "media-crawler-go/internal/platform/douyin"
	_ "media-crawler-go/internal/platform/weibo"
	_ "media-crawler-go/internal/platform/xhs"
	"net/http"
	"os"
)

func main() {
	configPath := flag.String("config", ".", "path to config file")
	apiMode := flag.Bool("api", false, "start api server")
	apiAddr := flag.String("addr", ":8080", "api server address")
	flag.Parse()

	if err := config.LoadConfig(*configPath); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	logger.InitFromConfig()

	if *apiMode {
		srv := api.NewServer(nil)
		logger.Info("starting api server", "addr", *apiAddr)
		if err := http.ListenAndServe(*apiAddr, srv.Handler()); err != nil {
			logger.Error("api server failed", "err", err)
			os.Exit(1)
		}
		return
	}

	logger.Info("starting crawler", "platform", config.AppConfig.Platform)

	c, err := platform.New(config.AppConfig.Platform)
	if err != nil {
		logger.Error("crawler init failed", "err", err)
		os.Exit(1)
	}
	err = c.Start(context.Background())

	if err != nil {
		logger.Error("crawler failed", "err", err)
		os.Exit(1)
	}

	logger.Info("crawler finished successfully")
}
