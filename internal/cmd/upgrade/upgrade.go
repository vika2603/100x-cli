// Package upgrade implements the `100x upgrade` self-update command.
//
// It downloads a release archive from GitHub, verifies its SHA-256 against
// the published checksums.txt, extracts the `100x` binary, and atomically
// replaces the running executable. The download asset is discovered by
// matching the host OS/arch suffix in checksums.txt, mirroring the logic
// in script/install.sh so odd release tags or naming changes do not break
// the upgrade path.
package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/imroc/req/v3"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/output"
	"github.com/vika2603/100x-cli/internal/version"
)

const binaryName = "100x"

type options struct {
	targetVersion string
	check         bool
	force         bool
}

type renderPayload struct {
	Path            string `json:"path,omitempty"`
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	Installed       string `json:"installed,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	Action          string `json:"action"`
}

// NewCmdUpgrade returns the `100x upgrade` command.
func NewCmdUpgrade(f *factory.Factory) *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade 100x to the latest release",
		Long: "Download a release archive from GitHub, verify its SHA-256, and replace " +
			"the running binary in place. Use --version to pin a tag, --check to inspect " +
			"availability without installing, and --force to reinstall the current target.",
		Example: "# Upgrade to the latest release\n" +
			"  100x upgrade\n\n" +
			"# Only check whether a newer release is available\n" +
			"  100x upgrade --check\n\n" +
			"# Install a specific tag\n" +
			"  100x upgrade --version v1.2.3",
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.targetVersion, "version", "", "Release tag to install (default: latest)")
	cmd.Flags().BoolVar(&opts.check, "check", false, "Check for updates without installing")
	cmd.Flags().BoolVar(&opts.force, "force", false, "Reinstall even when already at the target version")
	_ = cmd.RegisterFlagCompletionFunc("version", cobra.NoFileCompletions)
	return cmd
}

func run(f *factory.Factory, opts options) error {
	// Detach from the global --timeout so a 15s API budget cannot cancel a
	// multi-second binary download. Ctrl+C / SIGTERM still aborts via the
	// signal-aware context.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	binPath, err := resolveBinaryPath()
	if err != nil {
		return fmt.Errorf("resolve current binary: %w", err)
	}

	current := strings.TrimSpace(version.Current.Version)

	// No release stream baked in: this is a source build with nowhere to
	// upgrade from. Report up-to-date so scripts that probe `upgrade --check`
	// don't fail, and so `upgrade` itself is a harmless no-op.
	if strings.TrimSpace(version.Current.RepoSlug) == "" {
		f.IO.Println(fmt.Sprintf("Already on %s; nothing to upgrade.", current))
		return f.IO.Render(renderPayload{
			Path:            binPath,
			Current:         current,
			Latest:          current,
			Installed:       current,
			UpdateAvailable: false,
			Action:          "noop",
		}, func() error { return nil })
	}

	client := req.C().SetTimeout(5 * time.Minute)

	tag, err := resolveTargetTag(ctx, client, opts.targetVersion)
	if err != nil {
		return err
	}

	cmp := compareVersions(current, tag)
	updateAvailable := cmp < 0

	if opts.check {
		return f.IO.Render(renderPayload{
			Path:            binPath,
			Current:         current,
			Latest:          tag,
			UpdateAvailable: updateAvailable,
			Action:          "checked",
		}, func() error {
			return f.IO.Object([]output.KV{
				{Key: "Current", Value: current},
				{Key: "Latest", Value: tag},
				{Key: "Status", Value: statusLabel(cmp)},
			})
		})
	}

	if cmp >= 0 && !opts.force {
		msg := fmt.Sprintf("Already on %s; nothing to upgrade.", current)
		if cmp > 0 {
			msg = fmt.Sprintf("Current %s is ahead of latest release %s; nothing to upgrade.", current, tag)
		}
		f.IO.Println(msg)
		return f.IO.Render(renderPayload{
			Path:            binPath,
			Current:         current,
			Latest:          tag,
			Installed:       current,
			UpdateAvailable: false,
			Action:          "noop",
		}, func() error { return nil })
	}

	if err := install(ctx, client, f.IO, tag, binPath); err != nil {
		return err
	}

	f.IO.Println(fmt.Sprintf("Upgraded %s -> %s", current, tag))
	return f.IO.Render(renderPayload{
		Path:            binPath,
		Current:         current,
		Latest:          tag,
		Installed:       tag,
		UpdateAvailable: false,
		Action:          "upgraded",
	}, func() error {
		return f.IO.Object([]output.KV{
			{Key: "Path", Value: binPath},
			{Key: "Previous", Value: current},
			{Key: "Installed", Value: tag},
		})
	})
}

func resolveBinaryPath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return filepath.Clean(p), nil
}

// compareVersions defers to golang.org/x/mod/semver. Both sides are coerced
// to a leading "v"; invalid input (e.g. the "dev" / "none" build defaults
// or `git describe` outputs that include a "-dirty" suffix without a
// preceding numeric segment) compares less than any valid version, so
// unstamped binaries always accept an upgrade.
func compareVersions(current, target string) int {
	return semver.Compare(ensureV(current), ensureV(target))
}

func ensureV(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func statusLabel(cmp int) string {
	switch {
	case cmp < 0:
		return "update available"
	case cmp > 0:
		return "ahead of latest"
	}
	return "up to date"
}

// resolveTargetTag returns the canonical "vX.Y.Z" tag of the release to
// install. An empty requested value resolves "latest" via GitHub's
// unauthenticated 302 from /releases/latest, whose Location header points
// at /releases/tag/<tag>.
func resolveTargetTag(ctx context.Context, client *req.Client, requested string) (string, error) {
	if requested != "" {
		if !strings.HasPrefix(requested, "v") {
			requested = "v" + requested
		}
		return requested, nil
	}
	url := fmt.Sprintf("https://github.com/%s/releases/latest", version.Current.RepoSlug)
	noRedirect := client.Clone().SetRedirectPolicy(req.NoRedirectPolicy())
	resp, err := noRedirect.R().SetContext(ctx).Head(url)
	if err != nil && !isExpectedRedirectError(err) {
		return "", fmt.Errorf("query latest release: %w", err)
	}
	if resp.Response == nil {
		return "", errors.New("empty response resolving latest release")
	}
	if resp.StatusCode/100 != 3 {
		return "", fmt.Errorf("unexpected status %d resolving latest release", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", errors.New("no Location header from GitHub for latest release")
	}
	tag := loc[strings.LastIndex(loc, "/")+1:]
	if tag == "" || tag == "releases" {
		return "", fmt.Errorf("could not parse tag from %q", loc)
	}
	return tag, nil
}

// isExpectedRedirectError ignores the synthetic error req returns when
// NoRedirectPolicy stops a 3xx; the response object is still usable.
func isExpectedRedirectError(err error) bool {
	if err == nil {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "redirect is disabled") || strings.Contains(msg, "stopped after")
}

func install(ctx context.Context, client *req.Client, io *output.Renderer, tag, binPath string) error {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	archiveExt := ".tar.gz"
	if osName == "windows" {
		archiveExt = ".zip"
	}
	suffix := fmt.Sprintf("_%s_%s%s", osName, arch, archiveExt)
	base := fmt.Sprintf("https://github.com/%s/releases/download/%s", version.Current.RepoSlug, tag)

	io.Println(fmt.Sprintf("Resolving %s", tag))
	sumsResp, err := client.R().SetContext(ctx).Get(base + "/checksums.txt")
	if err != nil {
		return fmt.Errorf("download checksums.txt: %w", err)
	}
	if sumsResp.StatusCode/100 != 2 {
		return fmt.Errorf("download checksums.txt: status %d", sumsResp.StatusCode)
	}
	asset, expected, ok := findAsset(sumsResp.String(), suffix)
	if !ok {
		return fmt.Errorf("no asset matching %s/%s in %s checksums.txt", osName, arch, tag)
	}

	body, err := downloadWithProgress(ctx, client, base+"/"+asset, fmt.Sprintf("Downloading %s", asset), io)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset, err)
	}
	sum := sha256.Sum256(body)
	got := hex.EncodeToString(sum[:])
	if got != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", asset, expected, got)
	}

	binInArchive := binaryName
	if osName == "windows" {
		binInArchive = binaryName + ".exe"
	}
	extracted, err := extractBinary(body, asset, binInArchive)
	if err != nil {
		return fmt.Errorf("extract %s: %w", binInArchive, err)
	}

	io.Println(fmt.Sprintf("Installing to %s", binPath))
	return replace(binPath, extracted)
}

// findAsset selects the row in checksums.txt whose filename ends with the
// host suffix (e.g. "_linux_arm64.tar.gz"). Goreleaser writes "<sha>  <name>"
// rows; some toolchains prefix names with "*" for binary mode.
func findAsset(sums, suffix string) (name, hash string, ok bool) {
	for line := range strings.SplitSeq(sums, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		n := strings.TrimPrefix(fields[1], "*")
		if strings.HasSuffix(n, suffix) {
			return n, fields[0], true
		}
	}
	return "", "", false
}

func extractBinary(archive []byte, assetName, binName string) ([]byte, error) {
	if strings.HasSuffix(assetName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) != binName {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			return data, err
		}
		return nil, fmt.Errorf("%s not found in zip", binName)
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == binName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in tar.gz", binName)
}

// replace writes payload into a sibling temp file and renames it over
// target. On Unix the rename swaps inodes atomically while the running
// process keeps the old image. On Windows the running .exe cannot be
// deleted, so we rename it aside first and try to clean up after the
// next launch.
func replace(target string, payload []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".new-*")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("cannot write to %s (try with elevated permissions or reinstall via the install script): %w", dir, err)
		}
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if runtime.GOOS == "windows" {
		backup := target + ".old"
		_ = os.Remove(backup)
		if err := os.Rename(target, backup); err != nil {
			cleanup()
			return fmt.Errorf("rename current binary: %w", err)
		}
		if err := os.Rename(tmpPath, target); err != nil {
			_ = os.Rename(backup, target)
			return fmt.Errorf("install new binary: %w", err)
		}
		return nil
	}
	if err := os.Rename(tmpPath, target); err != nil {
		cleanup()
		return fmt.Errorf("install new binary: %w", err)
	}
	return nil
}
