package packages

import (
	"testing"
)

func TestParseMavenPomData(t *testing.T) {
	tests := []struct {
		name        string
		xml         string
		wantOk      bool
		groupID     string
		artifactID  string
		version     string
		description string
	}{
		{
			name: "full direct fields",
			xml: `<?xml version="1.0"?>
<project>
  <groupId>com.example</groupId>
  <artifactId>mylib</artifactId>
  <version>1.2.3</version>
  <name>My Library</name>
</project>`,
			wantOk:      true,
			groupID:     "com.example",
			artifactID:  "mylib",
			version:     "1.2.3",
			description: "My Library",
		},
		{
			name: "groupId and version inherited from parent",
			xml: `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>org.springframework</groupId>
    <version>6.1.0</version>
  </parent>
  <artifactId>spring-core</artifactId>
</project>`,
			wantOk:      true,
			groupID:     "org.springframework",
			artifactID:  "spring-core",
			version:     "6.1.0",
			description: "spring-core",
		},
		{
			name: "description falls back to artifactId when name is empty",
			xml: `<?xml version="1.0"?>
<project>
  <groupId>com.example</groupId>
  <artifactId>mylib</artifactId>
  <version>1.0.0</version>
</project>`,
			wantOk:      true,
			groupID:     "com.example",
			artifactID:  "mylib",
			version:     "1.0.0",
			description: "mylib",
		},
		{
			name: "missing artifactId",
			xml: `<?xml version="1.0"?>
<project>
  <groupId>com.example</groupId>
  <version>1.0.0</version>
</project>`,
			wantOk: false,
		},
		{
			name: "missing groupId and no parent",
			xml: `<?xml version="1.0"?>
<project>
  <artifactId>mylib</artifactId>
  <version>1.0.0</version>
</project>`,
			wantOk: false,
		},
		{
			name:   "invalid XML",
			xml:    `not xml at all`,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupID, artifactID, version, description, ok := parseMavenPomData([]byte(tt.xml))
			if ok != tt.wantOk {
				t.Fatalf("ok: got %v, want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if groupID != tt.groupID {
				t.Errorf("groupID: got %q, want %q", groupID, tt.groupID)
			}
			if artifactID != tt.artifactID {
				t.Errorf("artifactID: got %q, want %q", artifactID, tt.artifactID)
			}
			if version != tt.version {
				t.Errorf("version: got %q, want %q", version, tt.version)
			}
			if description != tt.description {
				t.Errorf("description: got %q, want %q", description, tt.description)
			}
		})
	}
}
