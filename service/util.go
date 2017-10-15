package service

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var logPrintf = log.Printf
var dockerApiVersion string = "v1.22"

func getSenderAddressesFromEnvVars(catchAllType, senderType, altSenderType string) (createServiceAddr, removeServiceAddr []string) {
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

func getServiceParams(s *SwarmService) map[string]string {
	params := map[string]string{}
	if _, ok := s.Spec.Labels[os.Getenv("DF_NOTIFY_LABEL")]; ok {
		params["serviceName"] = s.Spec.Name
		for k, v := range s.Spec.Labels {
			if strings.HasPrefix(k, "com.df") && k != os.Getenv("DF_NOTIFY_LABEL") {
				params[strings.TrimPrefix(k, "com.df.")] = v
			}
		}
		if s.Service.Spec.Mode.Replicated != nil {
			params["replicas"] = fmt.Sprintf("%d", *s.Service.Spec.Mode.Replicated.Replicas)
		}
	}
	return params
}
