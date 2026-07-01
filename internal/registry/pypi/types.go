package pypi

type SimpleMeta struct {
	APIVersion string `json:"api-version"`
}

type SimpleProject struct {
	Name string `json:"name"`
}

type SimpleIndexResponse struct {
	Meta     SimpleMeta      `json:"meta"`
	Projects []SimpleProject `json:"projects"`
}

type SimpleFileHashes struct {
	SHA256 string `json:"sha256"`
}

type SimpleFile struct {
	Filename   string           `json:"filename"`
	URL        string           `json:"url"`
	Hashes     SimpleFileHashes `json:"hashes"`
	Size       int64            `json:"size,omitempty"`
	UploadTime string           `json:"upload-time,omitempty"`
}

type SimplePackageResponse struct {
	Meta     SimpleMeta   `json:"meta"`
	Name     string       `json:"name"`
	Versions []string     `json:"versions"`
	Files    []SimpleFile `json:"files"`
}
