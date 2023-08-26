package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

func init() {
	initializeData()
}

func main() {
	defer closeConnection()
	e := echo.New()
	// <input type="hidden" name="csrf" value="dfasjkjhl(random文字列)" ～ではなく
	// Phalconのように <input type="hidden" name="jfuioashfg;lsa(random文字列)" value="dfasjkjhl(random文字列)"としたいので非採用
	// random文字列の生成についてはechoに準拠(auth.go参照)
	// ---
	// echo v4.2でSameSite設定がきた https://github.com/labstack/echo/pull/1524/files/8b2c77b1079c17fc9d7b1b420b2c3102c4069d6f
	/*e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "form:csrftoken",
		CookiePath:     settings.RootPath + settings.BackendURI,
		CookieHTTPOnly: true,
		CookieMaxAge:   0,
		CookieName:     "_dct",
		CookieSecure:   !isDevelopment(), // 開発環境ではfalse
		CookieSameSite: http.SameSiteStrictMode,
	}))*/
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	e.Static(settings.RootPath+"files", "./files")
	e.File("/favicon.ico", "files/images/favicon.ico")
	e.Renderer = getTemplateRenderer()
	e.GET(settings.RootPath, indexAction)
	e.GET(settings.RootPath+":entry_code", entryAction)
	e.GET(settings.RootPath+"page/:num", pageAction)
	e.GET(settings.RootPath+"tag/:tagName", tagAction)
	e.GET(settings.RootPath+"error/:code", errorAction)
	e.GET(settings.RootPath+settings.BackendURI, backendLoginAction)
	e.POST(settings.RootPath+settings.BackendURI, authenticationAction)
	e.GET(settings.RootPath+settings.BackendURI+"manager/", managerAction)
	e.GET(settings.RootPath+settings.BackendURI+"manager/api/:param", apiGetAction)
	e.POST(settings.RootPath+settings.BackendURI+"manager/api/:param", apiPostAction)
	e.HTTPErrorHandler = errorHandler
	// start server
	go func() {
		if err := e.Start(settings.HttpdPort); err != nil {
			/*
				TODO: Log全般の実装
			*/
			e.Logger.Info("shutting down the server")
		}
	}()
	// graceful shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
