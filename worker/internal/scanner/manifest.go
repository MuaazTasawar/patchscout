package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MuaazTasawar/patchscout-worker/internal/models"
)

// DiscoverManifests walks a cloned repo looking for supported manifest
// files (go.mod, requirements.txt/pyproject.toml, pubspec.yaml — v1 scope
// per the project spec) and parses each into a models.Manifest.
func DiscoverManifests(repoPath string) ([]models.Manifest, error) {
	var manifests []models.Manifest

	targets := map[string]string{
		"go.mod":           "go",
		"requirements.txt": "python",
		"pyproject.toml":   "python",
		"pubspec.yaml":     "flutter",
	}

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip vendor/node_modules-style directories to keep the walk fast.
			base := filepath.Base(path)
			if base == "vendor" || base == "node_modules" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		ecosystem, ok := targets[info.Name()]
		if !ok {
			return nil
		}

		var pkgs []models.Package
		var parseErr error

		switch info.Name() {
		case "go.mod":
			pkgs, parseErr = parseGoMod(path)
		case "requirements.txt":
			pkgs, parseErr = parseRequirementsTxt(path)
		case "pyproject.toml":
			pkgs, parseErr = parsePyprojectToml(path)
		case "pubspec.yaml":
			pkgs, parseErr = parsePubspecYaml(path)
		}

		if parseErr != nil {
			// Don't fail the whole scan for one unparseable manifest —
			// skip it and continue with the rest of the repo.
			return nil
		}

		rel, _ := filepath.Rel(repoPath, path)
		manifests = append(manifests, models.Manifest{
			Ecosystem: ecosystem,
			Path:      rel,
			Packages:  pkgs,
		})

		return nil
	})

	return manifests, err
}

var goRequireLineRe = regexp.MustCompile(`^\s*([a-zA-Z0-9\.\-_/]+)\s+v([0-9]+\.[0-9]+\.[0-9]+[a-zA-Z0-9\-\.\+]*)`)

func parseGoMod(path string) ([]models.Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []models.Package
	inRequireBlock := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		var target string
		if inRequireBlock {
			target = line
		} else if strings.HasPrefix(line, "require ") {
			target = strings.TrimPrefix(line, "require ")
		} else {
			continue
		}

		match := goRequireLineRe.FindStringSubmatch(target)
		if match != nil {
			pkgs = append(pkgs, models.Package{Name: match[1], Version: match[2]})
		}
	}

	return pkgs, scanner.Err()
}

var requirementsLineRe = regexp.MustCompile(`^([a-zA-Z0-9\-_\.]+)\s*==\s*([a-zA-Z0-9\.\-_]+)`)

func parseRequirementsTxt(path string) ([]models.Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []models.Package
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		match := requirementsLineRe.FindStringSubmatch(line)
		if match != nil {
			pkgs = append(pkgs, models.Package{Name: match[1], Version: match[2]})
		}
	}

	return pkgs, scanner.Err()
}

var pyprojectDepLineRe = regexp.MustCompile(`^([a-zA-Z0-9\-_\.]+)\s*=\s*"([\^~=]?[0-9][a-zA-Z0-9\.\-_]*)"`)

func parsePyprojectToml(path string) ([]models.Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []models.Package
	inDeps := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[tool.poetry.dependencies]") || strings.HasPrefix(line, "[project.dependencies]") {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") && inDeps {
			inDeps = false
			continue
		}
		if !inDeps {
			continue
		}

		match := pyprojectDepLineRe.FindStringSubmatch(line)
		if match != nil && match[1] != "python" {
			version := strings.TrimLeft(match[2], "^~=")
			pkgs = append(pkgs, models.Package{Name: match[1], Version: version})
		}
	}

	return pkgs, scanner.Err()
}

var pubspecDepLineRe = regexp.MustCompile(`^\s{2}([a-zA-Z0-9_]+):\s*\^?([0-9][a-zA-Z0-9\.\-_+]*)\s*$`)

func parsePubspecYaml(path string) ([]models.Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []models.Package
	inDeps := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "dependencies:" {
			inDeps = true
			continue
		}
		if inDeps && trimmed != "" && !strings.HasPrefix(raw, "  ") {
			inDeps = false
			continue
		}
		if !inDeps {
			continue
		}

		match := pubspecDepLineRe.FindStringSubmatch(raw)
		if match != nil {
			pkgs = append(pkgs, models.Package{Name: match[1], Version: match[2]})
		}
	}

	return pkgs, scanner.Err()
}
