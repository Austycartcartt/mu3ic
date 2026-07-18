package library

import (
	"os"
	"testing"
)

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		wantTitle   string
		wantArtist  string
		wantAlbum   string
		wantTrack   int
		wantYear    int
		wantArtwork bool
	}{
		{
			name:        "id3v2.3 tags with embedded artwork",
			file:        "testdata/tagged.mp3",
			wantTitle:   "Test Title",
			wantArtist:  "Test Artist",
			wantAlbum:   "Test Album",
			wantTrack:   5,
			wantYear:    2024,
			wantArtwork: true,
		},
		{
			name: "no tags at all",
			file: "testdata/untagged.mp3",
		},
		{
			name:       "flac vorbis comments, no artwork",
			file:       "testdata/tagged.flac",
			wantTitle:  "Test Title",
			wantArtist: "Test Artist",
			wantAlbum:  "Test Album",
			wantTrack:  5,
			wantYear:   2024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.file)
			if err != nil {
				t.Fatalf("opening fixture: %v", err)
			}
			defer f.Close()

			md, err := ExtractMetadata(f)
			if err != nil {
				t.Fatalf("ExtractMetadata() error = %v, want nil (unparseable files are not errors)", err)
			}

			if md.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", md.Title, tt.wantTitle)
			}
			if md.Artist != tt.wantArtist {
				t.Errorf("Artist = %q, want %q", md.Artist, tt.wantArtist)
			}
			if md.Album != tt.wantAlbum {
				t.Errorf("Album = %q, want %q", md.Album, tt.wantAlbum)
			}
			if md.TrackNumber != tt.wantTrack {
				t.Errorf("TrackNumber = %d, want %d", md.TrackNumber, tt.wantTrack)
			}
			if md.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", md.Year, tt.wantYear)
			}
			hasArtwork := md.Artwork != nil
			if hasArtwork != tt.wantArtwork {
				t.Errorf("Artwork present = %v, want %v", hasArtwork, tt.wantArtwork)
			}
			if tt.wantArtwork && md.Artwork.Ext != ".png" {
				t.Errorf("Artwork.Ext = %q, want %q", md.Artwork.Ext, ".png")
			}
		})
	}
}
