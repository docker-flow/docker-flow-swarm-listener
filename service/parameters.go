package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// GetNodeMiniCreateParameters converts `NodeMini` into parameters
func GetNodeMiniCreateParameters(node NodeMini) map[string]string {
	params := map[string]string{}

	for k, v := range node.EngineLabels {
		if !strings.HasPrefix(k, "com.df.") {
			continue
		}
		key := strings.TrimPrefix(k, "com.df.")
		if len(key) > 0 {
			params[key] = v
		}
	}

	for k, v := range node.NodeLabels {
		if !strings.HasPrefix(k, "com.df.") {
			continue
		}
		key := strings.TrimPrefix(k, "com.df.")
		if len(key) > 0 {
			params[key] = v
		}
	}

	params["id"] = node.ID
	params["hostname"] = node.Hostname
	params["address"] = node.Addr
	params["versionIndex"] = fmt.Sprintf("%d", node.VersionIndex)
	params["state"] = string(node.State)
	params["role"] = string(node.Role)
	params["availability"] = string(node.Availability)
	return params
}

// GetSwarmServiceMiniCreateParameters converts `SwarmServiceMini` into parameters
func GetSwarmServiceMiniCreateParameters(ssm SwarmServiceMini) map[string]string {
	params := map[string]string{}
	for k, v := range ssm.Labels {
		if !strings.HasPrefix(k, "com.df.") {
			continue
		}
		key := strings.TrimPrefix(k, "com.df.")
		if len(key) > 0 {
			params[key] = v
		}
	}
	serviceName := ssm.Name
	stackName := ssm.Labels["com.docker.stack.namespace"]
	if len(stackName) > 0 &&
		strings.EqualFold(ssm.Labels["com.df.shortName"], "true") {
		serviceName = strings.TrimPrefix(serviceName, stackName+"_")
	}
	params["serviceName"] = serviceName

	if !ssm.Global {
		params["replicas"] = fmt.Sprintf("%d", ssm.Replicas)
	}

	if _, ok := params["distribute"]; !ok {
		params["distribute"] = "true"
	}

	if ssm.NodeInfo != nil {
		b, err := json.Marshal(ssm.NodeInfo)
		if err == nil {
			params["nodeInfo"] = string(b)
		}
	}

	return params
}

// GetNodeMiniRemoveParameters converts `NodeMini` into remove parameters
func GetNodeMiniRemoveParameters(node NodeMini) map[string]string {
	params := map[string]string{}
	params["id"] = node.ID
	params["hostname"] = node.Hostname
	params["address"] = node.Addr

	return params
}

// GetSwarmServiceMiniRemoveParameters converts `SwarmServiceMini` into remove parameters
func GetSwarmServiceMiniRemoveParameters(ssm SwarmServiceMini) map[string]string {
	params := map[string]string{}
	for k, v := range ssm.Labels {
		if !strings.HasPrefix(k, "com.df.") {
			continue
		}
		key := strings.TrimPrefix(k, "com.df.")
		if len(key) > 0 {
			params[key] = v
		}
	}
	serviceName := ssm.Name
	stackName := ssm.Labels["com.docker.stack.namespace"]
	if len(stackName) > 0 &&
		strings.EqualFold(ssm.Labels["com.df.shortName"], "true") {
		serviceName = strings.TrimPrefix(serviceName, stackName+"_")
	}
	params["serviceName"] = serviceName

	if v, ok := ssm.Labels["com.df.distribute"]; ok {
		params["distribute"] = v
	}

	if _, ok := params["distribute"]; !ok {
		params["distribute"] = "true"
	}
	return params
}

// ConvertMapStringStringToURLValues converts params to `url.Values`
func ConvertMapStringStringToURLValues(params map[string]string) url.Values {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values
}
