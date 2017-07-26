package service

import (
	"log"
	"os"
	"strings"
	"fmt"
)

var logPrintf = log.Printf
var dockerApiVersion string = "v1.22"

func getSenderAddressesFromEnvVars(catchAllType, senderType, altSenderType string) (createServiceAddr, removeServiceAddr[]string) {
	catchAllVarName := fmt.Sprintf("DF_%s_URL", strings.ToUpper(catchAllType))
	createVarName := fmt.Sprintf("DF_%s_CREATE_SERVICE_URL", strings.ToUpper(senderType))
	createAltVarName := fmt.Sprintf("DF_%s_CREATE_SERVICE_URL", strings.ToUpper(altSenderType))
	removeVarName := fmt.Sprintf("DF_%s_REMOVE_SERVICE_URL", strings.ToUpper(senderType))
	removeAltVarName := fmt.Sprintf("DF_%s_REMOVE_SERVICE_URL", strings.ToUpper(altSenderType))
	if len(os.Getenv(createVarName)) > 0 {
		createServiceAddr = strings.Split(os.Getenv(createVarName), ",")
	} else if len(os.Getenv(createAltVarName)) > 0 {
		createServiceAddr = strings.Split(os.Getenv(createAltVarName), ",")
	} else {
		createServiceAddr = strings.Split(os.Getenv(catchAllVarName), ",")
	}
	if len(os.Getenv(removeVarName)) > 0 {
		removeServiceAddr = strings.Split(os.Getenv(removeVarName), ",")
	} else if len(os.Getenv(removeAltVarName)) > 0 {
		removeServiceAddr = strings.Split(os.Getenv(removeAltVarName), ",")
	} else {
		removeServiceAddr = strings.Split(os.Getenv(catchAllVarName), ",")
	}
	return createServiceAddr, removeServiceAddr
}