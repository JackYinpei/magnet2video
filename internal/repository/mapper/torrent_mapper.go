// Package mapper provides bidirectional mapping between domain models and GORM models.
// Author: Done-0
// Created: 2026-03-16
package mapper

import (
	domain "magnet2video/domain/torrent"
	torrentModel "magnet2video/internal/model/torrent"
)

// TorrentToDomain converts a GORM Torrent model to a domain Torrent.
func TorrentToDomain(m *torrentModel.Torrent) *domain.Torrent {
	if m == nil {
		return nil
	}
	t := &domain.Torrent{
		ID:                  m.ID,
		InfoHash:            m.InfoHash,
		Name:                m.Name,
		TotalSize:           m.TotalSize,
		PosterPath:          m.PosterPath,
		DownloadPath:        m.DownloadPath,
		Trackers:            []string(m.Trackers),
		Status:              domain.DownloadStatus(m.Status),
		Progress:            m.Progress,
		Visibility:          domain.Visibility(m.Visibility),
		CreatorID:           m.CreatorID,
		TranscodeStatus:     domain.TranscodeStatus(m.TranscodeStatus),
		TranscodeProgress:   m.TranscodeProgress,
		TranscodedCount:     m.TranscodedCount,
		TotalTranscode:      m.TotalTranscode,
		CloudUploadStatus:   domain.CloudUploadStatus(m.CloudUploadStatus),
		CloudUploadProgress: m.CloudUploadProgress,
		CloudUploadedCount:  m.CloudUploadedCount,
		TotalCloudUpload:    m.TotalCloudUpload,
		LocalDeleted:        m.LocalDeleted,
		WorkerID:            m.WorkerID,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}

	t.Files = make([]domain.TorrentFile, len(m.Files))
	for i := range m.Files {
		t.Files[i] = *FileToDomain(&m.Files[i])
	}
	return t
}

// TorrentToModel converts a domain Torrent to a GORM Torrent model.
func TorrentToModel(d *domain.Torrent) *torrentModel.Torrent {
	if d == nil {
		return nil
	}
	m := &torrentModel.Torrent{
		InfoHash:            d.InfoHash,
		Name:                d.Name,
		TotalSize:           d.TotalSize,
		PosterPath:          d.PosterPath,
		DownloadPath:        d.DownloadPath,
		Trackers:            torrentModel.StringSlice(d.Trackers),
		Status:              int(d.Status),
		Progress:            d.Progress,
		Visibility:          int(d.Visibility),
		CreatorID:           d.CreatorID,
		TranscodeStatus:     int(d.TranscodeStatus),
		TranscodeProgress:   d.TranscodeProgress,
		TranscodedCount:     d.TranscodedCount,
		TotalTranscode:      d.TotalTranscode,
		CloudUploadStatus:   int(d.CloudUploadStatus),
		CloudUploadProgress: d.CloudUploadProgress,
		CloudUploadedCount:  d.CloudUploadedCount,
		TotalCloudUpload:    d.TotalCloudUpload,
		LocalDeleted:        d.LocalDeleted,
		WorkerID:            d.WorkerID,
	}
	m.ID = d.ID
	m.CreatedAt = d.CreatedAt
	m.UpdatedAt = d.UpdatedAt

	m.Files = make([]torrentModel.TorrentFile, len(d.Files))
	for i := range d.Files {
		m.Files[i] = *FileToModel(&d.Files[i])
	}
	return m
}

// FileToDomain converts a GORM TorrentFile to a domain TorrentFile.
func FileToDomain(m *torrentModel.TorrentFile) *domain.TorrentFile {
	if m == nil {
		return nil
	}
	return &domain.TorrentFile{
		ID:                m.ID,
		TorrentID:         m.TorrentID,
		Index:             m.Index,
		Path:              m.Path,
		Size:              m.Size,
		IsSelected:        m.IsSelected,
		IsShareable:       m.IsShareable,
		IsStreamable:      m.IsStreamable,
		Type:              domain.FileType(m.Type),
		Source:            domain.FileSource(m.Source),
		ParentPath:        m.ParentPath,
		TranscodeStatus:   domain.TranscodeStatus(m.TranscodeStatus),
		TranscodedPath:    m.TranscodedPath,
		TranscodeError:    m.TranscodeError,
		CloudUploadStatus: domain.CloudUploadStatus(m.CloudUploadStatus),
		CloudPath:         m.CloudPath,
		CloudUploadError:  m.CloudUploadError,
		StreamIndex:       m.StreamIndex,
		Language:          m.Language,
		LanguageName:      m.LanguageName,
		Title:             m.Title,
		Format:            m.Format,
		OriginalCodec:     m.OriginalCodec,
	}
}

// FileToModel converts a domain TorrentFile to a GORM TorrentFile.
func FileToModel(d *domain.TorrentFile) *torrentModel.TorrentFile {
	if d == nil {
		return nil
	}
	m := &torrentModel.TorrentFile{
		TorrentID:         d.TorrentID,
		Index:             d.Index,
		Path:              d.Path,
		Size:              d.Size,
		IsSelected:        d.IsSelected,
		IsShareable:       d.IsShareable,
		IsStreamable:      d.IsStreamable,
		Type:              string(d.Type),
		Source:            string(d.Source),
		ParentPath:        d.ParentPath,
		TranscodeStatus:   int(d.TranscodeStatus),
		TranscodedPath:    d.TranscodedPath,
		TranscodeError:    d.TranscodeError,
		CloudUploadStatus: int(d.CloudUploadStatus),
		CloudPath:         d.CloudPath,
		CloudUploadError:  d.CloudUploadError,
		StreamIndex:       d.StreamIndex,
		Language:          d.Language,
		LanguageName:      d.LanguageName,
		Title:             d.Title,
		Format:            d.Format,
		OriginalCodec:     d.OriginalCodec,
	}
	m.ID = d.ID
	return m
}
