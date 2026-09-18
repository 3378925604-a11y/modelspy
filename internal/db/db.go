package db

import (
	_ "embed"
	"bytes"
	"encoding/json"
	"strings"
)

//go:embed db.json
var raw []byte

type Candidate struct {
	ID         string    `json:"id"`
	Family     string    `json:"family"`
	Aliases    []string  `json:"aliases"`
	RatioEN    [2]float64 `json:"ratio_en"`
	RatioZH    [2]float64 `json:"ratio_zh"`
	TPOTMsMin  float64   `json:"tpot_ms_min"`
	TPOTMsMax  float64   `json:"tpot_ms_max"`
	Cutoff     string    `json:"cutoff"`
	Capability string    `json:"capability"`
}

type DB struct {
	Version    int         `json:"version"`
	Candidates []Candidate `json:"candidates"`
}

func Load() *DB {
	var d DB
	data := bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(data, &d); err != nil {
		panic("fingerprint db corrupt: " + err.Error())
	}
	return &d
}

func (d *DB) Match(name string) *Candidate {
	n := normalize(name)
	for i := range d.Candidates {
		c := &d.Candidates[i]
		for _, a := range append([]string{c.ID}, c.Aliases...) {
			an := normalize(a)
			if an == "" {
				continue
			}
			if strings.Contains(n, an) || strings.Contains(an, n) {
				return c
			}
		}
	}
	return nil
}

func (d *DB) ByID(id string) *Candidate {
	for i := range d.Candidates {
		if d.Candidates[i].ID == id {
			return &d.Candidates[i]
		}
	}
	return nil
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
