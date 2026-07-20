package exporter

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

type EvidenceManifest struct {
	FileName         string `json:"file_name"`
	OriginalFilename string `json:"original_filename,omitempty"`
	SHA256           string `json:"sha256,omitempty"`
	Type             string `json:"type"`
}
type EventManifest struct {
	ID       string             `json:"id"`
	Title    string             `json:"title"`
	Details  string             `json:"details"`
	Status   string             `json:"status"`
	Evidence []EvidenceManifest `json:"evidence"`
}
type Manifest struct {
	SchemaVersion int              `json:"schema_version"`
	PublicID      string           `json:"public_id"`
	DisplayName   string           `json:"display_name"`
	Accounts      []models.Account `json:"accounts"`
	Events        []EventManifest  `json:"events"`
	ExportedAt    time.Time        `json:"exported_at"`
}

type ArchiveService struct {
	subjects *repository.SubjectRepository
	events   *repository.EventRepository
	evidence *repository.EvidenceRepository
	store    storage.Storage
}

func NewArchiveService(s *repository.SubjectRepository, e *repository.EventRepository, v *repository.EvidenceRepository, store storage.Storage) *ArchiveService {
	return &ArchiveService{subjects: s, events: e, evidence: v, store: store}
}

func (s *ArchiveService) Build(ctx context.Context, publicID string) ([]byte, error) {
	subject, err := s.subjects.GetSubjectByID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	events, err := s.events.ListBySubject(ctx, subject.ID)
	if err != nil {
		return nil, err
	}
	m := Manifest{SchemaVersion: 1, PublicID: subject.PublicID, DisplayName: subject.DisplayName, Accounts: subject.Accounts, ExportedAt: time.Now().UTC()}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, event := range events {
		em := EventManifest{ID: event.ID, Title: event.Title, Details: event.Details, Status: event.Status}
		items, err := s.evidence.GetEvidenceByEventID(ctx, event.ID)
		if err != nil {
			return nil, err
		}
		for _, v := range items {
			item := EvidenceManifest{Type: v.Type}
			if v.OriginalFilename != nil {
				item.OriginalFilename = *v.OriginalFilename
			}
			if v.SHA256 != nil {
				item.SHA256 = *v.SHA256
			}
			if v.StorageKey != nil {
				item.FileName = *v.StorageKey
				if rc, err := s.store.Open(ctx, *v.StorageKey); err == nil {
					b, _ := io.ReadAll(rc)
					rc.Close()
					sum := sha256.Sum256(b)
					if item.SHA256 != "" && item.SHA256 != hex.EncodeToString(sum[:]) {
						return nil, fmt.Errorf("evidence hash mismatch: %s", *v.StorageKey)
					}
					w, _ := zw.Create(*v.StorageKey)
					_, _ = w.Write(b)
				}
			}
			em.Evidence = append(em.Evidence, item)
		}
		m.Events = append(m.Events, em)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	w, err := zw.Create("manifest.json")
	if err != nil {
		return nil, err
	}
	_, _ = w.Write(data)
	readme := "UniBlack 对象归档包\n\nmanifest.json 使用 schema_version 1。evidence/ 中的文件由 manifest 的 SHA-256 校验。文本证据为 UTF-8 txt。导入前必须预览，不能覆盖已存在的 public_id。\n"
	w, err = zw.Create("README.txt")
	if err != nil {
		return nil, err
	}
	_, _ = w.Write([]byte(readme))
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type ImportPreview struct {
	PublicID  string   `json:"public_id"`
	Conflicts []string `json:"conflicts"`
	Valid     bool     `json:"valid"`
}

func (s *ArchiveService) PreviewImport(r io.Reader) (ImportPreview, error) {
	b, err := io.ReadAll(io.LimitReader(r, 64<<20))
	if err != nil {
		return ImportPreview{}, err
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return ImportPreview{}, err
	}
	var manifest Manifest
	for _, f := range zr.File {
		if f.Name != "manifest.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ImportPreview{}, err
		}
		err = json.NewDecoder(rc).Decode(&manifest)
		rc.Close()
		if err != nil {
			return ImportPreview{}, err
		}
		break
	}
	if manifest.SchemaVersion != 1 || manifest.PublicID == "" {
		return ImportPreview{}, fmt.Errorf("unsupported or missing manifest")
	}
	preview := ImportPreview{PublicID: manifest.PublicID, Valid: true}
	if _, err := s.subjects.GetSubjectByID(context.Background(), manifest.PublicID); err == nil {
		preview.Conflicts = []string{manifest.PublicID}
		preview.Valid = false
	}
	return preview, nil
}
