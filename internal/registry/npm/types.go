package npm

import "encoding/json"

// PackageMetadata represents the metadata document for an NPM package.
type PackageMetadata struct {
	Name     string                     `json:"name"`
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]VersionMetadata `json:"versions"`
}

type VersionMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    Dist   `json:"dist"`
}

type Dist struct {
	Tarball   string `json:"tarball"`
	Shasum    string `json:"shasum,omitempty"`
	Integrity string `json:"integrity,omitempty"`
}

// PublishPayload represents the payload sent by npm publish.
type PublishPayload struct {
	ID          string                     `json:"_id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	DistTags    map[string]string          `json:"dist-tags"`
	Versions    map[string]json.RawMessage `json:"versions"`
	Attachments map[string]Attachment      `json:"_attachments"`
}

type Attachment struct {
	ContentType string `json:"content_type"`
	Data        string `json:"data"` // base64 encoded
	Length      int    `json:"length"`
}
