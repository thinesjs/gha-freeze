package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v58/github"
)

const (
	owner = "thinesjs"
	repo  = "gha-freeze"
)

type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	DownloadURL    string
	ReleaseNotes   string
}

func CheckForUpdate(currentVersion string) (*UpdateInfo, error) {
	return CheckForUpdateWithToken(currentVersion, "")
}

func CheckForUpdateWithToken(currentVersion, token string) (*UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var client *github.Client
	if token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}

	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.GetTagName(), "v")
	currentVersionClean := strings.TrimPrefix(currentVersion, "v")

	info := &UpdateInfo{
		CurrentVersion: currentVersionClean,
		LatestVersion:  latestVersion,
		ReleaseNotes:   release.GetBody(),
		Available:      compareVersions(currentVersionClean, latestVersion) < 0,
	}

	if info.Available {
		downloadURL, err := findAssetURL(release)
		if err != nil {
			info.DownloadURL = ""
		} else {
			info.DownloadURL = downloadURL
		}
	}

	return info, nil
}

func compareVersions(current, latest string) int {
	if current == latest {
		return 0
	}
	if current == "dev" || current == "" {
		return -1
	}
	if latest > current {
		return -1
	}
	return 1
}

func findAssetURL(release *github.RepositoryRelease) (string, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	version := strings.TrimPrefix(release.GetTagName(), "v")

	assetName := fmt.Sprintf("gha-freeze_%s_%s_%s", version, osName, arch)

	if osName == "darwin" {
		assetName = fmt.Sprintf("gha-freeze_%s_macOS_%s", version, arch)
	}

	if osName == "windows" {
		assetName += ".zip"
	} else {
		assetName += ".tar.gz"
	}

	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			return asset.GetBrowserDownloadURL(), nil
		}
	}

	return "", fmt.Errorf("no compatible asset found for %s/%s", osName, arch)
}

func DownloadAndInstall(downloadURL string) (err error) {
	tempDir, err := os.MkdirTemp("", "gha-freeze-update-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if rerr := os.RemoveAll(tempDir); rerr != nil && err == nil {
			err = rerr
		}
	}()

	archivePath := filepath.Join(tempDir, "download")
	if err := downloadFile(archivePath, downloadURL); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	extractedPath, err := extractArchive(archivePath, tempDir)
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	backupPath := currentExe + ".old"
	if err := os.Rename(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to backup current executable: %w", err)
	}

	if err := os.Rename(extractedPath, currentExe); err != nil {
		_ = os.Rename(backupPath, currentExe)
		return fmt.Errorf("failed to install new executable: %w", err)
	}

	if err := os.Chmod(currentExe, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	_ = os.Remove(backupPath)

	return nil
}

func downloadFile(filepath string, url string) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractArchive(archivePath, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	}
	return extractTarGz(archivePath, destDir)
}

func extractTarGz(archivePath, destDir string) (binaryPath string, err error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := gzr.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Typeflag == tar.TypeReg && (header.Name == "gha-freeze" || strings.HasSuffix(header.Name, "/gha-freeze")) {
			binaryPath = filepath.Join(destDir, "gha-freeze")
			out, err := os.Create(binaryPath)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return "", err
			}
			if err := out.Close(); err != nil {
				return "", err
			}
			break
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("gha-freeze binary not found in archive")
	}

	return binaryPath, nil
}

func extractZip(archivePath, destDir string) (binaryPath string, err error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := r.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		if f.Name == "gha-freeze.exe" || strings.HasSuffix(f.Name, "/gha-freeze.exe") {
			binaryPath = filepath.Join(destDir, "gha-freeze.exe")
			rc, err := f.Open()
			if err != nil {
				return "", err
			}

			out, err := os.Create(binaryPath)
			if err != nil {
				_ = rc.Close()
				return "", err
			}

			_, err = io.Copy(out, rc)
			_ = out.Close()
			_ = rc.Close()

			if err != nil {
				return "", err
			}
			break
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("gha-freeze.exe binary not found in archive")
	}

	return binaryPath, nil
}
