package fsxprovider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Abraxas-365/craftale/errx"
	"github.com/Abraxas-365/craftale/fsx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	// Error registry for S3 file system
	s3Errors = errx.NewRegistry("S3FS")

	// Error codes
	ErrNotFound         = s3Errors.Register("NOT_FOUND", errx.TypeNotFound, 404, "Resource not found in S3")
	ErrAccessDenied     = s3Errors.Register("ACCESS_DENIED", errx.TypeAuthorization, 403, "Access denied to S3 resource")
	ErrInvalidPath      = s3Errors.Register("INVALID_PATH", errx.TypeValidation, 400, "Invalid S3 path")
	ErrBucketNotExists  = s3Errors.Register("BUCKET_NOT_EXISTS", errx.TypeNotFound, 404, "S3 bucket does not exist")
	ErrObjectNotExists  = s3Errors.Register("OBJECT_NOT_EXISTS", errx.TypeNotFound, 404, "S3 object does not exist")
	ErrFailedUpload     = s3Errors.Register("FAILED_UPLOAD", errx.TypeSystem, 500, "Failed to upload to S3")
	ErrFailedDownload   = s3Errors.Register("FAILED_DOWNLOAD", errx.TypeSystem, 500, "Failed to download from S3")
	ErrFailedDelete     = s3Errors.Register("FAILED_DELETE", errx.TypeSystem, 500, "Failed to delete from S3")
	ErrFailedList       = s3Errors.Register("FAILED_LIST", errx.TypeSystem, 500, "Failed to list S3 objects")
	ErrFailedStat       = s3Errors.Register("FAILED_STAT", errx.TypeSystem, 500, "Failed to get S3 object stats")
	ErrInvalidOperation = s3Errors.Register("INVALID_OPERATION", errx.TypeBadRequest, 400, "Invalid operation for S3")
	ErrEmptyBucketName  = s3Errors.Register("EMPTY_BUCKET_NAME", errx.TypeValidation, 400, "Bucket name cannot be empty")
	ErrInvalidKey       = s3Errors.Register("INVALID_KEY", errx.TypeValidation, 400, "Invalid S3 key format")
)

// S3FileSystem implements the FileSystem interface for AWS S3
type S3FileSystem struct {
	client   *s3.Client
	bucket   string
	rootPath string
}

// NewS3FileSystem creates a new S3FileSystem
func NewS3FileSystem(client *s3.Client, bucket string, rootPath string) *S3FileSystem {
	// Ensure rootPath doesn't start with slash but ends with one if not empty
	if rootPath != "" {
		rootPath = strings.TrimPrefix(rootPath, "/")
		if !strings.HasSuffix(rootPath, "/") {
			rootPath += "/"
		}
	}

	return &S3FileSystem{
		client:   client,
		bucket:   bucket,
		rootPath: rootPath,
	}
}

// s3Key converts a file system path to an S3 key
func (fs *S3FileSystem) s3Key(path string) string {
	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	// Join with root path if any
	if fs.rootPath != "" {
		return fs.rootPath + path
	}
	return path
}

// ReadFile reads an entire file from S3
func (fs *S3FileSystem) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if fs.bucket == "" {
		return nil, s3Errors.New(ErrEmptyBucketName)
	}

	key := fs.s3Key(path)

	output, err := fs.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, s3Errors.NewWithCause(ErrObjectNotExists, err).
				WithDetail("path", path).
				WithDetail("key", key)
		}

		return nil, errx.Wrap(err, "Failed to read file from S3", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, errx.Wrap(err, "Failed to read response body", errx.TypeSystem).
			WithDetail("path", path)
	}

	return data, nil
}

// ReadFileStream returns a reader for a file in S3
func (fs *S3FileSystem) ReadFileStream(ctx context.Context, path string) (io.ReadCloser, error) {
	if fs.bucket == "" {
		return nil, s3Errors.New(ErrEmptyBucketName)
	}

	key := fs.s3Key(path)

	output, err := fs.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, s3Errors.NewWithCause(ErrObjectNotExists, err).
				WithDetail("path", path).
				WithDetail("key", key)
		}

		return nil, errx.Wrap(err, "Failed to open file stream from S3", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	return output.Body, nil
}

// Stat returns file info for the specified path in S3
func (fs *S3FileSystem) Stat(ctx context.Context, path string) (fsx.FileInfo, error) {
	if fs.bucket == "" {
		return fsx.FileInfo{}, s3Errors.New(ErrEmptyBucketName)
	}

	key := fs.s3Key(path)

	// Check if it's a "directory" by looking for objects with this prefix
	// S3 doesn't have real directories, so we check if there are objects with this prefix
	if !strings.HasSuffix(key, "/") {
		dirKey := key + "/"
		listOutput, err := fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:    aws.String(fs.bucket),
			Prefix:    aws.String(dirKey),
			MaxKeys:   aws.Int32(1),
			Delimiter: aws.String("/"),
		})

		if err == nil && (len(listOutput.Contents) > 0 || len(listOutput.CommonPrefixes) > 0) {
			// This is a "directory"
			return fsx.FileInfo{
				Name:     filepath.Base(path),
				IsDir:    true,
				ModTime:  time.Time{}, // S3 doesn't have directory modification times
				Metadata: make(map[string]string),
			}, nil
		}
	}

	// Check if it's a file
	headOutput, err := fs.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var nsk *types.NotFound
		if errors.As(err, &nsk) {
			return fsx.FileInfo{}, s3Errors.NewWithCause(ErrObjectNotExists, err).
				WithDetail("path", path).
				WithDetail("key", key)
		}

		return fsx.FileInfo{}, errx.Wrap(err, "Failed to get S3 object metadata", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	// Convert S3 metadata to map
	metadata := make(map[string]string)
	for k, v := range headOutput.Metadata {
		metadata[k] = v
	}

	// Determine if it's a directory based on the key suffix
	isDir := strings.HasSuffix(key, "/")

	return fsx.FileInfo{
		Name:        filepath.Base(path),
		Size:        *headOutput.ContentLength,
		ModTime:     *headOutput.LastModified,
		IsDir:       isDir,
		ContentType: aws.ToString(headOutput.ContentType),
		Metadata:    metadata,
	}, nil
}

// List returns a listing of files and directories in the specified path
func (fs *S3FileSystem) List(ctx context.Context, path string) ([]fsx.FileInfo, error) {
	if fs.bucket == "" {
		return nil, s3Errors.New(ErrEmptyBucketName)
	}

	// Normalize path to ensure it ends with a slash if not empty
	if path != "" && !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	key := fs.s3Key(path)

	// Only use delimiter if we're not at the root
	var delimiter *string
	if key != "" {
		delimiter = aws.String("/")
	}

	listOutput, err := fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(fs.bucket),
		Prefix:    aws.String(key),
		Delimiter: delimiter,
	})

	if err != nil {
		return nil, errx.Wrap(err, "Failed to list S3 objects", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	files := make([]fsx.FileInfo, 0)

	// Add directories (common prefixes)
	for _, prefix := range listOutput.CommonPrefixes {
		dirName := filepath.Base(strings.TrimSuffix(aws.ToString(prefix.Prefix), "/"))
		files = append(files, fsx.FileInfo{
			Name:     dirName,
			IsDir:    true,
			ModTime:  time.Time{}, // S3 doesn't provide this for prefixes
			Metadata: make(map[string]string),
		})
	}

	// Add files
	for _, obj := range listOutput.Contents {
		// Skip if this is the directory itself (prefix ending with /)
		if aws.ToString(obj.Key) == key {
			continue
		}

		name := filepath.Base(aws.ToString(obj.Key))
		isDir := strings.HasSuffix(aws.ToString(obj.Key), "/")

		files = append(files, fsx.FileInfo{
			Name:     name,
			Size:     *obj.Size,
			ModTime:  *obj.LastModified,
			IsDir:    isDir,
			Metadata: make(map[string]string),
		})
	}

	return files, nil
}

// WriteFile writes data to a file in S3
func (fs *S3FileSystem) WriteFile(ctx context.Context, path string, data []byte) error {
	if fs.bucket == "" {
		return s3Errors.New(ErrEmptyBucketName)
	}

	key := fs.s3Key(path)

	// Detect content type
	contentType := "application/octet-stream"

	_, err := fs.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(fs.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return errx.Wrap(err, "Failed to write file to S3", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	return nil
}

// WriteFileStream writes a stream to a file in S3
func (fs *S3FileSystem) WriteFileStream(ctx context.Context, path string, r io.Reader) error {
	if fs.bucket == "" {
		return s3Errors.New(ErrEmptyBucketName)
	}

	key := fs.s3Key(path)

	// Detect content type
	contentType := "application/octet-stream"

	_, err := fs.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(fs.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return errx.Wrap(err, "Failed to write file stream to S3", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	return nil
}

// CreateDir creates a "directory" in S3 (which is just an empty object with a trailing slash)
func (fs *S3FileSystem) CreateDir(ctx context.Context, path string) error {
	if fs.bucket == "" {
		return s3Errors.New(ErrEmptyBucketName)
	}

	// Ensure path ends with slash
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	key := fs.s3Key(path)

	_, err := fs.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(fs.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte{}),
		ContentType: aws.String("application/x-directory"),
	})

	if err != nil {
		return errx.Wrap(err, "Failed to create directory in S3", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	return nil
}

// DeleteFile deletes a file from S3
func (fs *S3FileSystem) DeleteFile(ctx context.Context, path string) error {
	if fs.bucket == "" {
		return s3Errors.New(ErrEmptyBucketName)
	}

	key := fs.s3Key(path)

	_, err := fs.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return errx.Wrap(err, "Failed to delete file from S3", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	return nil
}

// DeleteDir deletes a directory from S3
func (fs *S3FileSystem) DeleteDir(ctx context.Context, path string, recursive bool) error {
	if fs.bucket == "" {
		return s3Errors.New(ErrEmptyBucketName)
	}

	// Ensure path ends with slash for directories
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	key := fs.s3Key(path)

	if !recursive {
		// Check if directory is empty
		listOutput, err := fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:  aws.String(fs.bucket),
			Prefix:  aws.String(key),
			MaxKeys: aws.Int32(2), // Directory marker + 1 more
		})

		if err != nil {
			return errx.Wrap(err, "Failed to check if directory is empty", errx.TypeExternal).
				WithDetail("path", path).
				WithDetail("bucket", fs.bucket).
				WithDetail("key", key)
		}

		// If we have more than just the directory marker, fail
		if len(listOutput.Contents) > 1 {
			return s3Errors.New(ErrInvalidOperation).
				WithDetail("message", "Directory is not empty").
				WithDetail("path", path)
		}

		// Delete the directory marker
		_, err = fs.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(fs.bucket),
			Key:    aws.String(key),
		})

		if err != nil {
			return errx.Wrap(err, "Failed to delete directory from S3", errx.TypeExternal).
				WithDetail("path", path).
				WithDetail("bucket", fs.bucket).
				WithDetail("key", key)
		}

		return nil
	}

	// For recursive delete, we need to list and delete all objects with the prefix
	var continuationToken *string

	for {
		listOutput, err := fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(fs.bucket),
			Prefix:            aws.String(key),
			ContinuationToken: continuationToken,
		})

		if err != nil {
			return errx.Wrap(err, "Failed to list objects for recursive delete", errx.TypeExternal).
				WithDetail("path", path).
				WithDetail("bucket", fs.bucket).
				WithDetail("key", key)
		}

		if len(listOutput.Contents) == 0 {
			break
		}

		// Create delete objects input
		objectsToDelete := make([]types.ObjectIdentifier, len(listOutput.Contents))
		for i, obj := range listOutput.Contents {
			objectsToDelete[i] = types.ObjectIdentifier{
				Key: obj.Key,
			}
		}

		_, err = fs.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(fs.bucket),
			Delete: &types.Delete{
				Objects: objectsToDelete,
				Quiet:   aws.Bool(true),
			},
		})

		if err != nil {
			return errx.Wrap(err, "Failed to delete objects during recursive delete", errx.TypeExternal).
				WithDetail("path", path).
				WithDetail("bucket", fs.bucket).
				WithDetail("key", key)
		}

		// If there are more objects to delete, continue
		if !*listOutput.IsTruncated {
			break
		}

		continuationToken = listOutput.NextContinuationToken
	}

	return nil
}

// Join joins path elements into a single path
func (fs *S3FileSystem) Join(elem ...string) string {
	// Remove any leading/trailing slashes from internal elements
	for i := 1; i < len(elem); i++ {
		elem[i] = strings.Trim(elem[i], "/")
	}
	// Keep any leading slash in the first element
	if len(elem) > 0 {
		elem[0] = strings.TrimSuffix(elem[0], "/")
	}

	return path.Join(elem...)
}

// Exists checks if a file or directory exists in S3
func (fs *S3FileSystem) Exists(ctx context.Context, path string) (bool, error) {
	if fs.bucket == "" {
		return false, s3Errors.New(ErrEmptyBucketName)
	}

	key := fs.s3Key(path)

	// Check if it's a file
	_, err := fs.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})

	if err == nil {
		return true, nil
	}

	// It's not a file, check if it's a directory

	// Add trailing slash if not present
	if !strings.HasSuffix(key, "/") {
		key = key + "/"
	}

	// For directories, we check if there are any objects with this prefix
	listOutput, err := fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(fs.bucket),
		Prefix:  aws.String(key),
		MaxKeys: aws.Int32(1),
	})

	if err != nil {
		return false, errx.Wrap(err, "Failed to check if path exists in S3", errx.TypeExternal).
			WithDetail("path", path).
			WithDetail("bucket", fs.bucket).
			WithDetail("key", key)
	}

	return len(listOutput.Contents) > 0, nil
}
