package coreassets

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed all:*
var files embed.FS

func Load(target string) ([]byte, error) {
	data, err := files.ReadFile(target + "/bundle-core")
	if err != nil && strings.HasPrefix(target, "windows-") {
		data, err = files.ReadFile(target + "/bundle-core.exe")
	}
	if err != nil {
		return nil, fmt.Errorf("load %s core: %w", target, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("load %s core: generated core is empty", target)
	}
	return data, nil
}
