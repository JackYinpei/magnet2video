package serializer

import "peer2http/db"

type Magnet struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Magnet string `json:"magnet"`
	Usage  uint64 `json:"usage"`
}

func BuildMagnet(item db.Magnet) Magnet {
	return Magnet{
		ID:     item.ID,
		Title:  item.Title,
		Magnet: item.Magnet,
		Usage:  item.Usage(),
	}
}

func BuildMagnetList(items []db.Magnet) (magets []Magnet) {
	for _, item := range items {
		magnet := BuildMagnet(item)
		magets = append(magets, magnet)
	}
	return magets
}
