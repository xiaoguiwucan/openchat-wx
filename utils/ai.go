package utils

import (
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/xiaoguiwucan/openchat-wx/vars"
)

func TrimAt(content string) string {
	// 去除@开头的触发词
	re := regexp.MustCompile(vars.TrimAtRegexp)
	return re.ReplaceAllString(content, "")
}

func TrimAITriggerWord(content, aiTriggerWord string) string {
	// 去除固定AI触发词
	re := regexp.MustCompile("^" + regexp.QuoteMeta(aiTriggerWord) + `[\s，,：:]*`)
	return re.ReplaceAllString(content, "")
}

func TrimAITriggerAll(content, aiTriggerWord string) string {
	return TrimAITriggerWord(TrimAt(content), aiTriggerWord)
}

// NormalizeAIBaseURL normalizes an OpenAI-compatible endpoint and makes
// host loopback addresses reachable from a Docker container.
func NormalizeAIBaseURL(baseURL string) string {
	inContainer, gateway := aiRuntimeNetworkSettings()
	return NormalizeAIBaseURLForRuntime(baseURL, inContainer, gateway)
}

func NormalizeAIBaseURLForRuntime(baseURL string, inContainer bool, gateway string) string {
	baseURL = strings.TrimRight(RewriteLoopbackURLForRuntime(baseURL, inContainer, gateway), "/")
	if baseURL == "" {
		return ""
	}
	versionRegex := regexp.MustCompile(`/v\d+$`)
	if !versionRegex.MatchString(baseURL) {
		baseURL += "/v1"
	}
	return baseURL
}

// RewriteLoopbackURL makes a URL returned by a host-side provider reachable
// from the client container without modifying its path.
func RewriteLoopbackURL(rawURL string) string {
	inContainer, gateway := aiRuntimeNetworkSettings()
	return RewriteLoopbackURLForRuntime(rawURL, inContainer, gateway)
}

func RewriteLoopbackURLForRuntime(rawURL string, inContainer bool, gateway string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" && inContainer {
		hostname := strings.ToLower(parsed.Hostname())
		if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
			if strings.TrimSpace(gateway) == "" {
				gateway = "host.docker.internal"
			}
			if port := parsed.Port(); port != "" {
				parsed.Host = gateway + ":" + port
			} else {
				parsed.Host = gateway
			}
			rawURL = parsed.String()
		}
	}
	return rawURL
}

func aiRuntimeNetworkSettings() (bool, string) {
	inContainer := false
	if value := strings.TrimSpace(os.Getenv("OPENCHAT_DOCKER")); value != "" {
		inContainer = value == "1" || strings.EqualFold(value, "true")
	} else if _, err := os.Stat("/.dockerenv"); err == nil {
		inContainer = true
	}
	gateway := strings.TrimSpace(os.Getenv("OPENCHAT_HOST_GATEWAY"))
	if gateway == "" {
		gateway = "host.docker.internal"
	}
	return inContainer, gateway
}
