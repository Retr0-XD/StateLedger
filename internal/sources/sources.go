package sources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Retr0-XD/StateLedger/internal/collectors"
)

type CaptureResult struct {
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Payload string `json:"payload"`
	Error   string `json:"error,omitempty"`
}

func CaptureGit(repoPath string) (collectors.CodePayload, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		repoPath = "."
	}

	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return collectors.CodePayload{}, errors.New("not a git repository")
	}

	repo, err := getGitRepoName(repoPath)
	if err != nil {
		return collectors.CodePayload{}, err
	}

	commit, err := getGitCommit(repoPath)
	if err != nil {
		return collectors.CodePayload{}, err
	}

	return collectors.CodePayload{
		Repo:   repo,
		Commit: commit,
	}, nil
}

func CaptureEnvironment() (collectors.EnvironmentPayload, error) {
	return collectors.EnvironmentPayload{
		OS:         runtime.GOOS,
		Kernel:     runtime.GOARCH,
		Runtime:    runtime.Version(),
		Arch:       runtime.GOARCH,
		TimeSource: "system",
	}, nil
}

func CaptureConfig(source string) (collectors.ConfigPayload, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return collectors.ConfigPayload{}, errors.New("config source path required")
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return collectors.ConfigPayload{}, err
	}

	snapshot := string(data)
	hash := computeConfigHash(snapshot)

	return collectors.ConfigPayload{
		Source:   source,
		Version:  "1",
		Hash:     hash,
		Snapshot: snapshot,
	}, nil
}

func getGitRepoName(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "config", "--get", "remote.origin.url")
	output, err := cmd.CombinedOutput()
	if err != nil {
		cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--show-toplevel")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", err
		}
		return filepath.Base(strings.TrimSpace(string(output))), nil
	}
	url := strings.TrimSpace(string(output))
	if strings.HasSuffix(url, ".git") {
		url = url[:len(url)-4]
	}
	return filepath.Base(url), nil
}

func getGitCommit(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func computeConfigHash(snapshot string) string {
	sum := sha256.Sum256([]byte(snapshot))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func CaptureFromManifest(kind, source string, params map[string]string) (CaptureResult, error) {
	var payload any
	var err error

	switch kind {
	case "code":
		payload, err = CaptureGit(source)
	case "config":
		payload, err = CaptureConfig(source)
	case "environment":
		payload, err = CaptureEnvironment()
	default:
		return CaptureResult{}, errors.New("unsupported capture kind: " + kind)
	}

	if err != nil {
		return CaptureResult{
			Kind:   kind,
			Source: source,
			Error:  err.Error(),
		}, nil
	}

	data, _ := json.Marshal(payload)
	return CaptureResult{
		Kind:    kind,
		Source:  source,
		Payload: string(data),
	}, nil
}
