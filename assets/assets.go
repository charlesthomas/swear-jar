package assets

import (
	_ "embed"
	"fmt"
	"strings"
)

const RepoURL = "https://github.com/charlesthomas/swear-jar"

//go:embed logo.svg
var logo string

const logoSize = `width="512" height="512"`

func Logo(x, y, size int) string {
	return strings.Replace(logo, logoSize, fmt.Sprintf(`x="%d" y="%d" width="%d" height="%d"`, x, y, size, size), 1)
}
