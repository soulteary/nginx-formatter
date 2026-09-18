package nginx

import (
	"strconv"
	"strings"
)

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
	// Verbatim bodies (*_by_lua_block and friends) are replaced by a one-line
	// placeholder while the whole-document text passes run, then spliced back.
	// Those passes cannot tell nginx structure from embedded script, and left
	// to themselves they collapsed blank runs inside Lua [[ ]] long strings
	// and injected a blank line after every line that happened to start with
	// "}" — both silently rewriting the script's own text.
	raw := &rawBodies{}
	renderNodes(&lines, cfg.Nodes, 0, unit, raw)

	lines = addEmptyLineAfterBraces(lines)
	out := strings.Join(lines, "\n")
	out = foldEmptyBrackets(out)
	out = raw.restore(out)
	// Terminate with exactly one newline. Without this the trailing newline is
	// an accident of whether addEmptyLineAfterBraces happened to fire on the
	// last line, so configs ending in a directive lost theirs entirely.
	return strings.TrimRight(out, "\n") + "\n"
}

// rawBodies holds the verbatim block bodies removed from the line stream, so
// the whole-document passes never see them.
type rawBodies struct {
	bodies []string
}

// placeholder stores body and returns the single line that stands in for it.
// U+0000 cannot occur in the output — Parse rejects input containing it — so
// the marker can never collide with real content.
func (r *rawBodies) placeholder(body []string) string {
	r.bodies = append(r.bodies, strings.Join(body, "\n"))
	return "\x00raw:" + strconv.Itoa(len(r.bodies)-1) + "\x00"
}

// restore substitutes every placeholder line back with its body.
func (r *rawBodies) restore(out string) string {
	for i, body := range r.bodies {
		out = strings.Replace(out, "\x00raw:"+strconv.Itoa(i)+"\x00", body, 1)
	}
	return out
}

func indentOf(level int, unit string) string {
	return strings.Repeat(unit, level)
}

func renderNodes(lines *[]string, nodes []Node, level int, unit string, raw *rawBodies) {
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
			if len(node.Body) == 0 && node.OpenComment == "" && node.InlineComment == "" {
				// Empty block with nothing to carry; rendered as "{  }".
				*lines = append(*lines, head+"{  }")
				continue
			}
			// An empty block that has a comment falls through to the two-line
			// form. The compact "{  }" cannot round-trip a comment: on re-parse
			// anything after it binds to the closing brace, so a second format
			// pass would drop it.
			*lines = append(*lines, head+"{"+inlineComment(node.OpenComment))
			renderNodes(lines, node.Body, level+1, unit, raw)
			*lines = append(*lines, indentOf(level, unit)+"}"+inlineComment(node.InlineComment))
		case *RawBlock:
			head := indentOf(level, unit) + renderStatementHead(node.Name, node.Args)
			if head != "" {
				head += " "
			}
			body := strings.Trim(node.Raw, "\n")
			if strings.TrimSpace(body) == "" && node.InlineComment == "" {
				// No meaningful body and no comment; render like an empty block.
				*lines = append(*lines, head+"{  }")
				continue
			}
			*lines = append(*lines, head+"{")
			base := indentOf(level+1, unit)
			*lines = append(*lines, raw.placeholder(reindentRawLines(body, base)))
			*lines = append(*lines, indentOf(level, unit)+"}"+inlineComment(node.InlineComment))
		}
	}
}

// longStringLines reports, for each line of body, whether that line *starts*
// inside a Lua long bracket -- [[ ... ]], [=[ ... ]=], or a --[[ ]] comment.
//
// Every byte between those brackets is string content, so re-indenting such a
// line or trimming its trailing whitespace changes the value the script
// actually sees. Quoted strings and "--" line comments are skipped so a "[["
// appearing inside one does not open a phantom bracket.
func longStringLines(body string) []bool {
	lines := strings.Split(body, "\n")
	inside := make([]bool, len(lines))
	level := -1 // -1 when not inside a long bracket, else its "=" count

	openLevel := func(r []rune, j int) (int, int, bool) {
		k, cnt := j+1, 0
		for k < len(r) && r[k] == '=' {
			cnt++
			k++
		}
		if k < len(r) && r[k] == '[' {
			return cnt, k, true
		}
		return 0, j, false
	}

	for i, line := range lines {
		inside[i] = level >= 0
		r := []rune(line)
		for j := 0; j < len(r); j++ {
			if level >= 0 {
				if r[j] == ']' {
					k, cnt := j+1, 0
					for k < len(r) && r[k] == '=' {
						cnt++
						k++
					}
					if cnt == level && k < len(r) && r[k] == ']' {
						level, j = -1, k
					}
				}
				continue
			}
			switch r[j] {
			case '\'', '"':
				quote := r[j]
				j++
				for j < len(r) && r[j] != quote {
					if r[j] == '\\' {
						j++
					}
					j++
				}
			case '-':
				if j+1 >= len(r) || r[j+1] != '-' {
					continue
				}
				// A Lua comment: "--[[ ... ]]" spans lines, "-- ..." does not.
				if lv, end, ok := openLevel(r, j+2); ok {
					level, j = lv, end
					continue
				}
				j = len(r)
			case '[':
				if lv, end, ok := openLevel(r, j); ok {
					level, j = lv, end
				}
			}
		}
	}
	return inside
}

// reindentRawLines normalizes the verbatim body of a raw block. It strips the
// common leading whitespace shared by all non-blank lines (so the block is
// idempotent regardless of the source indentation) and then prefixes each
// non-blank line with base. Blank lines are emitted empty.
//
// Lines inside a Lua long bracket are emitted byte for byte: their leading and
// trailing whitespace is string content, not indentation.
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

	inside := longStringLines(strings.Join(src, "\n"))

	common := -1
	for i, line := range src {
		if inside[i] || strings.TrimSpace(line) == "" {
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
	for i, line := range src {
		if inside[i] {
			out = append(out, line)
			continue
		}
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
