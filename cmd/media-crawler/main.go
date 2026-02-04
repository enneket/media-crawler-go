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
	"media-crawler-go/internal/store"
	"net/http"
	"os"
	"strings"
)

type optionalBool struct {
	set   bool
	value bool
}

func (b *optionalBool) String() string {
	if b == nil {
		return ""
	}
	if b.value {
		return "true"
	}
	return "false"
}

func (b *optionalBool) Set(s string) error {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "1", "true", "t", "yes", "y", "on":
		b.value = true
	case "0", "false", "f", "no", "n", "off":
		b.value = false
	default:
		return fmt.Errorf("invalid bool: %s", s)
	}
	b.set = true
	return nil
}

type overrides struct {
	platform      string
	mode          string
	keywords      string
	inputs        string
	startPage     int
	maxNotes      int
	concurrency   int
	cookies       string
	loginType     string
	loginPhone    string
	dataDir       string
	storeBackend  string
	saveData      string
	sqlitePath    string
	mysqlDSN      string
	postgresDSN   string
	mongoURI      string
	mongoDB       string
	enableIPProxy optionalBool
	proxyProvider string
	proxyPoolCnt  int
	proxyList     string
	proxyFile     string
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func applyOverrides(cfg *config.Config, o overrides) {
	if cfg == nil {
		return
	}
	if v := strings.TrimSpace(o.platform); v != "" {
		cfg.Platform = v
	}
	if v := strings.TrimSpace(o.mode); v != "" {
		cfg.CrawlerType = v
	}
	if v := strings.TrimSpace(o.keywords); v != "" {
		cfg.Keywords = v
	}
	if v := strings.TrimSpace(o.cookies); v != "" {
		cfg.Cookies = v
	}
	if v := strings.TrimSpace(o.loginType); v != "" {
		cfg.LoginType = v
	}
	if v := strings.TrimSpace(o.loginPhone); v != "" {
		cfg.LoginPhone = v
	}
	if v := strings.TrimSpace(o.dataDir); v != "" {
		cfg.DataDir = v
	}
	if v := strings.TrimSpace(o.storeBackend); v != "" {
		cfg.StoreBackend = v
	}
	if v := strings.TrimSpace(o.saveData); v != "" {
		cfg.SaveDataOption = v
	}
	if v := strings.TrimSpace(o.sqlitePath); v != "" {
		cfg.SQLitePath = v
	}
	if v := strings.TrimSpace(o.mysqlDSN); v != "" {
		cfg.MySQLDSN = v
	}
	if v := strings.TrimSpace(o.postgresDSN); v != "" {
		cfg.PostgresDSN = v
	}
	if v := strings.TrimSpace(o.mongoURI); v != "" {
		cfg.MongoURI = v
	}
	if v := strings.TrimSpace(o.mongoDB); v != "" {
		cfg.MongoDB = v
	}
	if o.enableIPProxy.set {
		cfg.EnableIPProxy = o.enableIPProxy.value
	}
	if v := strings.TrimSpace(o.proxyProvider); v != "" {
		cfg.IPProxyProviderName = v
	}
	if o.proxyPoolCnt > 0 {
		cfg.IPProxyPoolCount = o.proxyPoolCnt
	}
	if v := strings.TrimSpace(o.proxyList); v != "" {
		cfg.IPProxyList = v
	}
	if v := strings.TrimSpace(o.proxyFile); v != "" {
		cfg.IPProxyFile = v
	}
	if o.startPage > 0 {
		cfg.StartPage = o.startPage
	}
	if o.maxNotes > 0 {
		cfg.CrawlerMaxNotesCount = o.maxNotes
	}
	if o.concurrency > 0 {
		cfg.MaxConcurrencyNum = o.concurrency
	}
	if v := strings.TrimSpace(o.inputs); v != "" {
		items := splitCSV(v)
		platform := strings.ToLower(strings.TrimSpace(cfg.Platform))
		mode := strings.ToLower(strings.TrimSpace(cfg.CrawlerType))
		switch platform {
		case "xhs":
			if mode == "creator" {
				cfg.XhsCreatorIdList = items
			} else {
				cfg.XhsSpecifiedNoteUrls = items
			}
		case "douyin", "dy":
			if mode == "creator" {
				cfg.DouyinCreatorIdList = items
			} else {
				cfg.DouyinSpecifiedNoteUrls = items
			}
		case "bilibili", "bili", "b站", "b":
			if mode == "creator" {
				cfg.BiliCreatorIdList = items
			} else {
				cfg.BiliSpecifiedVideoUrls = items
			}
		case "weibo", "wb", "微博":
			if mode == "creator" {
				cfg.WBCreatorIdList = items
			} else {
				cfg.WBSpecifiedNoteUrls = items
			}
		case "tieba", "tb", "贴吧":
			if mode == "creator" {
				cfg.TiebaCreatorUrlList = items
			} else {
				cfg.TiebaSpecifiedNoteUrls = items
			}
		case "zhihu", "zh", "知乎":
			if mode == "creator" {
				cfg.ZhihuCreatorUrlList = items
			} else {
				cfg.ZhihuSpecifiedNoteUrls = items
			}
		case "kuaishou", "ks", "快手":
			if mode == "creator" {
				cfg.KuaishouCreatorUrlList = items
			} else {
				cfg.KuaishouSpecifiedNoteUrls = items
			}
		}
	}
	config.Normalize(cfg)
}

func registerRunFlags(fs *flag.FlagSet, o *overrides) {
	fs.StringVar(&o.platform, "platform", "", "platform: xhs/douyin/bilibili/weibo/tieba/zhihu/kuaishou")
	fs.StringVar(&o.mode, "mode", "", "mode: search/detail/creator")
	fs.StringVar(&o.mode, "crawler_type", "", "mode: search/detail/creator")
	fs.StringVar(&o.keywords, "keywords", "", "keywords csv")
	fs.StringVar(&o.inputs, "inputs", "", "inputs csv (meaning depends on platform+mode)")
	fs.IntVar(&o.startPage, "start_page", 0, "start page")
	fs.IntVar(&o.maxNotes, "max_notes", 0, "max notes")
	fs.IntVar(&o.concurrency, "concurrency", 0, "max concurrency")
	fs.StringVar(&o.cookies, "cookies", "", "cookie header string")
	fs.StringVar(&o.loginType, "login_type", "", "login type: qrcode/phone/cookie")
	fs.StringVar(&o.loginPhone, "login_phone", "", "login phone")
	fs.StringVar(&o.dataDir, "data_dir", "", "data dir")
	fs.Var(&o.enableIPProxy, "enable_ip_proxy", "enable ip proxy")
	fs.IntVar(&o.proxyPoolCnt, "ip_proxy_pool_count", 0, "ip proxy pool count")
	fs.StringVar(&o.proxyProvider, "ip_proxy_provider_name", "", "ip proxy provider name")
	fs.StringVar(&o.proxyList, "ip_proxy_list", "", "static proxy list csv")
	fs.StringVar(&o.proxyFile, "ip_proxy_file", "", "static proxy file path")
}

func registerStoreFlags(fs *flag.FlagSet, o *overrides) {
	fs.StringVar(&o.storeBackend, "store_backend", "", "store backend: file/sqlite/mysql/postgres/mongodb")
	fs.StringVar(&o.saveData, "save_data_option", "", "save option: json/csv/xlsx/excel")
	fs.StringVar(&o.sqlitePath, "sqlite_path", "", "sqlite db path")
	fs.StringVar(&o.mysqlDSN, "mysql_dsn", "", "mysql dsn")
	fs.StringVar(&o.postgresDSN, "postgres_dsn", "", "postgres dsn")
	fs.StringVar(&o.mongoURI, "mongo_uri", "", "mongo uri")
	fs.StringVar(&o.mongoDB, "mongo_db", "", "mongo db name")
}

func main() {
	var o overrides
	root := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	configPath := root.String("config", ".", "path to config file")
	apiMode := root.Bool("api", false, "start api server")
	apiAddr := root.String("addr", ":8080", "api server address")
	registerRunFlags(root, &o)
	registerStoreFlags(root, &o)
	_ = root.Parse(os.Args[1:])

	args := root.Args()
	subcmd := ""
	if len(args) > 0 {
		subcmd = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}

	switch subcmd {
	case "init-db", "init_db", "initdb":
		initFlags := flag.NewFlagSet("init-db", flag.ExitOnError)
		registerStoreFlags(initFlags, &o)
		_ = initFlags.Parse(args)

		if err := config.LoadConfig(*configPath); err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}
		applyOverrides(&config.AppConfig, o)
		logger.InitFromConfig()
		if err := store.Init(context.Background()); err != nil {
			logger.Error("init db failed", "err", err)
			os.Exit(1)
		}
		logger.Info("init db ok", "store_backend", config.AppConfig.StoreBackend)
		return
	case "run":
		runFlags := flag.NewFlagSet("run", flag.ExitOnError)
		registerRunFlags(runFlags, &o)
		registerStoreFlags(runFlags, &o)
		_ = runFlags.Parse(args)
	case "":
	default:
		if o.inputs == "" {
			rest := append([]string{subcmd}, args...)
			o.inputs = strings.Join(rest, ",")
		}
	}

	if err := config.LoadConfig(*configPath); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	applyOverrides(&config.AppConfig, o)
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
		errorURL := ""
		httpStatus := 0
		var ce crawler.Error
		if errors.As(err, &ce) {
			errorURL = ce.URL
			httpStatus = ce.StatusCode
			if ce.Kind == crawler.ErrorKindRiskHint {
				riskHint = ce.Hint
			}
		}
		logger.Error("crawler failed", "err", err, "error_kind", errorKind, "risk_hint", riskHint, "error_url", errorURL, "http_status", httpStatus, "platform", res.Platform, "mode", res.Mode, "processed", res.Processed, "succeeded", res.Succeeded, "failed", res.Failed, "failure_kinds", res.FailureKinds)
		os.Exit(1)
	}

	logger.Info("crawler finished successfully", "platform", res.Platform, "mode", res.Mode, "processed", res.Processed, "succeeded", res.Succeeded, "failed", res.Failed, "failure_kinds", res.FailureKinds)
}
