package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
)

const uploadDir = "./uploads"

func init() {
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		panic("Failed to create upload directory: " + err.Error())
	}
}

func uploadFileHandler(c *gin.Context) {
	// Allow up to 100MB uploads
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 100<<20) // 100 MB
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from form: " + err.Error()})
		return
	}
	defer file.Close()

	dstPath := filepath.Join(uploadDir, header.Filename)
	out, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create file: " + err.Error()})
		return
	}
	defer out.Close()

	if _, err := out.ReadFrom(file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully", "filename": header.Filename})
}

func listFilesHandler(c *gin.Context) {
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to list files: " + err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filename query parameter is required"})
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

	// Routes
	router.POST("/upload", uploadFileHandler)
	router.GET("/list", listFilesHandler)
	router.GET("/download", downloadFileHandler)

	// Run on port 8000
	router.Run(":8000")
}
