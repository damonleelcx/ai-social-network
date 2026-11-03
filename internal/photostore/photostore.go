// Package photostore holds photo bytes: a private S3 bucket in production, a
// directory in development. Nobody gets a direct URL. The API checks who may see
// a photo and streams it, so losing access is immediate (block, unmatch, delete).
package photostore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type Store interface {
	Put(ctx context.Context, key string, data []byte, mime string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	// Check proves the store is reachable and writable, for /readyz and deploys.
	Check(ctx context.Context) error
}

// ── S3 ──────────────────────────────────────────────────────────────────────

type S3 struct {
	Bucket string
	client *s3.Client
}

// NewS3 uses the default credential chain: the node's IAM role in production.
// endpoint is for an S3-compatible server in development and is usually empty.
func NewS3(ctx context.Context, bucket, region, endpoint string) (*S3, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})
	return &S3{Bucket: bucket, client: client}, nil
}

func (s *S3) Put(ctx context.Context, key string, data []byte, mime string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.Bucket, Key: &key, Body: bytes.NewReader(data), ContentType: &mime,
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	})
	return err
}

func (s *S3) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.Bucket, Key: &key})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(io.LimitReader(out.Body, MaxUpload))
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.Bucket, Key: &key})
	return err
}

func (s *S3) Check(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.Bucket})
	return err
}

// ── Disk (development and tests) ────────────────────────────────────────────

type Disk struct{ Dir string }

func (d *Disk) path(key string) (string, error) {
	if strings.Contains(key, "..") {
		return "", errors.New("bad photo key")
	}
	return filepath.Join(d.Dir, filepath.FromSlash(key)), nil
}

func (d *Disk) Put(_ context.Context, key string, data []byte, _ string) error {
	p, err := d.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func (d *Disk) Get(_ context.Context, key string) ([]byte, error) {
	p, err := d.path(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

func (d *Disk) Delete(_ context.Context, key string) error {
	p, err := d.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (d *Disk) Check(context.Context) error { return os.MkdirAll(d.Dir, 0o700) }

// ── Normalising an upload ───────────────────────────────────────────────────

const (
	// MaxUpload bounds the request body.
	MaxUpload = 12 << 20
	maxPixels = 40_000_000
	maxEdge   = 1600
)

var ErrBadImage = errors.New("that file is not a photo we can read (use JPEG, PNG or WebP)")

// Normalize decodes an upload and re-encodes it as a JPEG. Re-encoding is the
// point: it drops EXIF, so a phone's GPS tag can never reach another user. The
// pixel check runs before decoding, so a small file claiming huge dimensions
// cannot exhaust memory.
func Normalize(data []byte) (out []byte, w, h int, err error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, ErrBadImage
	}
	if cfg.Width*cfg.Height > maxPixels || cfg.Width < 200 || cfg.Height < 200 {
		return nil, 0, 0, fmt.Errorf("%w: it must be between 200 pixels on a side and 40 megapixels", ErrBadImage)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, ErrBadImage
	}
	b := img.Bounds()
	w, h = b.Dx(), b.Dy()
	if w > maxEdge || h > maxEdge {
		if w >= h {
			h, w = h*maxEdge/w, maxEdge
		} else {
			w, h = w*maxEdge/h, maxEdge
		}
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img = dst
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}
