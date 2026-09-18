package updater

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

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
			fmt.Printf("Skipping %s: %v\n", rel, err)
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
			fmt.Printf("Skipping %s: symbolic link\n", rel)
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

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
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
		name := filepath.Join(dir, "."+base+".tmp-"+hex.EncodeToString(suffix[:]))
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
	// inputFile is provided directly by the user running this local CLI tool via
	// the --input flag, so reading it is the intended behavior rather than an
	// untrusted-path file-inclusion risk. Suppress gosec G304 accordingly.
	buf, err := os.ReadFile(inputFile) // #nosec G304
	if err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not open the file: %v\n", inputFile, err)
		return err
	}

	modifiedData, err := fn(string(buf), indent, indentChar)
	if err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not format the file: %v\n", inputFile, err)
		return err
	}

	target, err := resolveTarget(inputFile, output)
	if err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not prepare the save dir: %v\n", inputFile, err)
		return err
	}

	if err := writeFileAtomic(target, []byte(modifiedData)); err != nil {
		fmt.Printf("Formatter Nginx Conf %s failed, can not save the file: %v\n", target, err)
		return err
	}

	fmt.Printf("Formatter Nginx Conf %s Successed\n", target)
	return nil
}

func UpdateConfInDir(rootDir string, outputDir string, indent int, indentChar string, fn func(s string, indent int, char string) (string, error)) error {
	files, err := ScanFiles(rootDir)
	if err != nil {
		return err
	}

	inRoot, err := os.OpenRoot(rootDir)
	if err != nil {
		return err
	}
	defer func() { _ = inRoot.Close() }()

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		fmt.Printf("Formatter Nginx Conf failed, can not prepare the save dir %s: %v\n", outputDir, err)
		return err
	}
	outRoot, err := os.OpenRoot(outputDir)
	if err != nil {
		return err
	}
	defer func() { _ = outRoot.Close() }()

	// A file that cannot be read or parsed is reported and skipped, so one bad
	// config no longer leaves the rest of the tree unprocessed. The failures
	// are surfaced as a single error once every file has been attempted.
	var failed []string
	for _, rel := range files {
		buf, err := inRoot.ReadFile(rel)
		if err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not open the file: %v\n", rel, err)
			failed = append(failed, rel)
			continue
		}

		modifiedData, err := fn(string(buf), indent, indentChar)
		if err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not format the file: %v\n", rel, err)
			failed = append(failed, rel)
			continue
		}

		if dir := filepath.Dir(rel); dir != "." {
			if err := outRoot.MkdirAll(dir, 0750); err != nil {
				fmt.Printf("Formatter Nginx Conf %s failed, can not prepare the save dir: %v\n", rel, err)
				failed = append(failed, rel)
				continue
			}
		}

		if err := writeRootFileAtomic(outRoot, rel, []byte(modifiedData)); err != nil {
			fmt.Printf("Formatter Nginx Conf %s failed, can not save the file: %v\n", rel, err)
			failed = append(failed, rel)
			continue
		}

		fmt.Printf("Formatter Nginx Conf %s Successed\n", rel)
	}

	if len(failed) > 0 {
		return fmt.Errorf("%d of %d file(s) could not be formatted: %s",
			len(failed), len(files), strings.Join(failed, ", "))
	}
	return nil
}
