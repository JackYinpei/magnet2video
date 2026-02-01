// Package ffmpeg provides FFmpeg/FFprobe wrapper utilities for video transcoding
// Author: Done-0
// Created: 2026-01-26
package ffmpeg

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// VideoInfo contains video stream information from ffprobe
type VideoInfo struct {
	Codec      string  `json:"codec"`       // Video codec (h264, hevc, vp9, etc.)
	Width      int     `json:"width"`       // Video width
	Height     int     `json:"height"`      // Video height
	Duration   float64 `json:"duration"`    // Duration in seconds
	Bitrate    int64   `json:"bitrate"`     // Bitrate in bits/s
	FrameRate  float64 `json:"frame_rate"`  // Frame rate
	AudioCodec string  `json:"audio_codec"` // Audio codec
}

// TranscodeType represents the type of transcoding operation
type TranscodeType string

const (
	TranscodeTypeRemux     TranscodeType = "remux"     // Fast container conversion (no re-encoding)
	TranscodeTypeTranscode TranscodeType = "transcode" // Full video transcoding
	TranscodeTypeNone      TranscodeType = "none"      // No transcoding needed
)

// ProgressCallback is called during transcoding to report progress
type ProgressCallback func(progress float64)

// FFmpeg wraps FFmpeg/FFprobe commands
type FFmpeg struct {
	ffmpegPath  string
	ffprobePath string
}

// New creates a new FFmpeg wrapper
func New(ffmpegPath, ffprobePath string) *FFmpeg {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	return &FFmpeg{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}
}

// ffprobeOutput represents the JSON output structure from ffprobe
type ffprobeOutput struct {
	Streams []struct {
		Index      int    `json:"index"`
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
		BitRate    string `json:"bit_rate"`
		Tags       struct {
			Language string `json:"language"`
			Title    string `json:"title"`
		} `json:"tags"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
}

// Probe analyzes a video file and returns its information
func (f *FFmpeg) Probe(ctx context.Context, inputPath string) (*VideoInfo, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probeOutput ffprobeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &VideoInfo{}

	// Parse video stream info
	for _, stream := range probeOutput.Streams {
		if stream.CodecType == "video" && info.Codec == "" {
			info.Codec = stream.CodecName
			info.Width = stream.Width
			info.Height = stream.Height

			// Parse frame rate (e.g., "24000/1001" or "30/1")
			if stream.RFrameRate != "" {
				parts := strings.Split(stream.RFrameRate, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den > 0 {
						info.FrameRate = num / den
					}
				}
			}
		}
		if stream.CodecType == "audio" && info.AudioCodec == "" {
			info.AudioCodec = stream.CodecName
		}
	}

	// Parse duration
	if probeOutput.Format.Duration != "" {
		info.Duration, _ = strconv.ParseFloat(probeOutput.Format.Duration, 64)
	}

	// Parse bitrate
	if probeOutput.Format.BitRate != "" {
		info.Bitrate, _ = strconv.ParseInt(probeOutput.Format.BitRate, 10, 64)
	}

	return info, nil
}

// DetermineTranscodeType determines what type of transcoding is needed
func (f *FFmpeg) DetermineTranscodeType(info *VideoInfo, inputPath string) TranscodeType {
	ext := strings.ToLower(filepath.Ext(inputPath))

	// Already browser-compatible formats
	browserCompatible := map[string]bool{
		".mp4":  true,
		".webm": true,
		".m4v":  true,
	}

	// Codecs that can be played directly in browsers
	browserCodecs := map[string]bool{
		"h264": true,
		"avc1": true,
		"hevc": true,
		"h265": true,
		"vp8":  true,
		"vp9":  true,
		"av1":  true,
	}

	// If already mp4/webm with compatible codec, no transcoding needed
	if browserCompatible[ext] && browserCodecs[info.Codec] {
		return TranscodeTypeNone
	}

	// If codec is h264/h265 but container is not mp4, just remux
	if browserCodecs[info.Codec] {
		return TranscodeTypeRemux
	}

	// Otherwise, full transcode is needed
	return TranscodeTypeTranscode
}

// Remux performs container conversion without re-encoding
func (f *FFmpeg) Remux(ctx context.Context, inputPath, outputPath string, callback ProgressCallback) error {
	// Get duration for progress calculation
	info, err := f.Probe(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("failed to probe input: %w", err)
	}

	args := []string{
		"-i", inputPath,
		"-map", "0:v:0",          // Map first video stream only
		"-map", "0:a?",           // Map all audio streams (optional)
		"-c:v", "copy",           // Copy video without re-encoding
		"-c:a", "aac",            // Convert audio to AAC for browser compatibility
		"-b:a", "192k",           // Audio bitrate
		"-movflags", "+faststart", // Enable fast start for web playback
		"-y", // Overwrite output
		"-progress", "pipe:1", // Output progress to stdout
		outputPath,
	}

	return f.runWithProgress(ctx, args, info.Duration, callback)
}

// Transcode performs full video transcoding to H.264
func (f *FFmpeg) Transcode(ctx context.Context, inputPath, outputPath string, preset string, crf int, callback ProgressCallback) error {
	// Get duration for progress calculation
	info, err := f.Probe(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("failed to probe input: %w", err)
	}

	if preset == "" {
		preset = "medium"
	}
	if crf == 0 {
		crf = 23
	}

	args := []string{
		"-i", inputPath,
		"-map", "0:v:0",  // Map first video stream only
		"-map", "0:a?",   // Map all audio streams (optional)
		"-c:v", "libx264", // H.264 video codec
		"-preset", preset, // Encoding preset (ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow)
		"-crf", strconv.Itoa(crf), // Constant Rate Factor (0-51, lower = better quality)
		"-c:a", "aac", // AAC audio codec
		"-b:a", "128k", // Audio bitrate
		"-movflags", "+faststart", // Enable fast start for web playback
		"-y",           // Overwrite output
		"-progress", "pipe:1", // Output progress to stdout
		outputPath,
	}

	return f.runWithProgress(ctx, args, info.Duration, callback)
}

// runWithProgress runs ffmpeg command and reports progress
func (f *FFmpeg) runWithProgress(ctx context.Context, args []string, totalDuration float64, callback ProgressCallback) error {
	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Capture stderr in background
	var stderrOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrOutput.WriteString(scanner.Text() + "\n")
		}
	}()

	// Parse progress output
	scanner := bufio.NewScanner(stdout)
	timeRegex := regexp.MustCompile(`out_time_ms=(\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := timeRegex.FindStringSubmatch(line); len(matches) > 1 {
			timeMs, _ := strconv.ParseInt(matches[1], 10, 64)
			currentTime := float64(timeMs) / 1000000.0 // Convert microseconds to seconds

			if totalDuration > 0 && callback != nil {
				progress := (currentTime / totalDuration) * 100
				if progress > 100 {
					progress = 100
				}
				callback(progress)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		errMsg := stderrOutput.String()
		if errMsg != "" {
			return fmt.Errorf("ffmpeg failed: %w, stderr: %s", err, errMsg)
		}
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	// Ensure 100% progress is reported
	if callback != nil {
		callback(100)
	}

	return nil
}

// GenerateOutputPath generates output path with _transcoded suffix
func GenerateOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)

	return filepath.Join(dir, base+"_transcoded.mp4")
}

// IsVideoFile checks if a file is a video file that might need transcoding
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	videoExts := map[string]bool{
		".mkv":  true,
		".avi":  true,
		".wmv":  true,
		".flv":  true,
		".mov":  true,
		".ts":   true,
		".m2ts": true,
		".mts":  true,
		".mpeg": true,
		".mpg":  true,
		".webm": true,
		".mp4":  true,
		".m4v":  true,
		".3gp":  true,
		".rmvb": true,
		".rm":   true,
	}
	return videoExts[ext]
}

// NeedsTranscoding checks if a file extension typically needs transcoding
func NeedsTranscoding(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	// Extensions that typically need transcoding or remux
	needsTranscode := map[string]bool{
		".mkv":  true,
		".avi":  true,
		".wmv":  true,
		".flv":  true,
		".ts":   true,
		".m2ts": true,
		".mts":  true,
		".mpeg": true,
		".mpg":  true,
		".rmvb": true,
		".rm":   true,
		".3gp":  true,
	}
	return needsTranscode[ext]
}

// SubtitleStream represents a subtitle stream detected in a video file
type SubtitleStream struct {
	Index     int    // FFmpeg stream index
	CodecName string // Codec name (subrip, ass, mov_text, etc.)
	Language  string // ISO 639-2 language code (eng, chi, jpn)
	Title     string // Subtitle title/label
}

// SubtitleExtractResult represents an extracted subtitle file
type SubtitleExtractResult struct {
	StreamIndex   int    // Original FFmpeg stream index
	Language      string // ISO 639-2 language code
	LanguageName  string // Human-readable language name
	Title         string // Subtitle title
	Format        string // Output format (srt, ass, vtt)
	OriginalCodec string // Original codec name
	FilePath      string // Extracted file path
	FileSize      int64  // File size in bytes
}

// languageCodeMap maps ISO 639-2 language codes to human-readable names
var languageCodeMap = map[string]string{
	"eng": "English", "chi": "Chinese", "zho": "Chinese",
	"jpn": "Japanese", "kor": "Korean", "spa": "Spanish",
	"fre": "French", "fra": "French", "ger": "German",
	"deu": "German", "ita": "Italian", "por": "Portuguese",
	"rus": "Russian", "ara": "Arabic", "hin": "Hindi",
	"tha": "Thai", "vie": "Vietnamese", "ind": "Indonesian",
	"may": "Malay", "msa": "Malay", "tur": "Turkish",
	"pol": "Polish", "nld": "Dutch", "dut": "Dutch",
	"swe": "Swedish", "nor": "Norwegian", "dan": "Danish",
	"fin": "Finnish", "ces": "Czech", "cze": "Czech",
	"hun": "Hungarian", "ron": "Romanian", "rum": "Romanian",
	"bul": "Bulgarian", "hrv": "Croatian", "srp": "Serbian",
	"slk": "Slovak", "slo": "Slovak", "slv": "Slovenian",
	"ukr": "Ukrainian", "heb": "Hebrew", "und": "Unknown",
}

// GetLanguageName converts ISO 639-2 language code to human-readable name
func GetLanguageName(code string) string {
	if name, ok := languageCodeMap[strings.ToLower(code)]; ok {
		return name
	}
	if code == "" {
		return "Unknown"
	}
	return code
}

// getSubtitleExtension returns the file extension for a subtitle codec
func getSubtitleExtension(codec string) string {
	switch strings.ToLower(codec) {
	case "subrip", "srt":
		return "srt"
	case "ass", "ssa":
		return "ass"
	case "webvtt", "vtt":
		return "vtt"
	case "mov_text", "tx3g":
		return "srt"
	default:
		return "srt"
	}
}

// ProbeSubtitles detects subtitle streams in a video file
func (f *FFmpeg) ProbeSubtitles(ctx context.Context, inputPath string) ([]SubtitleStream, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe subtitle detection failed: %w", err)
	}

	var probeOutput ffprobeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	var subtitles []SubtitleStream
	for _, stream := range probeOutput.Streams {
		subtitles = append(subtitles, SubtitleStream{
			Index:     stream.Index,
			CodecName: stream.CodecName,
			Language:  stream.Tags.Language,
			Title:     stream.Tags.Title,
		})
	}

	return subtitles, nil
}

// ExtractSubtitles extracts all subtitle streams from a video file to separate files
func (f *FFmpeg) ExtractSubtitles(ctx context.Context, inputPath, outputDir, baseName string) ([]SubtitleExtractResult, error) {
	subtitleStreams, err := f.ProbeSubtitles(ctx, inputPath)
	if err != nil {
		return nil, err
	}

	if len(subtitleStreams) == 0 {
		return nil, nil
	}

	var results []SubtitleExtractResult
	langCount := make(map[string]int)

	for _, stream := range subtitleStreams {
		ext := getSubtitleExtension(stream.CodecName)
		lang := stream.Language
		if lang == "" {
			lang = "und"
		}

		// Handle duplicate languages by appending index
		langCount[lang]++
		suffix := lang
		if langCount[lang] > 1 {
			suffix = fmt.Sprintf("%s_%d", lang, langCount[lang])
		}

		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.%s", baseName, suffix, ext))

		args := []string{
			"-i", inputPath,
			"-map", fmt.Sprintf("0:%d", stream.Index),
			"-c:s", getOutputCodec(ext),
			"-y",
			outputPath,
		}

		cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)
		if err := cmd.Run(); err != nil {
			// Skip failed extractions (some codecs may not convert)
			continue
		}

		var fileSize int64
		if info, err := os.Stat(outputPath); err == nil {
			fileSize = info.Size()
		}

		if fileSize == 0 {
			os.Remove(outputPath)
			continue
		}

		results = append(results, SubtitleExtractResult{
			StreamIndex:   stream.Index,
			Language:      lang,
			LanguageName:  GetLanguageName(lang),
			Title:         stream.Title,
			Format:        ext,
			OriginalCodec: stream.CodecName,
			FilePath:      outputPath,
			FileSize:      fileSize,
		})
	}

	return results, nil
}

// getOutputCodec returns the ffmpeg codec for subtitle output format
func getOutputCodec(ext string) string {
	switch ext {
	case "srt":
		return "srt"
	case "ass":
		return "ass"
	case "vtt":
		return "webvtt"
	default:
		return "srt"
	}
}
