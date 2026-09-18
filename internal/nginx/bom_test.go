package nginx

import (
	"strings"
	"testing"
)

// TestParseStripsBOM covers the UTF-8 byte order mark Windows editors add.
// The lexer had no idea what it was, so it became part of the first token:
// the directive name was "\ufeffserver", not "server". The file formatted
// "successfully" and nginx then refused it with "unknown directive".
func TestParseStripsBOM(t *testing.T) {
	cfg, err := Parse(BOM + "server {\nlisten 80;\n}\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	blk, ok := cfg.Nodes[0].(*Block)
	if !ok {
		t.Fatalf("expected a block, got %#v", cfg.Nodes[0])
	}
	if blk.Name != "server" {
		t.Errorf("directive name = %q, want %q (the BOM was left attached)", blk.Name, "server")
	}

	out := Format(cfg, 2, " ")
	if strings.Contains(out, BOM) {
		t.Errorf("the BOM survived into the output: %q", out)
	}
	if !strings.HasPrefix(out, "server {") {
		t.Errorf("output does not start with the directive: %q", out)
	}
}

// TestParseBOMBeforeComment covers the other position a BOM lands in: ahead of
// a leading comment, where it used to make the whole parse fail.
func TestParseBOMBeforeComment(t *testing.T) {
	out, err := Parse(BOM + "# a comment\nuser nginx;\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := Format(out, 2, " "); !strings.HasPrefix(got, "# a comment") {
		t.Errorf("output does not start with the comment: %q", got)
	}
}

func TestHasBOM(t *testing.T) {
	if !HasBOM(BOM + "server {}") {
		t.Error("HasBOM missed a leading BOM")
	}
	if HasBOM("server {}") {
		t.Error("HasBOM reported one where there is none")
	}
	// A BOM sequence in the middle of a file is ordinary content.
	if HasBOM("server {} " + BOM) {
		t.Error("HasBOM matched a mark that is not at the start")
	}
}
