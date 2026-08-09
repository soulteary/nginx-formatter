package nginx

import "strings"

// Format renders cfg back to text using indent copies of char per nesting
// level. It reproduces the historical formatter's layout: preserved blank
// lines between statements, a blank line after a "}" whose preceding line is a
// non-empty non-"}" line, empty blocks rendered as "{  }", and collapsing of
// 3+ consecutive newlines into 2.
func Format(cfg *Config, indent int, char string) string {
	if cfg == nil || len(cfg.Nodes) == 0 {
		return ""
	}
	unit := strings.Repeat(char, indent)

	var lines []string
	renderNodes(&lines, cfg.Nodes, 0, unit)

	lines = addEmptyLineAfterBraces(lines)
	out := strings.Join(lines, "\n")
	out = foldEmptyBrackets(out)
	return out
}

func indentOf(level int, unit string) string {
	return strings.Repeat(unit, level)
}

func renderNodes(lines *[]string, nodes []Node, level int, unit string) {
	for _, n := range nodes {
		switch node := n.(type) {
		case *BlankLine:
			*lines = append(*lines, "")
		case *Comment:
			*lines = append(*lines, indentOf(level, unit)+"#"+node.Text)
		case *Directive:
			*lines = append(*lines, indentOf(level, unit)+renderStatementHead(node.Name, node.Args)+";"+inlineComment(node.InlineComment))
		case *Block:
			head := indentOf(level, unit) + renderStatementHead(node.Name, node.Args)
			if head != "" {
				head += " "
			}
			if len(node.Body) == 0 {
				// Empty block; rendered as "{  }".
				*lines = append(*lines, head+"{  }"+inlineComment(node.OpenComment))
				continue
			}
			*lines = append(*lines, head+"{"+inlineComment(node.OpenComment))
			renderNodes(lines, node.Body, level+1, unit)
			*lines = append(*lines, indentOf(level, unit)+"}"+inlineComment(node.InlineComment))
		case *RawBlock:
			head := indentOf(level, unit) + renderStatementHead(node.Name, node.Args)
			if head != "" {
				head += " "
			}
			raw := strings.Trim(node.Raw, "\n")
			if strings.TrimSpace(raw) == "" {
				// No meaningful body; render like an empty block.
				*lines = append(*lines, head+"{  }"+inlineComment(node.OpenComment))
				continue
			}
			*lines = append(*lines, head+"{"+inlineComment(node.OpenComment))
			base := indentOf(level+1, unit)
			*lines = append(*lines, reindentRawLines(raw, base)...)
			*lines = append(*lines, indentOf(level, unit)+"}"+inlineComment(node.InlineComment))
		}
	}
}

// reindentRawLines normalizes the verbatim body of a raw block. It strips the
// common leading whitespace shared by all non-blank lines (so the block is
// idempotent regardless of the source indentation) and then prefixes each
// non-blank line with base. Blank lines are emitted empty.
func reindentRawLines(raw, base string) []string {
	src := strings.Split(raw, "\n")
	// Drop leading and trailing whitespace-only lines so the block body sits
	// flush against its braces (and stays idempotent across reformats).
	for len(src) > 0 && strings.TrimSpace(src[0]) == "" {
		src = src[1:]
	}
	for len(src) > 0 && strings.TrimSpace(src[len(src)-1]) == "" {
		src = src[:len(src)-1]
	}
	common := -1
	for _, line := range src {
		trimmed := strings.TrimRight(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if common == -1 || indent < common {
			common = indent
		}
	}
	if common < 0 {
		common = 0
	}
	out := make([]string, 0, len(src))
	for _, line := range src {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		body := line
		if len(body) >= common {
			body = body[common:]
		} else {
			body = strings.TrimLeft(body, " \t")
		}
		out = append(out, base+strings.TrimRight(body, " \t"))
	}
	return out
}

func renderStatementHead(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	if name != "" {
		parts = append(parts, name)
	}
	parts = append(parts, args...)
	return strings.Join(parts, " ")
}

func inlineComment(text string) string {
	if text == "" {
		return ""
	}
	return " #" + text
}

// addEmptyLineAfterBraces inserts a blank line after any "}" line whose
// immediately preceding line is a non-empty, non-"}" line, matching the
// historical add_empty_line_after_nginx_directives behavior.
func addEmptyLineAfterBraces(lines []string) []string {
	out := make([]string, 0, len(lines)+4)
	for i := 0; i < len(lines); i++ {
		cur := strings.TrimSpace(lines[i])
		out = append(out, lines[i])
		if strings.HasPrefix(cur, "}") {
			var prev string
			if i > 0 {
				prev = strings.TrimSpace(lines[i-1])
			}
			if prev != "" && !strings.HasPrefix(prev, "}") {
				out = append(out, "")
			}
		}
	}
	return out
}

// foldEmptyBrackets collapses any run of 3+ newlines down to 2, mirroring the
// historical fold_empty_brackets. (Empty blocks are already rendered as
// "{  }" directly by renderNodes; this function only normalizes blank runs.)
func foldEmptyBrackets(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}
