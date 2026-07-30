package fileutil

import (
	"archive/zip"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"   // 依赖: go get github.com/bodgit/sevenzip
	"github.com/nwaples/rardecode" // 依赖: go get github.com/nwaples/rardecode
)

// --- 文件上传 ---

// SaveUploadFile 处理单个文件上传，将其保存到指定的目标目录中。
func SaveUploadFile(fileHeader *multipart.FileHeader, destDir string) (savedPath string, err error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("打开上传的文件失败: %w", err)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("关闭上传的文件句柄失败: %w", closeErr)
		}
	}()

	fileName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	savedPath = filepath.Join(destDir, fileName)

	if err = os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("创建目标目录 '%s' 失败: %w", destDir, err)
	}

	destFile, err := os.Create(savedPath)
	if err != nil {
		return "", fmt.Errorf("创建目标文件 '%s' 失败: %w", savedPath, err)
	}
	defer func() {
		closeErr := destFile.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("关闭目标文件句柄 '%s' 失败: %w", savedPath, closeErr)
		}
	}()

	if _, err = io.Copy(destFile, file); err != nil {
		return "", fmt.Errorf("保存文件到 '%s' 失败: %w", savedPath, err)
	}

	return savedPath, nil
}

// --- 文件解压 ---

// Decompress 是解压文件的统一入口。
func Decompress(archivePath, destDir string) (finalDestDir string, err error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("创建目标目录 '%s' 失败: %w", destDir, err)
	}

	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".zip":
		return decompressZip(archivePath, destDir)
	case ".rar":
		return decompressRar(archivePath, destDir)
	case ".7z":
		return decompress7z(archivePath, destDir)
	default:
		return "", fmt.Errorf("不支持的压缩格式: %s", ext)
	}
}

// archiveEntry 定义了一个通用的压缩文件条目接口。
type archiveEntry interface {
	Path() string
	IsDir() bool
	Open() (io.ReadCloser, error)
}

// processEntries 是处理解压的核心逻辑。
func processEntries(destDir string, entries []archiveEntry) error {
	if len(entries) == 0 {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		paths = append(paths, entry.Path())
	}
	commonPrefix := findCommonPrefix(paths)

	for _, entry := range entries {
		strippedPath := strings.TrimPrefix(entry.Path(), commonPrefix)
		if strippedPath == "" {
			continue
		}

		targetPath := filepath.Join(destDir, strippedPath)

		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("检测到不安全的解压路径: %s", targetPath)
		}

		if entry.IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("创建解压目录 '%s' 失败: %w", targetPath, err)
			}
		} else {
			sourceReader, err := entry.Open()
			if err != nil {
				return fmt.Errorf("打开压缩包内文件 '%s' 失败: %w", entry.Path(), err)
			}

			extractErr := extractFile(targetPath, sourceReader)
			closeErr := sourceReader.Close() // 必须在这里关闭并检查错误

			if extractErr != nil {
				return extractErr // 优先返回解压/写入文件的错误
			}
			if closeErr != nil {
				return fmt.Errorf("关闭压缩包内文件 '%s' 的读取器失败: %w", entry.Path(), closeErr)
			}
		}
	}
	return nil
}

// findCommonPrefix 查找并返回路径列表中的单层公共根目录。
func findCommonPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	firstPath := filepath.ToSlash(paths[0])
	idx := strings.Index(firstPath, "/")
	if idx == -1 || idx == len(firstPath)-1 {
		return ""
	}
	potentialPrefix := firstPath[:idx+1]
	for _, p := range paths {
		if !strings.HasPrefix(filepath.ToSlash(p), potentialPrefix) {
			return ""
		}
	}
	return potentialPrefix
}

// extractFile 负责将一个 reader 的内容写入到目标文件中。
func extractFile(targetPath string, reader io.Reader) (err error) {
	if err = os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("为解压文件创建父目录 '%s' 失败: %w", filepath.Dir(targetPath), err)
	}

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("创建解压文件 '%s' 失败: %w", targetPath, err)
	}
	defer func() {
		closeErr := targetFile.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("关闭解压后的目标文件 '%s' 失败: %w", targetPath, closeErr)
		}
	}()

	if _, err = io.Copy(targetFile, reader); err != nil {
		return fmt.Errorf("解压文件时复制内容到 '%s' 失败: %w", targetPath, err)
	}

	return nil
}

// --- 格式适配器 ---

type zipEntry struct{ f *zip.File }

func (e zipEntry) Path() string                 { return e.f.Name }
func (e zipEntry) IsDir() bool                  { return e.f.FileInfo().IsDir() }
func (e zipEntry) Open() (io.ReadCloser, error) { return e.f.Open() }

func decompressZip(archivePath, destDir string) (finalDestDir string, err error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("打开 ZIP 文件 '%s' 失败: %w", archivePath, err)
	}
	defer func() {
		closeErr := r.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("关闭 ZIP 文件句柄 '%s' 失败: %w", archivePath, closeErr)
		}
	}()

	var entries []archiveEntry
	for _, f := range r.File {
		entries = append(entries, zipEntry{f})
	}

	return destDir, processEntries(destDir, entries)
}

type sevenZipEntry struct{ f *sevenzip.File }

func (e sevenZipEntry) Path() string                 { return e.f.Name }
func (e sevenZipEntry) IsDir() bool                  { return e.f.FileInfo().IsDir() }
func (e sevenZipEntry) Open() (io.ReadCloser, error) { return e.f.Open() }

func decompress7z(archivePath, destDir string) (finalDestDir string, err error) {
	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("打开 7z 文件 '%s' 失败: %w", archivePath, err)
	}
	defer func() {
		closeErr := r.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("关闭 7z 文件句柄 '%s' 失败: %w", archivePath, closeErr)
		}
	}()

	var entries []archiveEntry
	for _, f := range r.File {
		entries = append(entries, sevenZipEntry{f})
	}

	return destDir, processEntries(destDir, entries)
}

func decompressRar(archivePath, destDir string) (finalDestDir string, err error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("打开 RAR 文件 '%s' 失败: %w", archivePath, err)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("关闭 RAR 文件句柄 '%s' 失败: %w", archivePath, closeErr)
		}
	}()

	var paths []string
	r, err := rardecode.NewReader(file, "")
	if err != nil {
		return "", fmt.Errorf("创建 RAR 读取器失败 (pass 1): %w", err)
	}
	for {
		header, nextErr := r.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return "", fmt.Errorf("读取 RAR header 失败 (pass 1): %w", nextErr)
		}
		paths = append(paths, header.Name)
	}
	commonPrefix := findCommonPrefix(paths)

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("重置 RAR 文件指针失败: %w", err)
	}

	r, err = rardecode.NewReader(file, "")
	if err != nil {
		return "", fmt.Errorf("创建 RAR 读取器失败 (pass 2): %w", err)
	}
	for {
		header, nextErr := r.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return "", fmt.Errorf("读取 RAR header 失败 (pass 2): %w", nextErr)
		}

		strippedPath := strings.TrimPrefix(header.Name, commonPrefix)
		if strippedPath == "" {
			continue
		}
		targetPath := filepath.Join(destDir, strippedPath)

		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return "", fmt.Errorf("检测到不安全的解压路径: %s", targetPath)
		}

		if header.IsDir {
			if err = os.MkdirAll(targetPath, 0755); err != nil {
				return "", fmt.Errorf("创建解压目录 '%s' 失败: %w", targetPath, err)
			}
		} else {
			if err = extractFile(targetPath, r); err != nil {
				return "", err
			}
		}
	}

	return destDir, nil
}

// Compress 将源目录压缩成一个 zip 文件。
// `baseInZip` 参数指定了 zip 文件内部文件的目录前缀。
// 如果 `baseInZip` 是空字符串，文件将被放在 zip 的根目录下。
func Compress(source, target, baseInZip string) (err error) {
	zipfile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("创建目标zip文件 '%s' 失败: %w", target, err)
	}
	defer func() {
		closeErr := zipfile.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("关闭目标zip文件句柄 '%s' 失败: %w", target, closeErr)
		}
	}()

	archive := zip.NewWriter(zipfile)
	defer func() {
		closeErr := archive.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("关闭zip写入器失败: %w", closeErr)
		}
	}()

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 排除源目录本身
		if path == source {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("创建zip文件头失败 for '%s': %w", path, err)
		}

		header.Method = zip.Deflate

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("获取相对路径失败 for '%s': %w", path, err)
		}
		header.Name = filepath.ToSlash(filepath.Join(baseInZip, relPath))

		if info.IsDir() {
			header.Name += "/"
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("在zip中创建条目失败 for '%s': %w", header.Name, err)
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("打开源文件 '%s' 失败: %w", path, err)
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			if err != nil {
				return fmt.Errorf("复制文件内容失败 for '%s': %w", path, err)
			}
		}
		return nil
	})

	return err
}
