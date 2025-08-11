package config

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/adrg/xdg"
)

var separator = regexp.MustCompile(`[/\\]+`)

// FindPath expands environment-aware variables and returns an absolute path
// from a given base path (not CWD).
// Path Prefix Expands:
//   - $HOME or ~                  : xdg.Home
//   - $XDG_CONFIG_HOME             : xdg.ConfigHome
//   - $XDG_CACHE_HOME              : xdg.CacheHome
//   - $XDG_DATA_HOME               : xdg.DataHome
//   - $TMPDIR                      : os.TempDir()
//   - $PWD                         : Current Working Directorie
//   - $XDG_DESKTOP_DIR             : xdg.UserDirs.Desktop
//   - $XDG_DOCUMENTS_DIR           : xdg.UserDirs.Documents
//   - $XDG_DOWNLOAD_DIR            : xdg.UserDirs.Download
//   - $XDG_MUSIC_DIR               : xdg.UserDirs.Music
//   - $XDG_PICTURES_DIR            : xdg.UserDirs.Pictures
//   - $XDG_PUBLICSHARE_DIR         : xdg.UserDirs.PublicShare
//   - $XDG_TEMPLATES_DIR           : xdg.UserDirs.Templates
//   - $XDG_VIDEOS_DIR              : xdg.UserDirs.Videos
func FindPath(base, input string) (string, error) {
	if input == "" {
		return "", errors.New("empty path")
	}

	if filepath.IsAbs(input) {
		return filepath.Clean(input), nil
	}

	split := separator.Split(input, 2)
	if len(split) != 2 {
		joined := filepath.Join(base, input)
		return filepath.Clean(joined), nil
	}

	parent, rest := split[0], split[1]

	var path string
	switch parent {
	case "~", "$HOME":
		path = filepath.Join(xdg.Home, rest)

	// XDG based directories
	case "$XDG_CONFIG_HOME", "${XDG_CONFIG_HOME}":
		path = filepath.Join(xdg.ConfigHome, rest)
	case "$XDG_CACHE_HOME", "${XDG_CACHE_HOME}":
		path = filepath.Join(xdg.CacheHome, rest)
	case "$XDG_DATA_HOME", "${XDG_DATA_HOME}":
		path = filepath.Join(xdg.DataHome, rest)

	// System-related
	case "$TMPDIR", "${TMPDIR}":
		path = filepath.Join(os.TempDir(), rest)
	case "$PWD", "${PWD}":
		return filepath.Abs(rest)

	// XDG user directories
	case "$XDG_DESKTOP_DIR", "${XDG_DESKTOP_DIR}":
		path = filepath.Join(xdg.UserDirs.Desktop, rest)
	case "$XDG_DOCUMENTS_DIR", "${XDG_DOCUMENTS_DIR}":
		path = filepath.Join(xdg.UserDirs.Documents, rest)
	case "$XDG_DOWNLOAD_DIR", "${XDG_DOWNLOAD_DIR}":
		path = filepath.Join(xdg.UserDirs.Download, rest)
	case "$XDG_MUSIC_DIR", "${XDG_MUSIC_DIR}":
		path = filepath.Join(xdg.UserDirs.Music, rest)
	case "$XDG_PICTURES_DIR", "${XDG_PICTURES_DIR}":
		path = filepath.Join(xdg.UserDirs.Pictures, rest)
	case "$XDG_PUBLICSHARE_DIR", "${XDG_PUBLICSHARE_DIR}":
		path = filepath.Join(xdg.UserDirs.PublicShare, rest)
	case "$XDG_TEMPLATES_DIR", "${XDG_TEMPLATES_DIR}":
		path = filepath.Join(xdg.UserDirs.Templates, rest)
	case "$XDG_VIDEOS_DIR", "${XDG_VIDEOS_DIR}":
		path = filepath.Join(xdg.UserDirs.Videos, rest)

	default:
		path = filepath.Join(base, input)
	}

	return filepath.Clean(path), nil
}
