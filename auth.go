package main

import (
	"net/http"

	"math/rand"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// Token property
const (
	TokenValueChars      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ@!<>[]+=?/^~#,%.&{}()abcdefghijklmnopqrstuvwxyz"
	TokenNameSessionKey  = "token_name"
	TokenValueSessionKey = "token_value"
)

// Token - form csrf token.
type Token struct {
	Name  string
	Value string
}

// sessions.Options
func getSessionsOption() *sessions.Options {
	return &sessions.Options{
		Path:     settings.RootPath + settings.BackendURI,
		MaxAge:   0,
		HttpOnly: true,
		Secure:   !isDevelopment(), // 開発環境ではfalse
		SameSite: http.SameSiteStrictMode,
	}
}

// check csrf token
func isValidToken(c echo.Context) bool {
	ses, err := session.Get(settings.SessionName, c)
	if err != nil {
		// todo:logging
		return false
	}
	sesTokenName := ses.Values[TokenNameSessionKey].(string)
	sesTokenValue := ses.Values[TokenValueSessionKey].(string)
	formCsrfTokenValue := c.FormValue(sesTokenName)
	if formCsrfTokenValue == "" {
		return false
	}
	if formCsrfTokenValue != sesTokenValue {
		return false
	}
	return true
}

// csrf token - create/save and return
func getToken(c echo.Context) Token {
	var token Token
	n, v := 64, 128
	name := getChars(n)
	value := getChars(v)
	ses, _ := session.Get(settings.SessionName, c)
	ses.Options = getSessionsOption()
	token = Token{
		Name:  name,
		Value: value,
	}
	ses.Values[TokenNameSessionKey] = name
	ses.Values[TokenValueSessionKey] = value
	err := ses.Save(c.Request(), c.Response())
	if err != nil {
		// todo:logging
		return token
	}
	return token
}

// create token value
func getChars(length int) string {
	rand.Seed(time.Now().UnixNano() + int64(length))
	b := make([]byte, length)
	for i := range b {
		b[i] = TokenValueChars[rand.Int63()%int64(len(TokenValueChars))]
	}
	return string(b)
}

// loggedin success
func saveLoggedinSession(c echo.Context) error {
	ses, _ := session.Get(settings.SessionName, c)
	ses.Options = getSessionsOption()
	ses.Values[settings.LoggedinKey] = settings.LoggedinValue
	err := ses.Save(c.Request(), c.Response())
	if err != nil {
		// todo:logging
		return err
	}
	return nil
}

// check loggedin
func isLoggedin(c echo.Context) bool {
	ses, err := session.Get(settings.SessionName, c)
	if err != nil {
		// todo:logging
		return false
	}
	if val, ok := ses.Values[settings.LoggedinKey]; ok {
		if val.(string) == settings.LoggedinValue {
			return true
		}
	}
	return false
}

// check account
func allowUser(name, password string) bool {
	user := getUser(name)
	if user.Name == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.PassWord), []byte(password))
	if err != nil {
		return false
	}
	return true
}
