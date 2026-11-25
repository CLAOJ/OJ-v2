// Package storage puts uploaded media on Cloudflare R2 (S3 API) when R2_* is
// configured. Two buckets: a public one for images (served from R2_PUBLIC_URL)
// and a private one for statement PDFs (read only through short-lived signed
// links, after the API has checked the viewer may see the problem).
// Unconfigured, every function reports disabled and callers keep local disk.
package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/CLAOJ/claoj/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// PDFLinkTTL bounds how long a leaked statement link keeps working.
const PDFLinkTTL = 5 * time.Minute

var (
	once   sync.Once
	client *s3.Client
)

func cfg() config.R2Config { return config.C.R2 }

func s3c() *s3.Client {
	once.Do(func() {
		c := cfg()
		if c.AccountID == "" || c.AccessKeyID == "" || c.SecretAccessKey == "" {
			return
		}
		client = s3.New(s3.Options{
			Region:       "auto",
			BaseEndpoint: aws.String("https://" + c.AccountID + ".r2.cloudflarestorage.com"),
			Credentials:  credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, ""),
		})
	})
	return client
}

// PublicEnabled: images go to the public bucket.
func PublicEnabled() bool { return s3c() != nil && cfg().Bucket != "" && cfg().PublicURL != "" }

// PDFEnabled: statement PDFs go to the private bucket.
func PDFEnabled() bool { return s3c() != nil && cfg().PDFBucket != "" }

// PutPublic uploads to the public bucket and returns the object's public URL.
func PutPublic(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	if !PublicEnabled() {
		return "", errors.New("r2 public bucket not configured")
	}
	_, err := s3c().PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(cfg().Bucket),
		Key:          aws.String(key),
		Body:         body,
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(cfg().PublicURL, "/") + "/" + key, nil
}

// PutPDF uploads a statement PDF to the private bucket.
func PutPDF(ctx context.Context, key string, body io.Reader) error {
	if !PDFEnabled() {
		return errors.New("r2 pdf bucket not configured")
	}
	_, err := s3c().PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg().PDFBucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String("application/pdf"),
	})
	return err
}

// DeletePDF removes a statement PDF; a missing object is not an error.
func DeletePDF(ctx context.Context, key string) error {
	if !PDFEnabled() {
		return nil
	}
	_, err := s3c().DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(cfg().PDFBucket), Key: aws.String(key)})
	return err
}

// PDFLink returns a signed GET link for key, or ok=false when the object is not
// in the bucket (not migrated yet), so the caller can fall back to local disk.
func PDFLink(ctx context.Context, key, filename string) (link string, ok bool, err error) {
	if !PDFEnabled() {
		return "", false, nil
	}
	_, err = s3c().HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(cfg().PDFBucket), Key: aws.String(key)})
	if err != nil {
		var nf *types.NotFound
		var nsk *types.NoSuchKey
		if errors.As(err, &nf) || errors.As(err, &nsk) {
			return "", false, nil
		}
		return "", false, err
	}
	req, err := s3.NewPresignClient(s3c()).PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(cfg().PDFBucket),
		Key:                        aws.String(key),
		ResponseContentType:        aws.String("application/pdf"),
		ResponseContentDisposition: aws.String(`inline; filename="` + filename + `"`),
	}, s3.WithPresignExpires(PDFLinkTTL))
	if err != nil {
		return "", false, err
	}
	return req.URL, true, nil
}

// PDFKey maps a problem's stored pdf_url to its object key: v1-migrated
// "/pdf/<uuid>.pdf" -> "pdf/<uuid>.pdf" (same layout as the v1 media dir, so a
// plain rclone copy of it lines up); v2-native "statement.pdf" ->
// "problems/<code>/statement.pdf".
func PDFKey(pdfURL, code string) string {
	if strings.HasPrefix(pdfURL, "/") {
		return strings.TrimPrefix(pdfURL, "/")
	}
	return "problems/" + code + "/" + pdfURL
}
