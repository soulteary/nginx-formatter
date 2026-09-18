package updater

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/soulteary/nginx-formatter/internal/nginx"
)

// Mode selects what the updater does with the formatted result.
type Mode int

const (
	// ModeWrite saves the result over the target. The default.
	ModeWrite Mode = iota
	// ModeCheck writes nothing and only reports which files would change.
	ModeCheck
	// ModeDiff writes nothing and prints a unified diff per changed file.
	ModeDiff
)

// ErrNeedsFormatting is returned by ModeCheck and ModeDiff when at least one
// file is not formatted, so the process exits non-zero and CI notices.
var ErrNeedsFormatting = errors.New("some files are not formatted")

// Out receives the per-file progress lines. It is a variable so `--quiet` can
// point it at io.Discard; errors are unaffected, they travel back as values
// and are reported by the caller on stderr.
var Out io.Writer = os.Stdout

// noticesFor picks where a scan's skip notices go.
//
// ModeWrite sends them to Out, so --quiet covers them like every other
// progress line. ModeCheck and ModeDiff own stdout -- it carries a file list
// or a patch that something else parses -- so theirs go to stderr, where they
// are still read by a human or a CI log but cannot be mistaken for a result.
func noticesFor(mode Mode) io.Writer {
	if mode == ModeWrite {
		return Out
	}
	return os.Stderr
}

// defaultFileMode is used when creating a file that does not already exist.
// Configuration files must stay readable by the account nginx runs its workers
// as, which is usually not the account that ran the formatter.
const defaultFileMode = os.FileMode(0644)

// ScanFiles lists the formattable files under rootDir, as paths relative to it
// (isFormattableName defines the set).
//
// Symbolic links are reported and skipped rather than followed. The standard
// Debian/Ubuntu layout links sites-enabled/x.conf to sites-available/x.conf,
// so following links would format the same file twice, and an atomic write
// through a link would replace the link with a regular file. When the target
// lives inside the scanned tree it is formatted via its own path anyway.
func ScanFiles(rootDir string) ([]string, error) {
	return scanFilesTo(rootDir, Out)
}

// scanFilesTo is ScanFiles with the destination for its skip notices made
// explicit. Those notices are diagnostics, not results: under --check and
// --diff stdout carries a file list or a patch that something else parses, so
// a stray "Skipping ..." line there would be read as one more path. They go to
// stderr in those modes instead of being dropped, because a skipped file is
// exactly what someone running --check in CI needs to hear about.
func scanFilesTo(rootDir string, notices io.Writer) ([]string, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("scandir is empty")
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	var files []string
	err = fs.WalkDir(root.FS(), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			// One unreadable entry must not abort the whole scan: a single
			// permission-denied directory would otherwise mean nothing at all
			// gets formatted.
			fmt.Fprintf(notices, "Skipping %s: %v\n", rel, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !isFormattableName(rel) {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			fmt.Fprintf(notices, "Skipping %s: symbolic link\n", rel)
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// isFormattableName reports whether a scanned path should be formatted.
//
// The set is "*.conf" plus one special case. Debian and Ubuntu's nginx package
// ships sites-available/default and links sites-enabled/default at it: on those
// systems it is the most common site file there is, and the only one with no
// extension at all, so a plain `nginx-formatter format` over /etc/nginx used to
// skip it without a word.
//
// The exception is anchored to both the name and its parent directory, so an
// unrelated file called "default" elsewhere in the tree is still left alone.
func isFormattableName(rel string) bool {
	if strings.HasSuffix(rel, ".conf") {
		return true
	}
	slashed := filepath.ToSlash(rel)
	if path.Base(slashed) != "default" {
		return false
	}
	switch path.Base(path.Dir(slashed)) {
	case "sites-enabled", "sites-available":
		return true
	}
	return false
}

// sameDir reports whether two paths name the same directory.
func sameDir(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

// scanFilesExcluding is ScanFiles with any file living under excludeDir
// dropped. excludeDir is ignored when it is not inside rootDir.
func scanFilesExcluding(rootDir, excludeDir string, notices io.Writer) ([]string, error) {
	files, err := scanFilesTo(rootDir, notices)
	if err != nil {
		return nil, err
	}
	if excludeDir == "" || sameDir(rootDir, excludeDir) {
		return files, nil
	}
	absRoot, err1 := filepath.Abs(rootDir)
	absOut, err2 := filepath.Abs(excludeDir)
	if err1 != nil || err2 != nil {
		return files, nil
	}
	rel, err := filepath.Rel(absRoot, absOut)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return files, nil // the output tree is not inside the input tree
	}

	prefix := rel + string(filepath.Separator)
	kept := files[:0]
	for _, f := range files {
		if strings.HasPrefix(f, prefix) {
			fmt.Fprintf(notices, "Skipping %s: inside the output directory\n", f)
			continue
		}
		kept = append(kept, f)
	}
	return kept, nil
}

// resolveTarget decides where the formatted single-file output should be
// written based on the -output value:
//   - output == ""          -> overwrite inputFile in place
//   - output is a directory -> filepath.Join(output, filepath.Base(inputFile))
//   - otherwise             -> treat output as a file path, creating its parent
//     directory when needed
func resolveTarget(inputFile string, output string) (string, error) {
	if output == "" {
		return inputFile, nil
	}

	if info, err := os.Stat(output); err == nil && info.IsDir() {
		return filepath.Join(output, filepath.Base(inputFile)), nil
	}

	if dir := filepath.Dir(output); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return "", err
		}
	}
	return output, nil
}

// tempBase bounds the basename embedded in a temporary file's name so the
// whole name stays inside NAME_MAX (255 on Linux and macOS).
//
// The temporary name is "." + base + ".tmp-" + randomness, which is 16-22
// bytes longer than base. Without this, a .conf file with a long but
// perfectly legal name could not be formatted at all: creating its temporary
// file failed with ENAMETOOLONG. Uniqueness comes from the random suffix, not
// from the basename, so truncating here is safe.
//
// The cut is pulled back to a rune boundary: macOS rejects filenames that are
// not valid UTF-8, so slicing mid-rune would trade one failure for another.
func tempBase(base string) string {
	const max = 200
	if len(base) <= max {
		return base
	}
	b := base[:max]
	for len(b) > 0 && !utf8.ValidString(b) {
		b = b[:len(b)-1]
	}
	return b
}

// resolveSymlink follows a symlink chain to the path it finally names, even
// when that path does not exist yet.
//
// filepath.EvalSymlinks fails on a dangling link, and treating that failure as
// "not a symlink" would make the rename below replace the link itself with a
// regular file. Pointing --output at a not-yet-created target is a legitimate
// layout, and the previous os.WriteFile followed the link and created it.
func resolveSymlink(path string) string {
	// Bounded so a symlink loop cannot spin here.
	for range 16 {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return path
		}
		dest, err := os.Readlink(path)
		if err != nil {
			return path
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(filepath.Dir(path), dest)
		}
		path = dest
	}
	return path
}

// writeFileAtomic replaces path's contents in a single step: it writes a
// temporary file in the same directory, then renames it over the target.
//
// The tool's default mode overwrites the user's live configuration, so a plain
// truncate-and-write would leave a half-written, unloadable config behind if
// the process were interrupted or the filesystem filled up. A rename is
// atomic, so the file is either the old content or the new one.
//
// A rename installs a NEW inode, so the target's permissions and ownership are
// re-applied explicitly. When ownership cannot be carried over — the caller is
// not privileged enough to chown — this falls back to rewriting the existing
// inode, which preserves every piece of metadata (ownership, ACLs, security
// labels) at the cost of atomicity. That is exactly the behaviour this helper
// replaced, so the fallback is never worse than not having it.
func writeFileAtomic(path string, data []byte) error {
	path = resolveSymlink(path)

	perm := defaultFileMode
	var owner *fileOwnership
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
		owner = ownerOf(info)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+tempBase(filepath.Base(path))+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	// Removing the temporary file is a no-op once the rename has succeeded.
	defer func() { _ = os.Remove(name) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, perm); err != nil { // #nosec G302 -- see defaultFileMode
		return err
	}
	if owner != nil {
		if err := os.Chown(name, owner.uid, owner.gid); err != nil {
			return os.WriteFile(path, data, perm) // #nosec G306 -- see defaultFileMode
		}
	}
	return os.Rename(name, path)
}

// createRootTemp opens a uniquely named temporary file alongside rel.
//
// os.Root has no CreateTemp, and a fixed name would be a trap: a run killed
// between creating the file and renaming it would leave the name taken, and
// every later run would then fail at O_EXCL with EEXIST until someone found
// and deleted the hidden file by hand.
func createRootTemp(root *os.Root, rel string, perm os.FileMode) (string, *os.File, error) {
	dir, base := filepath.Dir(rel), filepath.Base(rel)
	for range 100 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", nil, err
		}
		name := filepath.Join(dir, "."+tempBase(base)+".tmp-"+hex.EncodeToString(suffix[:]))
		f, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) // #nosec G302 -- see defaultFileMode
		if err == nil {
			return name, f, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, err
		}
	}
	return "", nil, fmt.Errorf("could not create a temporary file next to %s", rel)
}

// writeRootFileAtomic is writeFileAtomic scoped to an os.Root, used by the
// directory walker so a write cannot escape the output tree. It carries over
// permissions and ownership, and falls back the same way.
func writeRootFileAtomic(root *os.Root, rel string, data []byte) error {
	perm := defaultFileMode
	var owner *fileOwnership
	if info, err := root.Stat(rel); err == nil {
		perm = info.Mode().Perm()
		owner = ownerOf(info)
	}

	tmpRel, f, err := createRootTemp(root, rel, perm)
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(tmpRel) }()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := root.Chmod(tmpRel, perm); err != nil { // #nosec G302 -- see defaultFileMode
		return err
	}
	if owner != nil {
		if err := root.Chown(tmpRel, owner.uid, owner.gid); err != nil {
			return root.WriteFile(rel, data, perm) // #nosec G306 -- see defaultFileMode
		}
	}
	return root.Rename(tmpRel, rel)
}

// UpdateConfFile formats a single file. Unlike UpdateConfInDir it does not
// filter by the .conf suffix, so any file can be formatted.
func UpdateConfFile(inputFile string, output string, indent int, indentChar string, fn func(s string, indent int, char string) (string, error)) error {
	return UpdateConfFileMode(inputFile, output, indent, indentChar, ModeWrite, fn)
}

// UpdateConfFileMode is UpdateConfFile with an explicit Mode.
func UpdateConfFileMode(inputFile string, output string, indent int, indentChar string, mode Mode, fn func(s string, indent int, char string) (string, error)) error {
	// inputFile is provided directly by the user running this local CLI tool via
	// the --input flag, so reading it is the intended behavior rather than an
	// untrusted-path file-inclusion risk. Suppress gosec G304 accordingly.
	buf, err := os.ReadFile(inputFile) // #nosec G304
	if err != nil {
		fmt.Fprintf(Out, "Formatter Nginx Conf %s failed, can not open the file: %v\n", inputFile, err)
		return err
	}

	// Recorded now, reported only once something has actually been written.
	// Announcing the removal up front says "removing it" on the paths where
	// nothing is removed: a file that fails to parse is left exactly as it
	// was, BOM included.
	hadBOM := nginx.HasBOM(string(buf))

	modifiedData, err := fn(string(buf), indent, indentChar)
	if err != nil {
		fmt.Fprintf(Out, "Formatter Nginx Conf %s failed, can not format the file: %v\n", inputFile, err)
		return err
	}

	if mode != ModeWrite {
		if modifiedData == string(buf) {
			return nil
		}
		reportUnformatted(mode, inputFile, string(buf), modifiedData)
		return ErrNeedsFormatting
	}

	target, err := resolveTarget(inputFile, output)
	if err != nil {
		fmt.Fprintf(Out, "Formatter Nginx Conf %s failed, can not prepare the save dir: %v\n", inputFile, err)
		return err
	}

	// Rewriting a file whose content is already correct would bump its mtime
	// for nothing, waking inotify watchers, config reloaders and make.
	if target == inputFile && modifiedData == string(buf) {
		fmt.Fprintf(Out, "Formatter Nginx Conf %s Successed (already formatted)\n", target)
		return nil
	}

	if err := writeFileAtomic(target, []byte(modifiedData)); err != nil {
		fmt.Fprintf(Out, "Formatter Nginx Conf %s failed, can not save the file: %v\n", target, err)
		return err
	}

	if hadBOM {
		reportBOM(inputFile, target, target == inputFile)
	}
	fmt.Fprintf(Out, "Formatter Nginx Conf %s Successed\n", target)
	return nil
}

// reportBOM describes what happened to a byte order mark, after the fact.
//
// The two cases are genuinely different and the distinction matters to anyone
// reading the log: formatting in place removes the mark from the user's own
// file, while writing to a separate target leaves the input untouched and
// simply produces a copy without one. Saying "removing it" in the second case
// claims an edit to a file this run never opened for writing.
func reportBOM(input, target string, inPlace bool) {
	if inPlace {
		fmt.Fprintf(Out, "Formatter Nginx Conf %s had a UTF-8 BOM; removed it (nginx rejects a config that starts with one)\n", target)
		return
	}
	fmt.Fprintf(Out, "Formatter Nginx Conf %s had a UTF-8 BOM; %s was written without one, the input is unchanged\n", input, target)
}

func UpdateConfInDir(rootDir string, outputDir string, indent int, indentChar string, fn func(s string, indent int, char string) (string, error)) error {
	return UpdateConfInDirMode(rootDir, outputDir, indent, indentChar, ModeWrite, fn)
}

// UpdateConfInDirMode is UpdateConfInDir with an explicit Mode.
func UpdateConfInDirMode(rootDir string, outputDir string, indent int, indentChar string, mode Mode, fn func(s string, indent int, char string) (string, error)) error {
	notices := noticesFor(mode)

	// An output directory nested inside the input tree would otherwise be
	// walked as input on the next run, so each run re-ingested its own
	// previous output and nested one level deeper: out/, out/out/, ...
	files, err := scanFilesExcluding(rootDir, outputDir, notices)
	if err != nil {
		return err
	}
	sameTree := sameDir(rootDir, outputDir)

	inRoot, err := os.OpenRoot(rootDir)
	if err != nil {
		return err
	}
	defer func() { _ = inRoot.Close() }()

	// ModeCheck and ModeDiff write nothing, so they must not create the output
	// directory either — a read-only mode with a side effect is a trap in CI.
	var outRoot *os.Root
	if mode == ModeWrite {
		if err := os.MkdirAll(outputDir, 0750); err != nil {
			fmt.Fprintf(Out, "Formatter Nginx Conf failed, can not prepare the save dir %s: %v\n", outputDir, err)
			return err
		}
		outRoot, err = os.OpenRoot(outputDir)
		if err != nil {
			return err
		}
		defer func() { _ = outRoot.Close() }()
	}

	// A file that cannot be read or parsed is reported and skipped, so one bad
	// config no longer leaves the rest of the tree unprocessed. The failures
	// are surfaced as a single error once every file has been attempted.
	var failed []string
	unformatted := false
	for _, rel := range files {
		buf, err := inRoot.ReadFile(rel)
		if err != nil {
			fmt.Fprintf(Out, "Formatter Nginx Conf %s failed, can not open the file: %v\n", rel, err)
			failed = append(failed, rel)
			continue
		}

		hadBOM := nginx.HasBOM(string(buf))

		modifiedData, err := fn(string(buf), indent, indentChar)
		if err != nil {
			fmt.Fprintf(Out, "Formatter Nginx Conf %s failed, can not format the file: %v\n", rel, err)
			failed = append(failed, rel)
			continue
		}

		if mode != ModeWrite {
			if modifiedData != string(buf) {
				reportUnformatted(mode, rel, string(buf), modifiedData)
				unformatted = true
			}
			continue
		}

		if dir := filepath.Dir(rel); dir != "." {
			if err := outRoot.MkdirAll(dir, 0750); err != nil {
				fmt.Fprintf(Out, "Formatter Nginx Conf %s failed, can not prepare the save dir: %v\n", rel, err)
				failed = append(failed, rel)
				continue
			}
		}

		if sameTree && modifiedData == string(buf) {
			fmt.Fprintf(Out, "Formatter Nginx Conf %s Successed (already formatted)\n", rel)
			continue
		}

		if err := writeRootFileAtomic(outRoot, rel, []byte(modifiedData)); err != nil {
			fmt.Fprintf(Out, "Formatter Nginx Conf %s failed, can not save the file: %v\n", rel, err)
			failed = append(failed, rel)
			continue
		}

		if hadBOM {
			reportBOM(rel, filepath.Join(outputDir, rel), sameTree)
		}
		fmt.Fprintf(Out, "Formatter Nginx Conf %s Successed\n", rel)
	}

	if len(failed) > 0 {
		return fmt.Errorf("%d of %d file(s) could not be formatted: %s",
			len(failed), len(files), strings.Join(failed, ", "))
	}
	if unformatted {
		return ErrNeedsFormatting
	}
	return nil
}

// reportUnformatted prints what ModeCheck and ModeDiff owe the caller: a bare
// path for check (the gofmt -l shape, easy to pipe), a unified diff for diff.
//
// These two write to os.Stdout directly rather than through Out, and that is
// the one place in this package where the distinction matters. Out exists so
// --quiet can silence progress narration; this is not narration, it is the
// result the caller asked for. Routing it through Out would make
// `--check --quiet` print nothing at all and report only through the exit
// code, silently discarding requested output -- a worse failure than the noise
// --quiet was added to remove. The skip notices beside it are diagnostics and
// do go through Out, or to stderr in these modes.
func reportUnformatted(mode Mode, name, before, after string) {
	if mode == ModeDiff {
		fmt.Fprint(os.Stdout, UnifiedDiff(name, before, after))
		return
	}
	fmt.Fprintln(os.Stdout, name)
}
