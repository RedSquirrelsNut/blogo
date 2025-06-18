package schema

import (
	"embed"
	"fmt"
)

//go:embed users.sql
//go:embed feeds.sql
//go:embed feed_follows.sql
//go:embed posts.sql
var ddlFiles embed.FS

func LoadSchema(name string) (string, error) {
	b, err := ddlFiles.ReadFile(name + ".sql")
	if err != nil {
		return "", fmt.Errorf("load Schema: %w", err)
	}
	return string(b), nil
}
