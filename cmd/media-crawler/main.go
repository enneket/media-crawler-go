package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"media-crawler-go/internal/api"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/platform"
	_ "media-crawler-go/internal/platform/bilibili"
	_ "media-crawler-go/internal/platform/douyin"
	_ "media-crawler-go/internal/platform/kuaishou"
	_ "media-crawler-go/internal/platform/tieba"
	_ "media-crawler-go/internal/platform/weibo"
	_ "media-crawler-go/internal/platform/xhs"
	_ "media-crawler-go/internal/platform/zhihu"
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

	r, err := platform.New(config.AppConfig.Platform)
	if err != nil {
		logger.Error("crawler init failed", "err", err)
		os.Exit(1)
	}
	req := crawler.RequestFromConfig(config.AppConfig)
	res, err := r.Run(context.Background(), req)

	if err != nil {
		errorKind := crawler.KindOf(err)
		riskHint := ""
		var ce crawler.Error
		if errors.As(err, &ce) && ce.Kind == crawler.ErrorKindRiskHint {
			riskHint = ce.Hint
		}
		logger.Error("crawler failed", "err", err, "error_kind", errorKind, "risk_hint", riskHint, "platform", res.Platform, "mode", res.Mode, "processed", res.Processed, "succeeded", res.Succeeded, "failed", res.Failed, "failure_kinds", res.FailureKinds)
		os.Exit(1)
	}

	logger.Info("crawler finished successfully", "platform", res.Platform, "mode", res.Mode, "processed", res.Processed, "succeeded", res.Succeeded, "failed", res.Failed, "failure_kinds", res.FailureKinds)
}
