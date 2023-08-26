package main

import (
	"os"
	"strconv"
)

func isDevelopment() bool {
	isDev, _ := strconv.ParseBool(os.Getenv("IS_DEVELOPMENT"))
	return isDev
}
