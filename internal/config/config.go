package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Platform             string `mapstructure:"PLATFORM"`
	Keywords             string `mapstructure:"KEYWORDS"`
	LoginType            string `mapstructure:"LOGIN_TYPE"`
	LoginPhone           string `mapstructure:"LOGIN_PHONE"`
	LoginWaitTimeoutSec  int    `mapstructure:"LOGIN_WAIT_TIMEOUT_SEC"`
	Cookies              string `mapstructure:"COOKIES"`
	CrawlerType          string `mapstructure:"CRAWLER_TYPE"`
	DataDir              string `mapstructure:"DATA_DIR"`
	StoreBackend         string `mapstructure:"STORE_BACKEND"`
	SQLitePath           string `mapstructure:"SQLITE_PATH"`
	LogLevel             string `mapstructure:"LOG_LEVEL"`
	LogFormat            string `mapstructure:"LOG_FORMAT"`
	HttpTimeoutSec       int    `mapstructure:"HTTP_TIMEOUT_SEC"`
	HttpRetryCount       int    `mapstructure:"HTTP_RETRY_COUNT"`
	HttpRetryBaseDelayMs int    `mapstructure:"HTTP_RETRY_BASE_DELAY_MS"`
	HttpRetryMaxDelayMs  int    `mapstructure:"HTTP_RETRY_MAX_DELAY_MS"`
	EnableIPProxy        bool   `mapstructure:"ENABLE_IP_PROXY"`
	IPProxyPoolCount     int    `mapstructure:"IP_PROXY_POOL_COUNT"`
	IPProxyProviderName  string `mapstructure:"IP_PROXY_PROVIDER_NAME"`
	Headless             bool   `mapstructure:"HEADLESS"`
	SaveLoginState       bool   `mapstructure:"SAVE_LOGIN_STATE"`
	EnableCDPMode        bool   `mapstructure:"ENABLE_CDP_MODE"`
	CDPDebugPort         int    `mapstructure:"CDP_DEBUG_PORT"`
	CustomBrowserPath    string `mapstructure:"CUSTOM_BROWSER_PATH"`
	CDPHeadless          bool   `mapstructure:"CDP_HEADLESS"`
	BrowserLaunchTimeout int    `mapstructure:"BROWSER_LAUNCH_TIMEOUT"`
	AutoCloseBrowser     bool   `mapstructure:"AUTO_CLOSE_BROWSER"`
	SaveDataOption       string `mapstructure:"SAVE_DATA_OPTION"`
	UserDataDir          string `mapstructure:"USER_DATA_DIR"`
	StartPage            int    `mapstructure:"START_PAGE"`
	CrawlerMaxNotesCount int    `mapstructure:"CRAWLER_MAX_NOTES_COUNT"`
	MaxConcurrencyNum    int    `mapstructure:"MAX_CONCURRENCY_NUM"`
	EnableGetMedias      bool   `mapstructure:"ENABLE_GET_MEDIAS"`
	EnableGetComments    bool   `mapstructure:"ENABLE_GET_COMMENTS"`
	CrawlerMaxComments   int    `mapstructure:"CRAWLER_MAX_COMMENTS_COUNT_SINGLENOTES"`
	EnableGetSubComments bool   `mapstructure:"ENABLE_GET_SUB_COMMENTS"`
	CrawlerMaxSleepSec   int    `mapstructure:"CRAWLER_MAX_SLEEP_SEC"`

	// XHS Specific
	SortType             string   `mapstructure:"SORT_TYPE"`
	XhsSpecifiedNoteUrls []string `mapstructure:"XHS_SPECIFIED_NOTE_URL_LIST"`
	XhsCreatorIdList     []string `mapstructure:"XHS_CREATOR_ID_LIST"`

	// Douyin Specific
	DouyinSpecifiedNoteUrls []string `mapstructure:"DY_SPECIFIED_NOTE_URL_LIST"`
	DouyinCreatorIdList     []string `mapstructure:"DY_CREATOR_ID_LIST"`

	// Bilibili Specific
	BiliSpecifiedVideoUrls []string `mapstructure:"BILI_SPECIFIED_VIDEO_URL_LIST"`

	// Weibo Specific
	WBSpecifiedNoteUrls []string `mapstructure:"WB_SPECIFIED_NOTE_URL_LIST"`
	WBCreatorIdList     []string `mapstructure:"WB_CREATOR_ID_LIST"`
	WBSearchType        string   `mapstructure:"WB_SEARCH_TYPE"`

	// Tieba Specific
	TiebaSpecifiedNoteUrls []string `mapstructure:"TIEBA_SPECIFIED_NOTE_URL_LIST"`

	// Zhihu Specific
	ZhihuSpecifiedNoteUrls []string `mapstructure:"ZHIHU_SPECIFIED_NOTE_URL_LIST"`

	// Kuaishou Specific
	KuaishouSpecifiedNoteUrls []string `mapstructure:"KS_SPECIFIED_NOTE_URL_LIST"`
}

var AppConfig Config

func LoadConfig(path string) error {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml") // Using yaml for go config, though python uses .py

	// Set defaults matching python config
	viper.SetDefault("PLATFORM", "xhs")
	viper.SetDefault("KEYWORDS", "编程副业,编程兼职")
	viper.SetDefault("LOGIN_TYPE", "qrcode")
	viper.SetDefault("LOGIN_PHONE", "")
	viper.SetDefault("LOGIN_WAIT_TIMEOUT_SEC", 120)
	viper.SetDefault("CRAWLER_TYPE", "search")
	viper.SetDefault("DATA_DIR", "data")
	viper.SetDefault("STORE_BACKEND", "file")
	viper.SetDefault("SQLITE_PATH", "data/media_crawler.db")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")
	viper.SetDefault("HTTP_TIMEOUT_SEC", 60)
	viper.SetDefault("HTTP_RETRY_COUNT", 3)
	viper.SetDefault("HTTP_RETRY_BASE_DELAY_MS", 500)
	viper.SetDefault("HTTP_RETRY_MAX_DELAY_MS", 4000)
	viper.SetDefault("ENABLE_IP_PROXY", false)
	viper.SetDefault("IP_PROXY_POOL_COUNT", 2)
	viper.SetDefault("IP_PROXY_PROVIDER_NAME", "kuaidaili")
	viper.SetDefault("HEADLESS", false)
	viper.SetDefault("SAVE_LOGIN_STATE", true)
	viper.SetDefault("ENABLE_CDP_MODE", true)
	viper.SetDefault("CDP_DEBUG_PORT", 9222)
	viper.SetDefault("CUSTOM_BROWSER_PATH", "")
	viper.SetDefault("CDP_HEADLESS", false)
	viper.SetDefault("BROWSER_LAUNCH_TIMEOUT", 60)
	viper.SetDefault("AUTO_CLOSE_BROWSER", true)
	viper.SetDefault("SAVE_DATA_OPTION", "json")
	viper.SetDefault("START_PAGE", 1)
	viper.SetDefault("CRAWLER_MAX_NOTES_COUNT", 15)
	viper.SetDefault("MAX_CONCURRENCY_NUM", 1)
	viper.SetDefault("ENABLE_GET_MEDIAS", false)
	viper.SetDefault("ENABLE_GET_COMMENTS", true)
	viper.SetDefault("CRAWLER_MAX_COMMENTS_COUNT_SINGLENOTES", 10)
	viper.SetDefault("CRAWLER_MAX_SLEEP_SEC", 2)
	viper.SetDefault("SORT_TYPE", "popularity_descending")
	viper.SetDefault("TIEBA_SPECIFIED_NOTE_URL_LIST", []string{})
	viper.SetDefault("ZHIHU_SPECIFIED_NOTE_URL_LIST", []string{})
	viper.SetDefault("KS_SPECIFIED_NOTE_URL_LIST", []string{})
	viper.SetDefault("WB_CREATOR_ID_LIST", []string{})
	viper.SetDefault("WB_SEARCH_TYPE", "1")

	viper.SetEnvPrefix("MEDIA_CRAWLER")
	viper.AutomaticEnv()
	viper.RegisterAlias("ENABLE_GET_MEIDAS", "ENABLE_GET_MEDIAS")

	// If no config file found, just use defaults/env
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return viper.Unmarshal(&AppConfig)
}

func GetKeywords() []string {
	if AppConfig.Keywords == "" {
		return []string{}
	}
	return strings.Split(AppConfig.Keywords, ",")
}
