package main

import (
	"os"
	"strconv"
)

type Args struct {
	Interval      int
	Retry         int
	RetryInterval int
}

func GetArgs() *Args {
	return &Args{
		Interval:      getValue(5, "DF_INTERVAL"),
		Retry:         getValue(1, "DF_RETRY"),
		RetryInterval: getValue(0, "DF_RETRY_INTERVAL"),
	}
}

func getValue(defValue int, varName string) int {
	value := defValue
	if len(os.Getenv(varName)) > 0 {
		value, _ = strconv.Atoi(os.Getenv(varName))
	}
	return value
}
