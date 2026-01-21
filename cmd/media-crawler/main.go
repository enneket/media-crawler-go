package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/platform/xhs"
	"os"
)

func main() {
	configPath := flag.String("config", ".", "path to config file")
	flag.Parse()

	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Starting crawler for platform: %s\n", config.AppConfig.Platform)

	ctx := context.Background()
	var err error

	switch config.AppConfig.Platform {
	case "xhs":
		crawler := xhs.NewCrawler()
		err = crawler.Start(ctx)
	default:
		log.Fatalf("Platform %s not implemented", config.AppConfig.Platform)
	}

	if err != nil {
		log.Printf("Crawler failed: %v", err)
		os.Exit(1)
	}
	
	fmt.Println("Crawler finished successfully")
}
