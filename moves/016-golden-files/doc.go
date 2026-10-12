// Package golden pins the exact rendered prompt in
// testdata/*.golden files. The whole move is a test:
// see golden_test.go.
//
// Move 16: golden-file tests for prompts.
//
//	go test ./moves/016-golden-files            # check
//	go test ./moves/016-golden-files -update    # rewrite, then read the git diff
package golden
