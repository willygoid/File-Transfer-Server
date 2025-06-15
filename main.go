package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const (
	uploadDir   = "./uploads"
	workerCount = 5
	jobQueueLen = 100
)

type UploadJob struct {
	FileHeader *multipart.FileHeader
	Ctx        *gin.Context
}

var jobQueue chan UploadJob

func init() {
	// Create upload directory if not exists
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		panic("Failed to create upload directory: " + err.Error())
	}
	// Initialize job queue
	jobQueue = make(chan UploadJob, jobQueueLen)
}

func startWorkerPool() {
	for i := 0; i < workerCount; i++ {
		go worker(i, jobQueue)
	}
}

func worker(id int, jobs <-chan UploadJob) {
	for job := range jobs {
		processUpload(job)
	}
}

func processUpload(job UploadJob) {
	file, err := job.FileHeader.Open()
	if err != nil {
		job.Ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: " + err.Error()})
		return
	}
	defer file.Close()

	dstPath := filepath.Join(uploadDir, filepath.Base(job.FileHeader.Filename))
	dst, err := os.Create(dstPath)
	if err != nil {
		job.Ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create destination file: " + err.Error()})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		job.Ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write file: " + err.Error()})
		return
	}

	job.Ctx.JSON(http.StatusOK, gin.H{
		"message":  "File uploaded successfully",
		"filename": job.FileHeader.Filename,
	})
}

func uploadFileHandler(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file: " + err.Error()})
		return
	}

	select {
	case jobQueue <- UploadJob{FileHeader: fileHeader, Ctx: c}:
		// Job submitted successfully
	default:
		// Queue full
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server is busy. Try again later."})
	}
}

func listFilesHandler(c *gin.Context) {
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files: " + err.Error()})
		return
	}

	var fileNames []string
	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}

	c.JSON(http.StatusOK, gin.H{"files": fileNames})
}

func downloadFileHandler(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing filename"})
		return
	}

	filePath := filepath.Join(uploadDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.FileAttachment(filePath, filename)
}

func main() {
	router := gin.Default()

	// Start worker pool
	startWorkerPool()

	// Routes
	router.POST("/upload", uploadFileHandler)
	router.GET("/list", listFilesHandler)
	router.GET("/download", downloadFileHandler)

	// Start server
	fmt.Println("Server running on http://localhost:8000")
	router.Run(":8000")
}
