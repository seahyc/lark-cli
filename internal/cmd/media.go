package cmd

import (
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"os"
)

// buildMediaContent builds the JSON content body for a video message
// (msg_type=media). Lark requires file_key plus a cover image_key; without the
// cover the message is stored as msg_type=nonsupport ("This message type is
// currently not supported"). file_name and duration are optional metadata.
func buildMediaContent(fileKey, imageKey, fileName string, durationMs int) (string, error) {
	payload := map[string]interface{}{"file_key": fileKey}
	if imageKey != "" {
		payload["image_key"] = imageKey
	}
	if fileName != "" {
		payload["file_name"] = fileName
	}
	if durationMs > 0 {
		payload["duration"] = durationMs
	}
	b, err := json.Marshal(payload)
	return string(b), err
}

// buildAudioContent builds the JSON content body for an audio message
// (msg_type=audio). duration is optional (Lark derives it when omitted).
func buildAudioContent(fileKey string, durationMs int) (string, error) {
	payload := map[string]interface{}{"file_key": fileKey}
	if durationMs > 0 {
		payload["duration"] = durationMs
	}
	b, err := json.Marshal(payload)
	return string(b), err
}

// generatePlaceholderCover writes a plain dark cover JPEG to a temp file and
// returns its path. Used as the video cover when no --cover is supplied; Lark
// overlays its own play button on the thumbnail, so a solid frame is enough.
// The caller is responsible for removing the returned file.
func generatePlaceholderCover() (string, error) {
	const w, h = 1280, 720
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0x20, 0x24, 0x2e, 0xff}}, image.Point{}, draw.Src)
	tmp, err := os.CreateTemp("", "lark-cover-*.jpg")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	if err := jpeg.Encode(tmp, img, &jpeg.Options{Quality: 80}); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// mp4DurationMs is a best-effort parse of an mp4/mov file's duration in
// milliseconds by walking the moov/mvhd boxes. Returns 0 when it can't be
// determined (e.g. a non-mp4 container), in which case the caller omits it.
func mp4DurationMs(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0
	}
	moovStart, moovSize, ok := findBox(f, 0, st.Size(), "moov")
	if !ok {
		return 0
	}
	mvhdStart, _, ok := findBox(f, moovStart+8, moovStart+moovSize, "mvhd")
	if !ok {
		return 0
	}
	ver := make([]byte, 1)
	if _, err := f.ReadAt(ver, mvhdStart+8); err != nil {
		return 0
	}
	var timescale uint32
	var duration uint64
	if ver[0] == 1 {
		buf := make([]byte, 16)
		if _, err := f.ReadAt(buf, mvhdStart+8+20); err != nil {
			return 0
		}
		timescale = binary.BigEndian.Uint32(buf[0:4])
		duration = binary.BigEndian.Uint64(buf[4:12])
	} else {
		buf := make([]byte, 8)
		if _, err := f.ReadAt(buf, mvhdStart+8+12); err != nil {
			return 0
		}
		timescale = binary.BigEndian.Uint32(buf[0:4])
		duration = uint64(binary.BigEndian.Uint32(buf[4:8]))
	}
	if timescale == 0 {
		return 0
	}
	return int(duration * 1000 / uint64(timescale))
}

// findBox scans sibling ISO-BMFF boxes in [start, end) and returns the start
// offset and total size of the first box of the given type.
func findBox(r io.ReaderAt, start, end int64, boxType string) (int64, int64, bool) {
	pos := start
	hdr := make([]byte, 8)
	for pos+8 <= end {
		if _, err := r.ReadAt(hdr, pos); err != nil {
			return 0, 0, false
		}
		size := int64(binary.BigEndian.Uint32(hdr[0:4]))
		hdrLen := int64(8)
		if size == 1 {
			large := make([]byte, 8)
			if _, err := r.ReadAt(large, pos+8); err != nil {
				return 0, 0, false
			}
			size = int64(binary.BigEndian.Uint64(large))
			hdrLen = 16
		} else if size == 0 {
			size = end - pos
		}
		if size < hdrLen {
			return 0, 0, false
		}
		if string(hdr[4:8]) == boxType {
			return pos, size, true
		}
		pos += size
	}
	return 0, 0, false
}
