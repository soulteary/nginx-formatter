package updater

import (
	"fmt"
	"strings"
)

// UnifiedDiff renders a unified diff between two versions of one file.
//
// It is line based with three lines of context, which is all a formatter needs:
// the only differences it ever reports are whitespace and line-level layout.
// Written here rather than pulled in as a dependency — the whole algorithm is
// a longest-common-subsequence table and a walk over it.
func UnifiedDiff(name, before, after string) string {
	if before == after {
		return ""
	}
	a, b := splitLines(before), splitLines(after)
	hunks := groupHunks(lcsOps(a, b), 3)
	if len(hunks) == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", name, name)
	for _, h := range hunks {
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n", h.aStart+1, h.aLines, h.bStart+1, h.bLines)
		for _, op := range h.ops {
			sb.WriteString(op.kind)
			sb.WriteString(op.text)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// splitLines splits into lines, dropping the empty element a trailing newline
// produces so "a\n" is one line rather than two.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// op is one line of diff output: " " context, "-" removed, "+" added.
type op struct {
	kind string
	text string
}

// lcsOps turns two line slices into a flat op list via a
// longest-common-subsequence table.
func lcsOps(a, b []string) []op {
	// table[i][j] is the LCS length of a[i:] and b[j:].
	table := make([][]int, len(a)+1)
	for i := range table {
		table[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				table[i][j] = table[i+1][j+1] + 1
			} else {
				table[i][j] = max(table[i+1][j], table[i][j+1])
			}
		}
	}

	var ops []op
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			ops = append(ops, op{" ", a[i]})
			i, j = i+1, j+1
		case table[i+1][j] >= table[i][j+1]:
			ops = append(ops, op{"-", a[i]})
			i++
		default:
			ops = append(ops, op{"+", b[j]})
			j++
		}
	}
	for ; i < len(a); i++ {
		ops = append(ops, op{"-", a[i]})
	}
	for ; j < len(b); j++ {
		ops = append(ops, op{"+", b[j]})
	}
	return ops
}

// hunk is a run of changes plus its surrounding context.
type hunk struct {
	aStart, aLines int
	bStart, bLines int
	ops            []op
}

// groupHunks slices the op list into hunks carrying context lines either side
// of each run of changes, collapsing the unchanged stretches between them.
func groupHunks(ops []op, context int) []hunk {
	changed := make([]bool, len(ops))
	any := false
	for i, o := range ops {
		if o.kind != " " {
			changed[i] = true
			any = true
		}
	}
	if !any {
		return nil
	}

	// keep[i] marks an op that falls within `context` of a change.
	keep := make([]bool, len(ops))
	for i, c := range changed {
		if !c {
			continue
		}
		for j := max(0, i-context); j <= min(len(ops)-1, i+context); j++ {
			keep[j] = true
		}
	}

	var hunks []hunk
	aLine, bLine := 0, 0
	for i := 0; i < len(ops); {
		if !keep[i] {
			switch ops[i].kind {
			case " ":
				aLine, bLine = aLine+1, bLine+1
			case "-":
				aLine++
			case "+":
				bLine++
			}
			i++
			continue
		}

		h := hunk{aStart: aLine, bStart: bLine}
		for ; i < len(ops) && keep[i]; i++ {
			h.ops = append(h.ops, ops[i])
			switch ops[i].kind {
			case " ":
				h.aLines, h.bLines = h.aLines+1, h.bLines+1
				aLine, bLine = aLine+1, bLine+1
			case "-":
				h.aLines++
				aLine++
			case "+":
				h.bLines++
				bLine++
			}
		}
		hunks = append(hunks, h)
	}
	return hunks
}
