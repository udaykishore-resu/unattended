// Package emailrecord strips quoted history from an
// email before any model sees it.
//
// Move 25: email-to-record pipelines.
package emailrecord

import (
	"bufio"
	"strings"
)

func newText(body string) string {
	var b strings.Builder
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		ln := sc.Text()
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, ">") ||
			strings.HasPrefix(t, "On ") &&
				strings.HasSuffix(t, "wrote:") ||
			strings.HasPrefix(t, "-----Original") {
			break
		}
		b.WriteString(ln + "\n")
	}
	return b.String()
}
