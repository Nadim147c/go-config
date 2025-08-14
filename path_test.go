package config_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/Nadim147c/go-config"
)

func TestFindPath(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty path", "/base", "", "", true},
		{"absolute path", "/base", "/abs/path", "/abs/path", false},
		{"relative path", "/base", "rel/path", "/base/rel/path", false},
		{"tilde expansion", "/base", "~/file.txt", "/home/<username>/file.txt", false},
		{"home var expansion", "/base", "$HOME/file.txt", "/home/<username>/file.txt", false},
		{"xdg config home", "/base", "$XDG_CONFIG_HOME/app.conf", "/home/<username>/.config/app.conf", false},
		{"xdg cache home", "/base", "$XDG_CACHE_HOME/cache", "/home/<username>/.cache/cache", false},
		{"xdg data home", "/base", "$XDG_DATA_HOME/data", "/home/<username>/.local/share/data", false},
		{"tmpdir", "/base", "$TMPDIR/temp.txt", os.TempDir() + "/temp.txt", false},
		{"pwd", "/base", "$PWD/file.txt", must(os.Getwd()) + "/file.txt", false},
		{"xdg desktop dir", "/base", "$XDG_DESKTOP_DIR/icon.png", "/home/<username>/Desktop/icon.png", false},
		{"xdg documents dir", "/base", "$XDG_DOCUMENTS_DIR/doc.txt", "/home/<username>/Documents/doc.txt", false},
		{"xdg downloads dir", "/base", "$XDG_DOWNLOAD_DIR/file.bin", "/home/<username>/Downloads/file.bin", false},
		{"xdg music dir", "/base", "$XDG_MUSIC_DIR/song.mp3", "/home/<username>/Music/song.mp3", false},
		{"xdg pictures dir", "/base", "$XDG_PICTURES_DIR/image.jpg", "/home/<username>/Pictures/image.jpg", false},
		{"xdg publicshare dir", "/base", "$XDG_PUBLICSHARE_DIR/readme.txt", "/home/<username>/Public/readme.txt", false},
		{"xdg templates dir", "/base", "$XDG_TEMPLATES_DIR/template.txt", "/home/<username>/Templates/template.txt", false},
		{"xdg videos dir", "/base", "$XDG_VIDEOS_DIR/movie.mp4", "/home/<username>/Videos/movie.mp4", false},
	}

	var username string
	if runtime.GOOS == "windows" {
		username = os.Getenv("USERNAME")
	} else {
		username = os.Getenv("USER")
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.FindPath(tt.base, tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FindPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			want := strings.Replace(tt.want, "<username>", username, 1)
			if got != want {
				t.Errorf("FindPath() = %v, want %v", got, want)
			}
		})
	}
}
