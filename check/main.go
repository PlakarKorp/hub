/*
 * Copyright (c) 2026 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

// Command check verifies every recipe.yaml in the tree: the fields are well
// formed, the checksum names the commit the tag resolves to upstream right
// now, the hash algorithm in its prefix is the one the repository uses, and
// source agrees with the repository's visibility on GitHub.
//
//	GITHUB_TOKEN=$(gh auth token) go run ./check
//
// The token reads the private repositories enterprise recipes point at.
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/semver"
)

// maxRecipe bounds how much of a recipe file is read.
const maxRecipe = 64 << 10

var (
	nameRe     = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	handleRe   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	checksumRe = regexp.MustCompile(`^(sha1|sha256):([0-9a-f]+)$`)
	hashAlgo   = map[int]string{40: "sha1", 64: "sha256"}

	// What krossbuild can target; a platform is "<os>/<arch>".
	knownOS   = []string{"darwin", "freebsd", "linux", "netbsd", "openbsd", "windows"}
	knownArch = []string{"386", "amd64", "arm", "arm64", "riscv64"}
)

type recipe struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
	Checksum   string `yaml:"checksum"`
	// Platforms lists the <os>/<arch> pairs the plugin is built for; absent
	// means every target.
	Platforms []string `yaml:"platforms"`
	// Source is "open" or "proprietary" and must match the repository.
	Source     string `yaml:"source"`
	Maintainer string `yaml:"maintainer"` // a GitHub handle or organization
	Contact    string `yaml:"contact"`    // where users bring problems: an email address or an https URL
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "check: %v\n", err)
		os.Exit(2)
	}
	paths := os.Args[1:]
	if len(paths) == 0 {
		paths, err = findRecipes(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "check: %v\n", err)
			os.Exit(2)
		}
	}

	tags := newTagCache(os.Getenv("GITHUB_TOKEN"))
	failed := false
	for _, p := range paths {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			rel = p
		}
		if errs := checkRecipe(p, tags); len(errs) > 0 {
			failed = true
			for _, err := range errs {
				fmt.Printf("FAIL %s: %v\n", rel, err)
			}
			continue
		}
		fmt.Printf("ok   %s\n", rel)
	}
	if failed {
		os.Exit(1)
	}
}

// findRecipes returns every <tier>/<compat>/<name>/recipe.yaml under root.
func findRecipes(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".github") {
			return filepath.SkipDir
		}
		if d.Type().IsRegular() && d.Name() == "recipe.yaml" {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

func checkRecipe(path string, tags *tagCache) []error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	r, err := load(path)
	if err != nil {
		return []error{err}
	}

	// The recipe sits at <tier>/<compat>/<name>/recipe.yaml.
	dir := filepath.Dir(path)
	compat := filepath.Base(filepath.Dir(dir))
	if !semver.IsValid(compat) {
		fail("directory %s is not an API version", compat)
	}
	if r.Name != filepath.Base(dir) {
		fail("recipe is named %q but lives in %s/", r.Name, filepath.Base(dir))
	}
	if !nameRe.MatchString(r.Name) {
		fail("name %q is not a plugin name", r.Name)
	}

	// version is the git tag, "<name>/vX.Y.Z" in a mono-repo, "vX.Y.Z" otherwise.
	subdir, ver := ".", r.Version
	if i := strings.LastIndex(r.Version, "/"); i >= 0 {
		subdir, ver = r.Version[:i], r.Version[i+1:]
	}
	if !semver.IsValid(ver) || !strings.HasPrefix(ver, "v") {
		fail("version %q does not end in a semantic version with a v prefix", r.Version)
	}
	if subdir != "." && subdir != r.Name {
		fail("version %q is prefixed with %q, not the plugin name", r.Version, subdir)
	}
	if !strings.HasPrefix(r.Repository, "https://github.com/") {
		fail("repository %q is not an https://github.com URL", r.Repository)
	}

	switch r.Source {
	case "open", "proprietary":
	case "":
		fail("source is required: open or proprietary")
	default:
		fail("source %q is neither open nor proprietary", r.Source)
	}
	if !handleRe.MatchString(r.Maintainer) {
		fail("maintainer %q is not a GitHub handle", r.Maintainer)
	}
	if r.Contact == "" {
		fail("contact is required: an email address or an https URL")
	} else if !isContact(r.Contact) {
		fail("contact %q is neither an email address nor a URL", r.Contact)
	}
	for i, p := range r.Platforms {
		goos, goarch, ok := strings.Cut(p, "/")
		if !ok || !slices.Contains(knownOS, goos) || !slices.Contains(knownArch, goarch) {
			fail("platform %q is not a known <os>/<arch>", p)
		}
		if slices.Contains(r.Platforms[:i], p) {
			fail("platform %s listed twice", p)
		}
	}

	m := checksumRe.FindStringSubmatch(r.Checksum)
	if r.Checksum == "" {
		fail("no checksum; expected the commit %s resolves to, as sha1:<hex> or sha256:<hex>", r.Version)
	} else if m == nil {
		fail("checksum %q is not sha1:<hex> or sha256:<hex>", r.Checksum)
	} else if hashAlgo[len(m[2])] != m[1] {
		fail("checksum says %s but holds %d hex digits", m[1], len(m[2]))
	}
	if len(errs) > 0 {
		return errs
	}

	// What the tag points at upstream, right now.
	sha, err := tags.resolve(r.Repository, r.Version)
	if err != nil {
		return []error{err}
	}
	if algo := hashAlgo[len(sha)]; algo != m[1] {
		fail("repository uses %s hashes, checksum is %s", algo, m[1])
	} else if sha != m[2] {
		fail("tag %s is %s upstream, checksum says %s", r.Version, sha[:12], m[2][:12])
	}

	// source must say what the repository's visibility says.
	private, err := tags.private(r.Repository)
	if err != nil {
		fail("%v", err)
	} else if private && r.Source != "proprietary" {
		fail("repository is private but source says %s", r.Source)
	} else if !private && r.Source != "open" {
		fail("repository is public but source says %s", r.Source)
	}
	return errs
}

// isContact accepts what plakar's manifest accepts: an email address, with
// or without mailto:, or an https URL.
func isContact(s string) bool {
	if _, err := mail.ParseAddress(strings.TrimPrefix(s, "mailto:")); err == nil {
		return true
	}
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "https" && u.Host != ""
}

func load(path string) (*recipe, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	var r recipe
	dec := yaml.NewDecoder(io.LimitReader(fp, maxRecipe))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return &r, nil
}

// tagCache asks each repository for its tags, and its visibility, once.
type tagCache struct {
	token   string
	client  *http.Client
	repos   map[string]map[string]string // repository -> ref -> sha
	fails   map[string]error
	privacy map[string]bool
}

func newTagCache(token string) *tagCache {
	return &tagCache{
		token:   token,
		client:  &http.Client{Timeout: 30 * time.Second},
		repos:   map[string]map[string]string{},
		fails:   map[string]error{},
		privacy: map[string]bool{},
	}
}

// private asks the GitHub API whether the repository is private; an internal
// repository counts as private, which is what matters to a user.
func (c *tagCache) private(repository string) (bool, error) {
	if p, ok := c.privacy[repository]; ok {
		return p, nil
	}
	ownerRepo := strings.TrimPrefix(strings.TrimSuffix(repository, ".git"), "https://github.com/")
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+ownerRepo, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "hub-check")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("github: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("github: %s: %s", ownerRepo, resp.Status)
	}
	var info struct {
		Private bool `json:"private"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRecipe)).Decode(&info); err != nil {
		return false, fmt.Errorf("github: %s: %w", ownerRepo, err)
	}
	c.privacy[repository] = info.Private
	return info.Private, nil
}

func (c *tagCache) resolve(repository, tag string) (string, error) {
	refs, err := c.tags(repository)
	if err != nil {
		return "", err
	}
	ref := "refs/tags/" + tag
	if sha, ok := refs[ref+"^{}"]; ok {
		return sha, nil
	}
	if sha, ok := refs[ref]; ok {
		return sha, nil
	}
	return "", fmt.Errorf("tag %s does not exist in %s", tag, repository)
}

// tags lists a repository's tags with git ls-remote, peeling annotated ones.
// It runs outside any checkout with the machine's git config left out, so
// the URL means what it says and no stored credential, such as the one
// actions/checkout writes into .git/config, is sent alongside the token.
func (c *tagCache) tags(repository string) (map[string]string, error) {
	if refs, ok := c.repos[repository]; ok {
		return refs, nil
	}
	if err, ok := c.fails[repository]; ok {
		return nil, err
	}

	args := []string{}
	if c.token != "" {
		auth := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + c.token))
		args = append(args, "-c", "http.https://github.com/.extraheader=Authorization: Basic "+auth)
	}
	args = append(args, "ls-remote", "--tags", repository)
	cmd := exec.Command("git", args...)
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		err = fmt.Errorf("ls-remote %s: %w: %s", repository, err, strings.TrimSpace(stderr.String()))
		c.fails[repository] = err
		return nil, err
	}

	refs := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		if sha, ref, ok := strings.Cut(sc.Text(), "\t"); ok {
			refs[ref] = sha
		}
	}
	c.repos[repository] = refs
	return refs, nil
}
