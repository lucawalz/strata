package check

import "github.com/lucawalz/strata/internal/config"

type Level int

const (
	Info Level = iota
	Warning
	Error
)

func (l Level) String() string {
	switch l {
	case Info:
		return "info"
	case Warning:
		return "warning"
	default:
		return "error"
	}
}

const (
	CheckFrontmatter = "frontmatter"
	CheckID          = "id"
	CheckSize        = "size"
	CheckStructure   = "structure"
	CheckLinks       = "links"
	CheckArchive     = "archive"
	CheckConfig      = "config"
	CheckInject      = "inject"
	CheckPII         = "pii"
)

type Finding struct {
	Check   string
	Level   Level
	Path    string
	Line    int
	Message string
}

type Repo struct {
	Root   string
	Config config.Config
}
