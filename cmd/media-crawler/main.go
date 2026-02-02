package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"media-crawler-go/internal/api"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/platform"
	_ "media-crawler-go/internal/platform/douyin"
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
		log.Fatalf("Failed to load config: %v", err)
	}

	if *apiMode {
		srv := api.NewServer(nil)
		fmt.Printf("Starting api server at %s\n", *apiAddr)
		if err := http.ListenAndServe(*apiAddr, srv.Handler()); err != nil {
			log.Printf("Api server failed: %v", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("Starting crawler for platform: %s\n", config.AppConfig.Platform)

	c, err := platform.New(config.AppConfig.Platform)
	if err != nil {
		log.Printf("Crawler init failed: %v", err)
		os.Exit(1)
	}
	err = c.Start(context.Background())

	if err != nil {
		log.Printf("Crawler failed: %v", err)
		os.Exit(1)
	}

	fmt.Println("Crawler finished successfully")
}
