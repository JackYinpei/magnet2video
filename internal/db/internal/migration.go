// Package internal provides database migration functionality
// Author: Done-0
// Created: 2025-08-24
package internal

import (
	"fmt"
	"log"

	"magnet2video/internal/model"
	torrentModel "magnet2video/internal/model/torrent"
)

// migrate performs database auto migration
func (m *Manager) migrate() error {
	err := m.db.AutoMigrate(
		model.GetAllModels()...,
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate database: %w", err)
	}

	if err := m.ensureTorrentFileUniqueIndex(); err != nil {
		// Don't fail startup over a non-creatable index — the application-layer
		// transactional Count + Create still gives best-effort uniqueness, and
		// blocking the whole service over a single legacy-data issue would be
		// worse than running with a soft guarantee. The error is logged loudly
		// so the operator can inspect.
		log.Printf("WARN: torrent_files unique index not created: %v", err)
	}

	log.Println("Database auto migration succeeded")
	return nil
}

// ensureTorrentFileUniqueIndex installs a composite unique index on
// torrent_files(torrent_id, index). This is the DB-level guard that
// app-layer Count + Create (in handleTranscodeCompleted etc.) cannot
// give us under MySQL REPEATABLE READ — two concurrent transactions
// can read the same Count snapshot and both INSERT the same index.
//
// Idempotent: if the index already exists, this is a no-op. If existing
// data contains duplicates, we log them and skip creation rather than
// failing — operator must dedupe manually.
func (m *Manager) ensureTorrentFileUniqueIndex() error {
	const indexName = "uniq_torrent_file_torrent_id_index"

	if m.db.Migrator().HasIndex(&torrentModel.TorrentFile{}, indexName) {
		return nil
	}

	// Detect duplicates first so we don't error out cryptically inside the
	// driver. Quoting `index` is needed because it's a reserved word in MySQL.
	type dupRow struct {
		TorrentID int64 `gorm:"column:torrent_id"`
		Idx       int   `gorm:"column:index"`
		Cnt       int64 `gorm:"column:cnt"`
	}
	var dups []dupRow
	if err := m.db.Raw(
		"SELECT torrent_id, `index`, COUNT(*) AS cnt FROM torrent_files " +
			"GROUP BY torrent_id, `index` HAVING COUNT(*) > 1",
	).Scan(&dups).Error; err != nil {
		return fmt.Errorf("scan torrent_files duplicates: %w", err)
	}
	if len(dups) > 0 {
		for _, d := range dups {
			log.Printf("ERROR: duplicate torrent_files row torrent_id=%d index=%d count=%d — fix manually before unique index can be created",
				d.TorrentID, d.Idx, d.Cnt)
		}
		return fmt.Errorf("found %d (torrent_id, index) duplicate group(s); skipping unique index creation", len(dups))
	}

	if err := m.db.Exec(
		"CREATE UNIQUE INDEX " + indexName + " ON torrent_files (torrent_id, `index`)",
	).Error; err != nil {
		return fmt.Errorf("create unique index %s: %w", indexName, err)
	}
	log.Printf("created unique index %s on torrent_files(torrent_id, index)", indexName)
	return nil
}
