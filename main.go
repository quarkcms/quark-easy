package main

import (
	"io"
	"log"
	"os"

	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/quarkcloudio/quark-go/v3"
	adminCoreService "github.com/quarkcloudio/quark-go/v3/app/admin"
	miniappCoreService "github.com/quarkcloudio/quark-go/v3/app/miniapp"
	adminModule "github.com/quarkcloudio/quark-go/v3/template/admin"
	"github.com/quarkcloudio/quark-go/v3/utils/file"
	"github.com/quarkcloudio/quark-go/v3/utils/rand"
	"github.com/quarkcloudio/quark-smart/v2/config"
	"github.com/quarkcloudio/quark-smart/v2/database"
	adminEngineService "github.com/quarkcloudio/quark-smart/v2/internal/app/admin/engine"
	toolEngineService "github.com/quarkcloudio/quark-smart/v2/internal/app/tool/engine"
	"github.com/quarkcloudio/quark-smart/v2/internal/middleware"
	"github.com/quarkcloudio/quark-smart/v2/internal/router"
	"github.com/quarkcloudio/quark-smart/v2/pkg/env"
	"github.com/quarkcloudio/quark-smart/v2/pkg/template"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {

	// 服务
	var providers []interface{}

	// 配置信息
	var (
		appPro     = config.App.Pro
		appKey     = config.App.Key
		dbUser     = config.Mysql.Username
		dbPassword = config.Mysql.Password
		dbHost     = config.Mysql.Host
		dbPort     = config.Mysql.Port
		dbName     = config.Mysql.Database
		dbCharset  = config.Mysql.Charset
	)

	// 如果appKey尚未配置时，自动初始化AppKey
	if appKey == "YOUR_APP_KEY" || appKey == "" {
		appKey = rand.MakeAlphanumeric(50)
		env.Set("APP_KEY", appKey)
	}

	// Redis配置信息
	var redisConfig *quark.RedisConfig

	// 数据库配置信息
	dsn := dbUser + ":" + dbPassword + "@tcp(" + dbHost + ":" + dbPort + ")/" + dbName + "?charset=" + dbCharset + "&parseTime=True&loc=Local"

	// Redis配置信息
	if config.Redis.Host != "" {
		redisConfig = &quark.RedisConfig{
			Host:     config.Redis.Host,
			Password: config.Redis.Password,
			Port:     config.Redis.Port,
			Database: config.Redis.Database,
		}
	}

	// 加载管理后台服务
	providers = append(providers, adminCoreService.Providers...)

	// 加载MiniApp服务
	providers = append(providers, miniappCoreService.Providers...)

	// 加载自定义管理后台服务
	providers = append(providers, adminEngineService.Providers...)

	// 加载自定义工具服务
	providers = append(providers, toolEngineService.Providers...)

	// 配置资源
	getConfig := &quark.Config{
		AppKey: appKey,
		DBConfig: &quark.DBConfig{
			Dialector: mysql.Open(dsn),
			Opts: &gorm.Config{
				Logger: logger.New(log.Default(), logger.Config{
					LogLevel:                  logger.Error, // 打印错误日志
					IgnoreRecordNotFoundError: true,         // 忽略记录未找到错误
				}),
			},
		},
		RedisConfig: redisConfig,
		Providers:   providers,
	}

	// 实例化对象
	b := quark.New(getConfig)

	// WEB根目录
	b.Static("/", config.App.RootPath)

	// 静态文件目录
	b.Static("/static/", config.App.StaticPath)

	// 避免每次重启都构建数据库
	if !file.IsExist("install.lock") {
		// 构建Admin数据库
		adminModule.Install()

		// 构建本项目数据库
		database.Handle()
	}

	// 管理后台中间件
	b.Use(adminModule.Middleware)

	// 本项目中间件
	b.Use(middleware.AppMiddleware)

	// 开启Debug模式
	b.Echo().Debug = config.App.Debug

	// 加载Html模板
	b.Echo().Renderer = template.New(config.App.TemplatePath)

	// 日志中间件
	if config.App.Logger {
		b.Echo().Use(echoMiddleware.Logger())
	}

	// 日志文件位置
	if config.App.LoggerFilePath != "" {
		f, _ := os.OpenFile(config.App.LoggerFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)

		// 记录日志
		b.Echo().Logger.SetOutput(io.MultiWriter(f, os.Stdout))
	}

	// 崩溃后自动恢复
	if config.App.Recover {
		b.Echo().Use(echoMiddleware.Recover())
	}

	// 注册后台路由
	router.AdminRegister(b)

	// 注册Web路由
	router.WebRegister(b)

	// 开启高级功能
	if appPro {
		// 注册MiniApp路由
		router.MiniAppRegister(b)
	}

	// 启动服务
	b.Run(config.App.Host)
}
