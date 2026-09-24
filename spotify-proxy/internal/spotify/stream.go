package spotify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/devgianlu/go-librespot/audio"
	storagepb "github.com/devgianlu/go-librespot/proto/spotify/download"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
)

// preferredAudioOrder prioriza OGG Vorbis (lo reproduce el browser nativo)
// y evita MP3_160_ENC (variante encriptada distinta).
var preferredAudioOrder = []metadatapb.AudioFile_Format{
	metadatapb.AudioFile_OGG_VORBIS_160,
	metadatapb.AudioFile_OGG_VORBIS_96,
	metadatapb.AudioFile_MP3_160,
	metadatapb.AudioFile_MP3_256,
	metadatapb.AudioFile_MP3_96,
	metadatapb.AudioFile_OGG_VORBIS_320,
	metadatapb.AudioFile_MP3_320,
}

func pickAudioFile(track *metadatapb.Track) *metadatapb.AudioFile {
	byFormat := make(map[metadatapb.AudioFile_Format]*metadatapb.AudioFile, len(track.GetFile()))
	for _, f := range track.GetFile() {
		if f.GetFileId() == nil {
			continue
		}
		if _, ok := byFormat[f.GetFormat()]; !ok {
			byFormat[f.GetFormat()] = f
		}
	}
	for _, want := range preferredAudioOrder {
		if f, ok := byFormat[want]; ok {
			return f
		}
	}
	return nil
}

func audioContentType(f *metadatapb.AudioFile) string {
	switch f.GetFormat() {
	case metadatapb.AudioFile_OGG_VORBIS_96,
		metadatapb.AudioFile_OGG_VORBIS_160,
		metadatapb.AudioFile_OGG_VORBIS_320:
		return "audio/ogg"
	default:
		return "audio/mpeg"
	}
}

// cdnReaderAt lee por rangos un archivo del CDN de Spotify.
type cdnReaderAt struct {
	client *http.Client
	url    string
	size   int64
}

func (c *cdnReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= c.size {
		return 0, io.EOF
	}
	end := off + int64(len(p)) - 1
	if end >= c.size {
		end = c.size - 1
	}
	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, end))
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("cdn status %d", resp.StatusCode)
	}
	n, err := io.ReadFull(resp.Body, p[:end-off+1])
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	return n, err
}

// decryptSeeker adapta el Decryptor (ReaderAt) a ReadSeeker para ServeContent.
type decryptSeeker struct {
	dec  *audio.Decryptor
	size int64
	pos  int64
}

func (s *decryptSeeker) Read(p []byte) (int, error) {
	if s.pos >= s.size {
		return 0, io.EOF
	}
	n, err := s.dec.ReadAt(p, s.pos)
	s.pos += int64(n)
	return n, err
}

func (s *decryptSeeker) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = s.pos + offset
	case io.SeekEnd:
		abs = s.size + offset
	default:
		return 0, fmt.Errorf("whence inválido")
	}
	if abs < 0 {
		return 0, fmt.Errorf("offset negativo")
	}
	s.pos = abs
	return abs, nil
}

// streamTrack vuelca el audio completo del track: metadata → mejor archivo →
// audio key (con el token Premium) → URL del CDN → decrypt AES → HTTP Range.
func streamTrack(ctx context.Context, sess *SpSession, uri string, w http.ResponseWriter, r *http.Request) error {
	if !strings.HasPrefix(uri, "spotify:track:") {
		return fmt.Errorf("uri inválida")
	}

	track, err := fetchTrack(ctx, sess, uri)
	if err != nil {
		return err
	}
	file := pickAudioFile(track)
	if file == nil {
		return fmt.Errorf("sin archivo de audio disponible")
	}

	key, err := sess.Keys.Request(ctx, track.GetGid(), file.GetFileId())
	if err != nil {
		return fmt.Errorf("audio key: %w", err)
	}

	format := file.GetFormat()
	storage, err := sess.Sp.ResolveStorageInteractive(ctx, file.GetFileId(), &format, false)
	if err != nil {
		return fmt.Errorf("storage resolve: %w", err)
	}
	if storage.GetResult() != storagepb.StorageResolveResponse_CDN || len(storage.GetCdnurl()) == 0 {
		return fmt.Errorf("storage no disponible: %v", storage.GetResult())
	}

	client := &http.Client{Timeout: 30 * time.Second}
	cdn := &cdnReaderAt{client: client, url: storage.GetCdnurl()[0]}
	cdn.size, err = cdnSize(client, cdn.url)
	if err != nil {
		return fmt.Errorf("cdn size: %w", err)
	}

	dec, err := audio.NewAesAudioDecryptor(cdn, key)
	if err != nil {
		return fmt.Errorf("decryptor: %w", err)
	}

	w.Header().Set("Content-Type", audioContentType(file))
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, track.GetName()+".ogg", time.Now(), &decryptSeeker{dec: dec, size: cdn.size})
	return nil
}

func cdnSize(client *http.Client, url string) (int64, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.ContentLength > 0 {
		return resp.ContentLength, nil
	}
	// Fallback: primer byte para leer el total del Content-Range
	req2, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req2.Header.Set("Range", "bytes=0-0")
	resp2, err := client.Do(req2)
	if err != nil {
		return 0, err
	}
	defer resp2.Body.Close()
	var start, end, size int64
	if _, err := fmt.Sscanf(resp2.Header.Get("Content-Range"), "bytes %d-%d/%d", &start, &end, &size); err != nil {
		return 0, fmt.Errorf("sin Content-Range")
	}
	return size, nil
}
