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
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

const (
	maxArchiveBytes      = 64 << 20
	maxArchiveFiles      = 256
	maxArchiveEntryBytes = 16 << 20
	maxArchiveExpanded   = 128 << 20
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
	// Serializes confirmation imports so conflict checks and key writes cannot race.
	mu sync.Mutex
}

func NewArchiveService(s *repository.SubjectRepository, e *repository.EventRepository, v *repository.EvidenceRepository, store storage.Storage) *ArchiveService {
	return &ArchiveService{subjects: s, events: e, evidence: v, store: store}
}

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != 1 || manifest.PublicID == "" || manifest.DisplayName == "" {
		return fmt.Errorf("unsupported or missing manifest")
	}
	return nil
}

func evidencePrefix(publicID string) string {
	return "subjects/" + publicID + "/evidence/"
}

func validateEvidenceName(publicID, name string) error {
	if name == "" {
		return fmt.Errorf("missing evidence file name")
	}
	cleaned := path.Clean("/" + name)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned != name || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid evidence file name: %s", name)
	}
	prefix := evidencePrefix(publicID)
	if !strings.HasPrefix(name, prefix) {
		return fmt.Errorf("evidence file outside subject namespace: %s", name)
	}
	return nil
}

func readArchive(r io.Reader) (Manifest, map[string][]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxArchiveBytes+1))
	if err != nil {
		return Manifest{}, nil, err
	}
	if len(b) > maxArchiveBytes {
		return Manifest{}, nil, fmt.Errorf("archive exceeds compressed size limit")
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return Manifest{}, nil, err
	}
	if len(zr.File) == 0 || len(zr.File) > maxArchiveFiles {
		return Manifest{}, nil, fmt.Errorf("archive file count out of range")
	}

	files := make(map[string][]byte, len(zr.File))
	var manifest Manifest
	var expanded uint64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.UncompressedSize64 > maxArchiveEntryBytes {
			return Manifest{}, nil, fmt.Errorf("archive entry too large: %s", f.Name)
		}
		if expanded+f.UncompressedSize64 > maxArchiveExpanded {
			return Manifest{}, nil, fmt.Errorf("archive expanded size limit exceeded")
		}
		if _, exists := files[f.Name]; exists {
			return Manifest{}, nil, fmt.Errorf("duplicate archive entry: %s", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return Manifest{}, nil, err
		}
		content, err := io.ReadAll(io.LimitReader(rc, int64(maxArchiveEntryBytes)+1))
		rc.Close()
		if err != nil {
			return Manifest{}, nil, err
		}
		if len(content) > maxArchiveEntryBytes {
			return Manifest{}, nil, fmt.Errorf("archive entry too large: %s", f.Name)
		}
		expanded += uint64(len(content))
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

	seenEvidence := map[string]bool{}
	for _, event := range manifest.Events {
		switch event.Status {
		case "", "published", "corrected", "withdrawn", "draft", "hidden_pending_review":
		default:
			return Manifest{}, nil, fmt.Errorf("invalid event status: %s", event.Status)
		}
		for _, evidence := range event.Evidence {
			if evidence.FileName == "" {
				continue
			}
			if err := validateEvidenceName(manifest.PublicID, evidence.FileName); err != nil {
				return Manifest{}, nil, err
			}
			if seenEvidence[evidence.FileName] {
				return Manifest{}, nil, fmt.Errorf("duplicate evidence file name: %s", evidence.FileName)
			}
			seenEvidence[evidence.FileName] = true
			if evidence.Type != "file" && evidence.Type != "text" && evidence.Type != "link" {
				return Manifest{}, nil, fmt.Errorf("invalid evidence type: %s", evidence.Type)
			}
			if evidence.Type != "link" {
				if len(evidence.SHA256) != 64 {
					return Manifest{}, nil, fmt.Errorf("missing evidence hash: %s", evidence.FileName)
				}
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
				if item.SHA256 == "" {
					item.SHA256 = hex.EncodeToString(sum[:])
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
// Files are stored under the subject namespace before metadata and removed if the
// transaction fails. A process-local mutex serializes confirmation imports.
func (s *ArchiveService) Import(ctx context.Context, r io.Reader, actorID string) (*models.Subject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	cleanup := func() {
		for _, key := range stored {
			_ = s.store.Delete(ctx, key)
		}
	}

	for _, event := range manifest.Events {
		for _, evidence := range event.Evidence {
			if evidence.FileName == "" || evidence.Type == "link" {
				continue
			}
			if err := validateEvidenceName(manifest.PublicID, evidence.FileName); err != nil {
				cleanup()
				return nil, err
			}
			// Refuse overwrite of existing object keys belonging to this namespace.
			if rc, err := s.store.Open(ctx, evidence.FileName); err == nil {
				rc.Close()
				cleanup()
				return nil, fmt.Errorf("evidence object already exists: %s", evidence.FileName)
			}
			if _, err := s.store.Upload(ctx, evidence.FileName, bytes.NewReader(files[evidence.FileName]), "application/octet-stream"); err != nil {
				cleanup()
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
		manifest.Accounts[i].ID = ""
		manifest.Accounts[i].SubjectID = ""
	}
	events := make([]models.Event, 0, len(manifest.Events))
	evidenceRows := make([]repository.EventEvidence, 0)
	for eventIndex, m := range manifest.Events {
		status := m.Status
		if status == "" {
			status = "published"
		}
		events = append(events, models.Event{Title: m.Title, Details: m.Details, Status: status, Severity: 1, SubmittedBy: &actorID})
		for _, item := range m.Evidence {
			if item.FileName == "" || item.Type == "link" {
				continue
			}
			key := item.FileName
			original := item.OriginalFilename
			hash := item.SHA256
			size := int64(len(files[key]))
			mime := "application/octet-stream"
			if item.Type == "text" {
				mime = "text/plain"
			}
			evidenceRows = append(evidenceRows, repository.EventEvidence{
				EventIndex: eventIndex,
				Evidence: models.Evidence{
					Type:             item.Type,
					StorageKey:       &key,
					OriginalFilename: &original,
					SHA256:           &hash,
					FileSize:         &size,
					MimeType:         &mime,
					UploadedBy:       &actorID,
				},
			})
		}
	}
	audit := &models.AuditLog{UserID: &actorID, Action: "import", ResourceType: "subject", Changes: map[string]interface{}{"public_id": manifest.PublicID}}
	if err := s.events.PublishWithEvidence(ctx, subject, manifest.Accounts, events, evidenceRows, audit); err != nil {
		cleanup()
		return nil, err
	}
	return subject, nil
}
