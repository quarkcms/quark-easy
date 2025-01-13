package router

import (
	"github.com/quarkcloudio/quark-go/v3"
	"github.com/quarkcloudio/quark-smart/v2/internal/app/miniapp/handler"
	"github.com/quarkcloudio/quark-smart/v2/internal/middleware"
)

// 注册MiniApp路由
func MiniAppRegister(b *quark.Engine) {

	// 不需要认证路由组
	g := b.Group("/api/miniapp")
	g.GET("/index/index", (&handler.Index{}).Index)
	g.POST("/register/index", (&handler.Register{}).Index)
	g.POST("/login/index", (&handler.Login{}).Index)
	g.GET("/login/mock", (&handler.Login{}).Mock)

	// 轮播组
	g.GET("/index/banner", (&handler.Index{}).Banner) // 轮播列表

	// 需要登录认证路由组
	ag := b.Group("/api/miniapp", middleware.MiniAppMiddleware)
	ag.GET("/user/index", (&handler.User{}).Index)
	ag.POST("/user/save", (&handler.User{}).Save)
	ag.POST("/user/delete", (&handler.User{}).Delete)
}
