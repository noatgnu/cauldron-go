package services

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/ulikunitz/xz"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type FileService struct {
	ctx              context.Context
	progressNotifier *ProgressNotifier
}

func NewFileService(ctx context.Context) *FileService {
	return &FileService{
		ctx:              ctx,
		progressNotifier: NewProgressNotifier(ctx),
	}
}

func (f *FileService) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *FileService) ReadFileLines(path string, limit int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() && count < limit {
		lines = append(lines, scanner.Text())
		count++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func (f *FileService) OpenFileDialog(title string, filters []wailsRuntime.FileFilter) (string, error) {
	path, err := wailsRuntime.OpenFileDialog(f.ctx, wailsRuntime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
	return path, err
}

func (f *FileService) OpenDirectoryDialog(title string) (string, error) {
	path, err := wailsRuntime.OpenDirectoryDialog(f.ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
	})
	return path, err
}

func (f *FileService) SaveFileDialog(title string, defaultName string) (string, error) {
	path, err := wailsRuntime.SaveFileDialog(f.ctx, wailsRuntime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultName,
	})
	return path, err
}

func (f *FileService) OpenDirectoryInExplorer(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absPath)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", absPath)
	case "darwin":
		cmd = exec.Command("open", absPath)
	case "linux":
		cmd = exec.Command("xdg-open", absPath)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func (f *FileService) GetFileInfo(path string) (*models.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	preview, _ := f.ReadFileLines(path, 10)

	return &models.FileInfo{
		Name:    info.Name(),
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Preview: preview,
	}, nil
}

func (f *FileService) ExtractTarGz(archivePath string, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	totalSize := fileInfo.Size()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var processedSize int64
	lastEmitSize := int64(0)
	emitThreshold := totalSize / 20

	fileName := filepath.Base(archivePath)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			processedSize += header.Size

			if processedSize-lastEmitSize >= emitThreshold || processedSize >= totalSize {
				percentage := float64(processedSize) / float64(totalSize) * 100
				if percentage > 100 {
					percentage = 100
				}

				f.progressNotifier.EmitProgress(ProgressTypeExtract, fileName,
					fmt.Sprintf("Extracting %s", fileName), percentage)

				lastEmitSize = processedSize
			}
		}
	}

	f.progressNotifier.EmitComplete(ProgressTypeExtract, fileName,
		fmt.Sprintf("Extracted %s", fileName))

	return nil
}

func (f *FileService) ExtractTarXz(archivePath string, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	totalSize := fileInfo.Size()

	xzReader, err := xz.NewReader(file)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(xzReader)

	var processedSize int64
	lastEmitSize := int64(0)
	emitThreshold := totalSize / 20

	fileName := filepath.Base(archivePath)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			processedSize += header.Size

			if processedSize-lastEmitSize >= emitThreshold || processedSize >= totalSize {
				percentage := float64(processedSize) / float64(totalSize) * 100
				if percentage > 100 {
					percentage = 100
				}

				f.progressNotifier.EmitProgress(ProgressTypeExtract, fileName,
					fmt.Sprintf("Extracting %s", fileName), percentage)

				lastEmitSize = processedSize
			}
		}
	}

	f.progressNotifier.EmitComplete(ProgressTypeExtract, fileName,
		fmt.Sprintf("Extracted %s", fileName))

	return nil
}

func (f *FileService) DownloadFile(url string, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	os.MkdirAll(filepath.Dir(destPath), 0755)

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

type DataFilePreview struct {
	Headers   []string   `json:"headers"`
	Rows      [][]string `json:"rows"`
	TotalRows int        `json:"totalRows"`
	FileType  string     `json:"fileType"`
}

func (f *FileService) ParseDataFile(path string, previewRows int) (*DataFilePreview, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(path))
	delimiter := ','
	fileType := "csv"

	if ext == ".tsv" || ext == ".txt" {
		delimiter = '\t'
		fileType = "tsv"
	}

	reader := csv.NewReader(file)
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var rows [][]string
	rowCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if rowCount < previewRows {
			rows = append(rows, record)
		}
		rowCount++
	}

	return &DataFilePreview{
		Headers:   headers,
		Rows:      rows,
		TotalRows: rowCount,
		FileType:  fileType,
	}, nil
}

func (f *FileService) OpenDataFileDialog() (string, error) {
	filters := []wailsRuntime.FileFilter{
		{
			DisplayName: "Data Files (*.csv, *.tsv, *.txt)",
			Pattern:     "*.csv;*.tsv;*.txt",
		},
		{
			DisplayName: "CSV Files (*.csv)",
			Pattern:     "*.csv",
		},
		{
			DisplayName: "TSV Files (*.tsv)",
			Pattern:     "*.tsv",
		},
		{
			DisplayName: "All Files (*.*)",
			Pattern:     "*.*",
		},
	}

	return f.OpenFileDialog("Select Data File", filters)
}
