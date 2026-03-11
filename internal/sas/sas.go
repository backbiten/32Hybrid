// Package sas provides Azure Blob Storage Shared Access Signature (SAS) URL
// generation using a storage account name and key.
//
// This implementation follows the Azure Blob Service SAS v2020-12-06
// specification and signs with HMAC-SHA256 using the account key directly —
// no additional SDK dependency is required.
//
// Reference:
// https://learn.microsoft.com/en-us/rest/api/storageservices/create-service-sas
package sas

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// serviceVersion is the storage REST API version used in SAS tokens.
	serviceVersion = "2020-12-06"

	// blobEndpointFmt is the Azure Blob service endpoint template.
	blobEndpointFmt = "https://%s.blob.core.windows.net"
)

// Params holds the inputs needed to mint a single SAS URL.
type Params struct {
	// AccountName is the Azure Storage account name.
	AccountName string

	// AccountKey is the base64-encoded primary or secondary storage account key.
	AccountKey string

	// Container is the blob container name.
	Container string

	// BlobName is the path within the container (e.g. "uploads/abc123/app.exe").
	BlobName string

	// Permissions is the SAS permission string, e.g. "r" (read) or "w" (write).
	// See: https://learn.microsoft.com/en-us/rest/api/storageservices/create-service-sas#permissions-for-a-blob
	Permissions string

	// Start is the optional validity start time. Zero means "now minus 5 min"
	// (clock-skew buffer).
	Start time.Time

	// Expiry is the token expiration time.
	Expiry time.Time
}

// BlobSASURL mints an Azure Blob SAS URL from an account key.
func BlobSASURL(p Params) (string, error) {
	if p.AccountName == "" {
		return "", fmt.Errorf("sas: account name is required")
	}
	if p.AccountKey == "" {
		return "", fmt.Errorf("sas: account key is required")
	}
	if p.Container == "" {
		return "", fmt.Errorf("sas: container is required")
	}
	if p.BlobName == "" {
		return "", fmt.Errorf("sas: blob name is required")
	}
	if p.Expiry.IsZero() {
		return "", fmt.Errorf("sas: expiry time is required")
	}

	start := p.Start
	if start.IsZero() {
		start = time.Now().UTC().Add(-5 * time.Minute)
	}

	startStr := start.UTC().Format("2006-01-02T15:04:05Z")
	expiryStr := p.Expiry.UTC().Format("2006-01-02T15:04:05Z")

	canonicalizedResource := fmt.Sprintf("/blob/%s/%s/%s",
		p.AccountName, p.Container, p.BlobName)

	// String-to-sign for Blob Service SAS (API version 2020-12-06).
	// Fields that are left empty correspond to optional SAS parameters that
	// we do not use in this MVP implementation.
	stringToSign := strings.Join([]string{
		p.Permissions,  // signedPermissions
		startStr,       // signedStart
		expiryStr,      // signedExpiry
		canonicalizedResource,
		"",             // signedIdentifier
		"",             // signedIP
		"https",        // signedProtocol
		serviceVersion, // signedVersion
		"b",            // signedResource (b = blob)
		"",             // signedSnapshotTime
		"",             // signedEncryptionScope
		"",             // rscc (Cache-Control)
		"",             // rscd (Content-Disposition)
		"",             // rsce (Content-Encoding)
		"",             // rscl (Content-Language)
		"",             // rsct (Content-Type)
	}, "\n")

	sig, err := signHMACSHA256(p.AccountKey, stringToSign)
	if err != nil {
		return "", fmt.Errorf("sas: sign: %w", err)
	}

	q := url.Values{}
	q.Set("sv", serviceVersion)
	q.Set("st", startStr)
	q.Set("se", expiryStr)
	q.Set("sr", "b")
	q.Set("sp", p.Permissions)
	q.Set("spr", "https")
	q.Set("sig", sig)

	blobURL := fmt.Sprintf("%s/%s/%s?%s",
		fmt.Sprintf(blobEndpointFmt, p.AccountName),
		p.Container,
		url.PathEscape(p.BlobName),
		q.Encode(),
	)
	return blobURL, nil
}

// signHMACSHA256 decodes the base64-encoded account key and signs message
// with HMAC-SHA256, returning the base64-encoded signature.
func signHMACSHA256(accountKeyB64, message string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(accountKeyB64)
	if err != nil {
		return "", fmt.Errorf("base64-decode account key: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}
