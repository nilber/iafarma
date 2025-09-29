package services

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

// StorageService provides file storage functionality
type StorageService struct {
	s3Client *s3.S3
	bucket   string
	baseURL  string
}

// NewStorageService creates a new storage service
func NewStorageService() (*StorageService, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	bucket := os.Getenv("S3_BUCKET")

	if accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("S3 configuration missing")
	}

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:   aws.String("us-east-1"),
		Endpoint: aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			"",
		),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create S3 client
	s3Client := s3.New(sess)

	// Build base URL for public access
	baseURL := fmt.Sprintf("https://%s", bucket)

	return &StorageService{
		s3Client: s3Client,
		bucket:   bucket,
		baseURL:  baseURL,
	}, nil
}

// UploadAudioFile downloads, converts and uploads an audio file to S3
func (s *StorageService) UploadAudioFile(mediaURL, tenantID, customerID, messageID string) (string, error) {
	log.Printf("Starting audio file upload process for message: %s", messageID)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "audio_conversion_")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download original file
	originalPath := filepath.Join(tempDir, "original")
	err = s.downloadFile(mediaURL, originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to download audio file: %w", err)
	}

	// Convert to MP3
	convertedPath := filepath.Join(tempDir, "converted.mp3")
	err = s.convertToMP3(originalPath, convertedPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert audio file: %w", err)
	}

	// Generate S3 key with structure: tenant_id/conversations/customer_id/audio_messageID.mp3
	s3Key := fmt.Sprintf("%s/conversations/%s/audio_%s.mp3", tenantID, customerID, messageID)

	// Upload to S3
	publicURL, err := s.uploadToS3(convertedPath, s3Key, "audio/mp3")
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Printf("Audio file successfully uploaded to S3: %s", publicURL)
	return publicURL, nil
}

// UploadImageFile uploads an image file to S3
func (s *StorageService) UploadImageFile(mediaURL, tenantID, customerID, messageID string) (string, error) {
	log.Printf("Starting image file upload process for message: %s", messageID)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "image_upload_")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download original file
	originalPath := filepath.Join(tempDir, "original")
	err = s.downloadFile(mediaURL, originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to download image file: %w", err)
	}

	// Detect content type
	file, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read file for content type detection: %w", err)
	}
	file.Seek(0, 0) // Reset file pointer

	contentType := http.DetectContentType(buffer)
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("file is not an image: %s", contentType)
	}

	// Determine file extension
	var ext string
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	default:
		ext = ".jpg" // Default to jpg
	}

	// Generate S3 key with structure: tenant_id/conversations/customer_id/image_messageID.ext
	s3Key := fmt.Sprintf("%s/conversations/%s/image_%s%s", tenantID, customerID, messageID, ext)

	// Upload to S3
	publicURL, err := s.uploadToS3(originalPath, s3Key, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Printf("Image file successfully uploaded to S3: %s", publicURL)
	return publicURL, nil
}

// downloadFile downloads a file from URL to local path
func (s *StorageService) downloadFile(url, filepath string) error {
	log.Printf("Downloading file from: %s", url)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	log.Printf("File downloaded successfully to: %s", filepath)
	return nil
}

// convertToMP3 converts audio file to MP3 format using FFmpeg
func (s *StorageService) convertToMP3(inputPath, outputPath string) error {
	log.Printf("Converting audio file to MP3: %s -> %s", inputPath, outputPath)

	// FFmpeg command to convert to MP3
	cmd := exec.Command("ffmpeg",
		"-i", inputPath, // Input file
		"-acodec", "mp3", // Audio codec
		"-ab", "128k", // Audio bitrate
		"-ar", "44100", // Audio sample rate
		"-y",       // Overwrite output file
		outputPath, // Output file
	)

	// Capture output for debugging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("FFmpeg error: %s", stderr.String())
		return fmt.Errorf("FFmpeg conversion failed: %w", err)
	}

	log.Printf("Audio conversion completed successfully")
	return nil
}

// convertToOggOpus converts audio file to OGG Opus format using FFmpeg
func (s *StorageService) convertToOggOpus(inputPath, outputPath string) error {
	log.Printf("Converting audio file to OGG Opus: %s -> %s", inputPath, outputPath)

	// FFmpeg command to convert to OGG Opus
	cmd := exec.Command("ffmpeg",
		"-i", inputPath, // Input file
		"-c:a", "libopus", // Opus codec
		"-b:a", "64k", // Audio bitrate optimized for voice
		"-ar", "48000", // Sample rate (48kHz is optimal for Opus)
		"-application", "voip", // Optimize for voice
		"-y",       // Overwrite output file
		outputPath, // Output file
	)

	// Capture output for debugging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("FFmpeg error: %s", stderr.String())
		return fmt.Errorf("FFmpeg OGG Opus conversion failed: %w", err)
	}

	log.Printf("Audio conversion to OGG Opus completed successfully")
	return nil
}

// uploadToS3 uploads a file to S3 with public read access
func (s *StorageService) uploadToS3(filePath, s3Key, contentType string) (string, error) {
	log.Printf("Uploading file to S3: %s", s3Key)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Upload to S3 without ACL
	_, err = s.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(s3Key),
		Body:          file,
		ContentLength: aws.Int64(fileInfo.Size()),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Build public URL
	publicURL := fmt.Sprintf("%s/%s", s.baseURL, s3Key)

	log.Printf("File uploaded to S3 successfully: %s", publicURL)
	return publicURL, nil
}

// UploadMultipartFile uploads a multipart file directly to S3
func (s *StorageService) UploadMultipartFile(fileHeader *multipart.FileHeader, tenantID, folderType string) (string, error) {
	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Generate unique filename
	fileID := uuid.New().String()
	ext := filepath.Ext(fileHeader.Filename)
	filename := fmt.Sprintf("%s%s", fileID, ext)

	// Generate S3 key with structure: tenant_id/folder_type/filename
	s3Key := fmt.Sprintf("%s/%s/%s", tenantID, folderType, filename)

	// Detect content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read file for content type detection: %w", err)
	}
	file.Seek(0, 0) // Reset file pointer

	contentType := http.DetectContentType(buffer)

	// Upload to S3 without ACL (bucket should have public access policy)
	_, err = s.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s3Key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Build public URL
	publicURL := fmt.Sprintf("%s/%s", s.baseURL, s3Key)

	log.Printf("Multipart file uploaded to S3 successfully: %s", publicURL)
	return publicURL, nil
}

// UploadMediaFile uploads a media file using message ID as filename
func (s *StorageService) UploadMediaFile(fileHeader *multipart.FileHeader, tenantID, messageID, mediaType string) (string, error) {
	// For audio files, we need to convert to OGG Opus format
	if mediaType == "audio" {
		return s.UploadAndConvertAudioFile(fileHeader, tenantID, messageID)
	}

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Use message ID as filename with appropriate extension
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		// Determine extension based on media type
		switch mediaType {
		case "image":
			ext = ".jpg"
		case "document":
			ext = ".pdf"
		default:
			ext = ".bin"
		}
	}
	filename := fmt.Sprintf("%s%s", messageID, ext)

	// Generate S3 key with structure: tenant_id/media/filename
	s3Key := fmt.Sprintf("%s/media/%s", tenantID, filename)

	// Detect content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read file for content type detection: %w", err)
	}
	file.Seek(0, 0) // Reset file pointer

	contentType := http.DetectContentType(buffer)

	// Upload to S3 without ACL
	_, err = s.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s3Key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Build public URL
	publicURL := fmt.Sprintf("%s/%s", s.baseURL, s3Key)

	log.Printf("Media file uploaded to S3 successfully: %s", publicURL)
	return publicURL, nil
}

// UploadAndConvertAudioFile uploads and converts audio to OGG Opus format
func (s *StorageService) UploadAndConvertAudioFile(fileHeader *multipart.FileHeader, tenantID, messageID string) (string, error) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "audio_upload_")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Save original file
	originalPath := filepath.Join(tempDir, "original"+filepath.Ext(fileHeader.Filename))
	originalFile, err := os.Create(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Open uploaded file and copy to temp location
	uploadedFile, err := fileHeader.Open()
	if err != nil {
		originalFile.Close()
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer uploadedFile.Close()

	_, err = io.Copy(originalFile, uploadedFile)
	originalFile.Close()
	if err != nil {
		return "", fmt.Errorf("failed to copy uploaded file: %w", err)
	}

	// Convert to OGG Opus
	convertedPath := filepath.Join(tempDir, "converted.ogg")
	err = s.convertToOggOpus(originalPath, convertedPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert audio file: %w", err)
	}

	// Generate S3 key with .ogg extension
	filename := fmt.Sprintf("%s.ogg", messageID)
	s3Key := fmt.Sprintf("%s/media/%s", tenantID, filename)

	// Upload converted file to S3
	publicURL, err := s.uploadToS3(convertedPath, s3Key, "audio/ogg; codecs=opus")
	if err != nil {
		return "", fmt.Errorf("failed to upload converted audio: %w", err)
	}

	log.Printf("Audio file converted and uploaded successfully: %s", publicURL)
	return publicURL, nil
}

// DeleteFile deletes a file from S3
func (s *StorageService) DeleteFile(s3Key string) error {
	_, err := s.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	log.Printf("File deleted from S3: %s", s3Key)
	return nil
}
