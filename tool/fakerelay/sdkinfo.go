package main

import (
	"bytes"
	"regexp"
)

var t = regexp.MustCompile(`sentry_client=(\S+)`)

type SDKInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func ParseSDKInfo(request []byte) SDKInfo {
	var sdkInfo SDKInfo
	match := t.FindSubmatch(request)
	if len(match) < 2 {
		return sdkInfo
	}
	s := bytes.SplitN(match[1], []byte("/"), 2)
	sdkInfo.Name = string(bytes.ReplaceAll(s[0], []byte("-"), []byte(".")))
	if len(s) > 1 {
		sdkInfo.Version = string(bytes.ReplaceAll(s[1], []byte(","), []byte("")))
	}
	return sdkInfo
}
