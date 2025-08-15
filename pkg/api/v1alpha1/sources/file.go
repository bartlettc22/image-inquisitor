package sources

const (
	// FileSourceType is the type of file source
	FileSourceType = "file"
)

// FileSource is a source of images from a file
type FileSource struct {

	// File is the path to the file
	File string `yaml:"file" json:"file"`

	// Line is the line number to start scanning from
	Line int `yaml:"line" json:"line"`
}
