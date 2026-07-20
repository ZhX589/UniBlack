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

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != 1 || manifest.PublicID == "" || manifest.DisplayName == "" {
		return fmt.Errorf("unsupported or missing manifest")
	}
	return nil
}

func readArchive(r io.Reader) (Manifest, map[string][]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, 64<<20))
	if err != nil {
		return Manifest{}, nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return Manifest{}, nil, err
	}
	files := make(map[string][]byte, len(zr.File))
	var manifest Manifest
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return Manifest{}, nil, err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return Manifest{}, nil, err
		}
		files[f.Name] = content
		if f.Name == "manifest.json" {
			if err := json.Unmarshal(content, &manifest); err != nil {
				return Manifest{}, nil, err
			}
		}
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, nil, err
	}
	for _, event := range manifest.Events {
		for _, evidence := range event.Evidence {
			if evidence.FileName == "" {
				continue
			}
			content, ok := files[evidence.FileName]
			if !ok {
				return Manifest{}, nil, fmt.Errorf("missing evidence file: %s", evidence.FileName)
			}
			if evidence.SHA256 != "" {
				sum := sha256.Sum256(content)
				if evidence.SHA256 != hex.EncodeToString(sum[:]) {
					return Manifest{}, nil, fmt.Errorf("evidence hash mismatch: %s", evidence.FileName)
				}
			}
		}
	}
	return manifest, files, nil
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
				rc, err := s.store.Open(ctx, *v.StorageKey)
				if err != nil {
					return nil, fmt.Errorf("open evidence %s: %w", *v.StorageKey, err)
				}
				b, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					return nil, fmt.Errorf("read evidence %s: %w", *v.StorageKey, err)
				}
				sum := sha256.Sum256(b)
				if item.SHA256 != "" && item.SHA256 != hex.EncodeToString(sum[:]) {
					return nil, fmt.Errorf("evidence hash mismatch: %s", *v.StorageKey)
				}
				w, err := zw.Create(*v.StorageKey)
				if err != nil {
					return nil, err
				}
				if _, err := w.Write(b); err != nil {
					return nil, err
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
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	readme := "UniBlack 对象归档包\n\nmanifest.json 使用 schema_version 1。evidence/ 中的文件由 manifest 的 SHA-256 校验。文本证据为 UTF-8 txt。导入前必须预览，不能覆盖已存在的 public_id。\n"
	w, err = zw.Create("README.txt")
	if err != nil {
		return nil, err
	}
	if _, err := w.Write([]byte(readme)); err != nil {
		return nil, err
	}
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
	manifest, _, err := readArchive(r)
	if err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{PublicID: manifest.PublicID, Valid: true}
	if _, err := s.subjects.GetSubjectByID(context.Background(), manifest.PublicID); err == nil {
		preview.Conflicts = []string{manifest.PublicID}
		preview.Valid = false
	}
	accountConflicts, err := s.subjects.AccountConflicts(context.Background(), manifest.Accounts)
	if err != nil {
		return ImportPreview{}, err
	}
	if len(accountConflicts) > 0 {
		preview.Conflicts = append(preview.Conflicts, accountConflicts...)
		preview.Valid = false
	}
	return preview, nil
}

// Import writes a validated archive only if its public ID is still absent.
// Files are stored before metadata and removed again if the transaction fails.
func (s *ArchiveService) Import(ctx context.Context, r io.Reader, actorID string) (*models.Subject, error) {
	manifest, files, err := readArchive(r)
	if err != nil {
		return nil, err
	}
	if _, err := s.subjects.GetSubjectByID(ctx, manifest.PublicID); err == nil {
		return nil, fmt.Errorf("subject public ID already exists")
	}
	accountConflicts, err := s.subjects.AccountConflicts(ctx, manifest.Accounts)
	if err != nil {
		return nil, err
	}
	if len(accountConflicts) > 0 {
		return nil, fmt.Errorf("account identity already exists")
	}
	stored := make([]string, 0)
	for _, event := range manifest.Events {
		for _, evidence := range event.Evidence {
			if evidence.FileName == "" {
				continue
			}
			if _, err := s.store.Upload(ctx, evidence.FileName, bytes.NewReader(files[evidence.FileName]), "application/octet-stream"); err != nil {
				for _, key := range stored {
					_ = s.store.Delete(ctx, key)
				}
				return nil, err
			}
			stored = append(stored, evidence.FileName)
		}
	}
	subject := &models.Subject{PublicID: manifest.PublicID, DisplayName: manifest.DisplayName, Status: "active", CreatedBy: &actorID}
	for i := range manifest.Accounts {
		if manifest.Accounts[i].CustomAttributes == nil {
			manifest.Accounts[i].CustomAttributes = map[string]interface{}{}
		}
		// Imported IDs belong to the source database; let PostgreSQL generate new IDs.
		manifest.Accounts[i].ID = ""
		manifest.Accounts[i].SubjectID = ""
	}
	events := make([]models.Event, 0, len(manifest.Events))
	evidenceRows := make([]repository.EventEvidence, 0)
	for eventIndex, m := range manifest.Events {
		events = append(events, models.Event{Title: m.Title, Details: m.Details, Status: m.Status, Severity: 1, SubmittedBy: &actorID})
		for _, item := range m.Evidence {
			if item.FileName == "" {
				continue
			}
			key := item.FileName
			original := item.OriginalFilename
			hash := item.SHA256
			size := int64(len(files[key]))
			evidenceRows = append(evidenceRows, repository.EventEvidence{EventIndex: eventIndex, Evidence: models.Evidence{Type: item.Type, StorageKey: &key, OriginalFilename: &original, SHA256: &hash, FileSize: &size, UploadedBy: &actorID}})
		}
	}
	audit := &models.AuditLog{UserID: &actorID, Action: "import", ResourceType: "subject", Changes: map[string]interface{}{"public_id": manifest.PublicID}}
	if err := s.events.PublishWithEvidence(ctx, subject, manifest.Accounts, events, evidenceRows, audit); err != nil {
		for _, key := range stored {
			_ = s.store.Delete(ctx, key)
		}
		return nil, err
	}
	return subject, nil
}
