package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
)

// error handler
func errorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	errorMessage := "server error"
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		errorMessage = he.Message.(string)
	}
	log.Print(errorMessage) // TODO:log周り全体的にちゃんと書く
	c.Redirect(http.StatusFound, settings.RootPath+"error/"+strconv.Itoa(code))
}

// error action
func errorAction(c echo.Context) error {
	code, err := strconv.Atoi(c.Param("code"))
	if err != nil {
		code = http.StatusInternalServerError
	}
	errorMessage := "internal server error"
	if code == http.StatusNotFound {
		errorMessage = "not found"
	} else if code == http.StatusBadRequest {
		errorMessage = "Bad Request"
	}
	return c.Render(code, "error.html", map[string]interface{}{
		"title":         strconv.Itoa(code),
		"error_code":    strconv.Itoa(code),
		"error_message": errorMessage,
		"root_path":     settings.RootPath,
	})
}

// index action
func indexAction(c echo.Context) error {
	entries, next, previous := getEntryList(0)
	if entries == nil {
		return c.Redirect(http.StatusFound, settings.RootPath+"error/404")
	}
	return c.Render(http.StatusOK, "multiple.html", map[string]interface{}{
		"title":     "",
		"root_path": settings.RootPath,
		"entries":   entries,
		"next":      next,
		"previous":  previous,
		"tags":      cacheTagsAll,
	})
}

// entry action
func entryAction(c echo.Context) error {
	entryItem := getEntry(c.Param("entry_code"))
	if entryItem.EntryID < 1 {
		return c.Redirect(http.StatusFound, settings.RootPath+"error/404")
	}
	return c.Render(http.StatusOK, "single.html", map[string]interface{}{
		"title":     entryItem.Title,
		"root_path": settings.RootPath,
		"entry":     entryItem,
		"tags":      cacheTagsAll,
	})
}

// page action
func pageAction(c echo.Context) error {
	num, err := strconv.Atoi(c.Param("num"))
	if err != nil {
		return c.Redirect(http.StatusFound, settings.RootPath+"error/400")
	}
	entries, next, previous := getEntryList(num)
	if entries == nil {
		return c.Redirect(http.StatusFound, settings.RootPath+"error/404")
	}
	return c.Render(http.StatusOK, "multiple.html", map[string]interface{}{
		"title":     "",
		"root_path": settings.RootPath,
		"entries":   entries,
		"next":      next,
		"previous":  previous,
		"tags":      cacheTagsAll,
	})
}

// tag action
func tagAction(c echo.Context) error {
	tagName := c.Param("tagName")
	titleList := getTitleList(tagName)
	if titleList == nil {
		return c.Redirect(http.StatusFound, settings.RootPath+"error/404")
	}
	return c.Render(http.StatusOK, "tag_page.html", map[string]interface{}{
		"title":     "tag : " + tagName,
		"root_path": settings.RootPath,
		"tagName":   tagName,
		"titleList": titleList,
		"tags":      cacheTagsAll,
	})
}

// backend login action
func backendLoginAction(c echo.Context) error {
	errorMessage := ""
	errQuery := c.QueryParam("err")
	if errQuery == "ac" {
		errorMessage = "Invalid username or password."
	} else if errQuery == "csrf" {
		errorMessage = "Invalid csrf token."
	}
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{
		"token":         getToken(c),
		"error_message": errorMessage,
	})
}

// authentication action
func authenticationAction(c echo.Context) error {
	// csrf token check
	if !isValidToken(c) {
		return c.Redirect(http.StatusFound, settings.RootPath+settings.BackendURI+"?err=csrf")
	}
	// loggedin
	if allowUser(c.FormValue("user"), c.FormValue("password")) {
		err := saveLoggedinSession(c)
		if err != nil {
			return c.Redirect(http.StatusFound, settings.RootPath+"error/500")
		}
		return c.Redirect(http.StatusFound, settings.RootPath+settings.BackendURI+"manager/")
	}
	return c.Redirect(http.StatusFound, settings.RootPath+settings.BackendURI+"?err=ac")
}

// manager action
func managerAction(c echo.Context) error {
	if !isLoggedin(c) {
		return c.Redirect(http.StatusFound, settings.RootPath+settings.BackendURI)
	}
	return c.Render(http.StatusOK, "manager.html", map[string]interface{}{})
}

// api get method
func apiGetAction(c echo.Context) error {
	// loggedin check
	if !isLoggedin(c) && !isDevelopment() {
		return c.JSON(http.StatusUnauthorized, 0)
	}
	switch c.Param("param") {
	case "getAllEntries":
		type Res struct {
			Entries []MongoEntries `json:"entries"`
		}
		log.Println("access")
		return c.JSON(http.StatusOK, Res{Entries: getAllEntries()})
	}
	return c.JSON(http.StatusForbidden, 0)
}

// api post method
func apiPostAction(c echo.Context) error {
	// loggedin check
	if !isLoggedin(c) && !isDevelopment() {
		return c.JSON(http.StatusUnauthorized, 0)
	}
	switch c.Param("param") {
	case "uploadImage":
		uploadPath := "./files/images/"
		uploadURI := settings.RootPath + "files/images/"
		type Res struct {
			FilePath string `json:"filePath"`
			Error    string `json:"error"`
		}
		file, err := c.FormFile("image")
		if err != nil {
			return c.JSON(http.StatusOK, Res{Error: err.Error()})
		}
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusOK, Res{Error: err.Error()})
		}
		defer src.Close()
		dst, err := os.Create(uploadPath + file.Filename)
		if err != nil {
			return c.JSON(http.StatusOK, Res{Error: err.Error()})
		}
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			return c.JSON(http.StatusOK, Res{Error: err.Error()})
		}

		return c.JSON(http.StatusOK, Res{FilePath: uploadURI + file.Filename})
	}
	return c.JSON(http.StatusForbidden, 0)
}
