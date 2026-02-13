package internal

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// sigV2Signer implements AWS SDK v2 HTTPSignerV4 interface with S3 Signature V2.
// Ceph RGW deployments may require this legacy signature scheme.
type sigV2Signer struct{}

// SignHTTP signs request with S3 Signature V2 and sets Authorization header.
func (s *sigV2Signer) SignHTTP(_ context.Context, credentials aws.Credentials, r *http.Request, _ string, _ string, _ string, signingTime time.Time, _ ...func(*v4.SignerOptions)) error {
	if r == nil || r.URL == nil {
		return fmt.Errorf("cannot sign nil request")
	}
	if credentials.AccessKeyID == "" || credentials.SecretAccessKey == "" {
		return fmt.Errorf("missing access key ID or secret access key")
	}

	if credentials.SessionToken != "" {
		r.Header.Set("x-amz-security-token", credentials.SessionToken)
	}
	if r.Header.Get("Date") == "" && r.Header.Get("x-amz-date") == "" {
		r.Header.Set("Date", signingTime.UTC().Format(http.TimeFormat))
	}

	signature := sigV2Signature(credentials.SecretAccessKey, buildSigV2StringToSign(r, ""))
	r.Header.Set("Authorization", "AWS "+credentials.AccessKeyID+":"+signature)

	return nil
}

func buildSigV2StringToSign(r *http.Request, expires string) string {
	method := http.MethodGet
	contentMD5 := ""
	contentType := ""
	dateOrExpires := expires
	canonicalizedAmzHeaders := ""
	canonicalizedResource := "/"

	if r != nil {
		if r.Method != "" {
			method = r.Method
		}
		contentMD5 = r.Header.Get("Content-MD5")
		contentType = r.Header.Get("Content-Type")

		if dateOrExpires == "" {
			if r.Header.Get("x-amz-date") == "" {
				dateOrExpires = r.Header.Get("Date")
			}
		}

		canonicalizedAmzHeaders = canonicalizeSigV2AmzHeaders(r.Header)
		canonicalizedResource = canonicalizeSigV2Resource(r)
	}

	return strings.Join([]string{
		method,
		contentMD5,
		contentType,
		dateOrExpires,
		canonicalizedAmzHeaders + canonicalizedResource,
	}, "\n")
}

func canonicalizeSigV2AmzHeaders(headers http.Header) string {
	if headers == nil {
		return ""
	}

	amzHeaders := make(map[string][]string)
	for key, values := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(lowerKey, "x-amz-") {
			continue
		}

		normalizedValues := make([]string, 0, len(values))
		for _, value := range values {
			normalizedValues = append(normalizedValues, normalizeSigV2HeaderValue(value))
		}
		amzHeaders[lowerKey] = append(amzHeaders[lowerKey], normalizedValues...)
	}

	if len(amzHeaders) == 0 {
		return ""
	}

	keys := make([]string, 0, len(amzHeaders))
	for key := range amzHeaders {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte(':')
		builder.WriteString(strings.Join(amzHeaders[key], ","))
		builder.WriteByte('\n')
	}

	return builder.String()
}

func canonicalizeSigV2Resource(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "/"
	}

	resource := r.URL.EscapedPath()
	if resource == "" {
		resource = "/"
	}

	query := r.URL.Query()
	subresources := make([]string, 0, len(query))
	for key, values := range query {
		canonicalKey, ok := canonicalSigV2SubresourceKey(key)
		if !ok {
			continue
		}

		if len(values) == 0 {
			subresources = append(subresources, canonicalKey)
			continue
		}

		sortedValues := append([]string(nil), values...)
		sort.Strings(sortedValues)
		for _, value := range sortedValues {
			if value == "" {
				subresources = append(subresources, canonicalKey)
			} else {
				subresources = append(subresources, canonicalKey+"="+value)
			}
		}
	}

	if len(subresources) == 0 {
		return resource
	}

	sort.Strings(subresources)
	return resource + "?" + strings.Join(subresources, "&")
}

func canonicalSigV2SubresourceKey(rawKey string) (string, bool) {
	lowerKey := strings.ToLower(rawKey)
	if strings.HasPrefix(lowerKey, "response-") {
		return lowerKey, true
	}
	canonicalKey, ok := sigV2Subresources[lowerKey]
	return canonicalKey, ok
}

func normalizeSigV2HeaderValue(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func sigV2Signature(secretKey, stringToSign string) string {
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func isSigV2SignatureVersion(version string) bool {
	switch strings.ToLower(strings.TrimSpace(version)) {
	case "s3", "v2", "2", "sigv2", "signaturev2":
		return true
	default:
		return false
	}
}

func escapeS3ObjectKey(objectPath string) string {
	parts := strings.Split(objectPath, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func joinURLPath(basePath string, parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	if trimmed := strings.Trim(basePath, "/"); trimmed != "" {
		segments = append(segments, trimmed)
	}
	for _, part := range parts {
		if trimmed := strings.Trim(part, "/"); trimmed != "" {
			segments = append(segments, trimmed)
		}
	}
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

var sigV2Subresources = map[string]string{
	"acl":            "acl",
	"accelerate":     "accelerate",
	"analytics":      "analytics",
	"cors":           "cors",
	"delete":         "delete",
	"inventory":      "inventory",
	"legal-hold":     "legal-hold",
	"lifecycle":      "lifecycle",
	"location":       "location",
	"logging":        "logging",
	"metrics":        "metrics",
	"notification":   "notification",
	"object-lock":    "object-lock",
	"partnumber":     "partNumber",
	"policy":         "policy",
	"replication":    "replication",
	"requestpayment": "requestPayment",
	"restore":        "restore",
	"retention":      "retention",
	"tagging":        "tagging",
	"torrent":        "torrent",
	"uploadid":       "uploadId",
	"uploads":        "uploads",
	"versionid":      "versionId",
	"versioning":     "versioning",
	"versions":       "versions",
	"website":        "website",
}
