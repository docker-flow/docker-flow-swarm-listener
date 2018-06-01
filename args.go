package main

import (
	"os"
	"strconv"
)

type args struct {
	ServicePollingInterval int
	NodePollingInterval    int
	Retry                  int
	RetryInterval          int
}

func getArgs() *args {
	return &args{
		ServicePollingInterval: getValue(-1, "DF_SERVICE_POLLING_INTERVAL"),
		NodePollingInterval:    getValue(-1, "DF_NODE_POLLING_INTERVAL"),
		Retry:                  getValue(1, "DF_RETRY"),
		RetryInterval:          getValue(0, "DF_RETRY_INTERVAL"),
	}
}

func getValue(defValue int, varName string) int {
	value := defValue
	if len(os.Getenv(varName)) > 0 {
		value, _ = strconv.Atoi(os.Getenv(varName))
	}
	return value
}
