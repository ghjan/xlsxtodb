package utils

import (
	"fmt"
	"os"
	"strings"
)

func Checkerr(err error, msg string) {
	if err != nil {
		fmt.Println(err)
		if msg != "" {
			fmt.Println(msg)
		}
		if strings.Index(err.Error(), "duplicate key value") >= 0 || strings.Index(err.Error(), "a failed transaction") >= 0{
			return
		}
		os.Exit(-1)
	}
}
