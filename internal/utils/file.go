package utils

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func FileChecksum(file *os.File, checksumType string) (string, error) {
	var hasher hash.Hash
	switch checksumType {
	case "sha256":
		hasher = sha256.New()
	case "sha512":
		hasher = sha512.New()
	case "sha1":
		hasher = sha1.New()
	case "md5":
		hasher = md5.New()
	default:
		return "", fmt.Errorf("unknown checksum type %s", checksumType)
	}

	file.Seek(0, 0)
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func FileMatchesChecksum(filename, checksum string) (bool, error) {
	// Expects: checksum_type:checksum_string
	checksumDetails := strings.Split(checksum, ":")
	if len(checksumDetails) != 2 {
		return false, fmt.Errorf("invalid checksum format: %s", checksum)
	}

	file, err := os.Open(filename)
	if err != nil {
		return false, fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	hash, err := FileChecksum(file, checksumDetails[0])
	if err != nil {
		return false, fmt.Errorf("failed to compute checksum for %s: %w", filename, err)
	}

	return hash == checksumDetails[1], nil
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || !os.IsNotExist(err)
}

func FileExistsAndMatchesChecksum(filename, checksum string) (bool, error) {
	if !FileExists(filename) {
		return false, nil
	}
	if checksum == "" {
		return true, nil
	}
	return FileMatchesChecksum(filename, checksum)
}

func CopyFile(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	return destFile.Sync()
}

func MoveFile(srcPath, dstPath, checksum string) error {
	exists, err := FileExistsAndMatchesChecksum(dstPath, checksum)
	if err != nil {
		return fmt.Errorf("failed to check destination file for existance and checksum: %w", err)
	}
	if exists {
		return nil
	}

	if err = CopyFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err = os.Remove(srcPath); err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}

func CopyAndReplaceFile(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create/open destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	err = os.Chmod(destPath, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}
	return nil
}

func CreateDirIfNotExists(parts ...string) (string, error) {
	path := filepath.Join(parts...)

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return path, nil
}

func DownloadFile(url, checksum string) (string, error) {
	filename := filepath.Join(os.TempDir(), url[strings.LastIndex(url, "/")+1:])

	out, err := os.Create(filename)
	if err != nil {
		return filename, err
	}
	defer out.Close()

	response, err := http.Get(url)
	if err != nil {
		return filename, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: HTTP %d", url, response.StatusCode)
	}

	_, err = io.Copy(out, response.Body)
	if err != nil {
		return filename, fmt.Errorf("failed to write file %s on disk: %w", filename, err)
	}
	err = out.Sync()
	if err != nil {
		return filename, fmt.Errorf("failed to sync file %s on disk: %w", filename, err)
	}

	if checksum != "" {
		match, err := FileMatchesChecksum(filename, checksum)
		if err != nil {
			return filename, err
		}
		if !match {
			return filename, fmt.Errorf("file %s does not match checksum %s", filename, checksum)
		}
	}

	return filename, err
}

func UnzipFile(source, destination string) error {
	r, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(destination, os.ModeDir)

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(destination, f.Name)

		// Check for ZipSlip (directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	if err := os.Remove(source); err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}
