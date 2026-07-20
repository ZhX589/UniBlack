package exporter

import "testing"

func TestImportManifestRejectsExistingPublicID(t *testing.T) {
	manifest := Manifest{SchemaVersion: 1, PublicID: "UBS_01ABCDEFGHJKMNPQRSTVWXYZ", DisplayName: "archive test"}
	if err := validateManifest(manifest); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	if err := validateManifest(Manifest{SchemaVersion: 1}); err == nil {
		t.Fatal("manifest without public ID accepted")
	}
}
