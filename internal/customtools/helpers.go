package customtools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResolveCode returns inline code or reads the source file.
func ResolveCode(inlineCode, filePath string, required bool) (string, error) {
	inlineCode = strings.TrimSpace(inlineCode)
	filePath = strings.TrimSpace(filePath)

	if inlineCode != "" && filePath != "" {
		return "", fmt.Errorf("use either inline code or --file, not both")
	}
	if inlineCode != "" {
		return inlineCode, nil
	}
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read code file: %w", err)
		}
		return string(data), nil
	}
	if required {
		return "", fmt.Errorf("code is required: pass --file or --code")
	}
	return "", nil
}

// ParseJSONObject parses JSON from either an inline string or a file path.
func ParseJSONObject(inlineJSON, filePath string) (map[string]interface{}, bool, error) {
	inlineJSON = strings.TrimSpace(inlineJSON)
	filePath = strings.TrimSpace(filePath)

	if inlineJSON != "" && filePath != "" {
		return nil, false, fmt.Errorf("use either inline JSON or a file, not both")
	}
	if inlineJSON == "" && filePath == "" {
		return nil, false, nil
	}

	var raw []byte
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, false, fmt.Errorf("read JSON file: %w", err)
		}
		raw = data
	} else {
		raw = []byte(inlineJSON)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, false, fmt.Errorf("parse JSON object: %w", err)
	}
	return out, true, nil
}

// ParseList parses comma-separated or newline-separated values into a stable list.
func ParseList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return DedupSorted(out)
}

// DedupSorted sorts and deduplicates a string slice.
func DedupSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

// InferThirdPartyImports extracts top-level third-party imports from Python code.
func InferThirdPartyImports(code string, scriptPath string) []string {
	candidates := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(code))

	for scanner.Scan() {
		line := stripPythonComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "import "):
			remainder := strings.TrimSpace(strings.TrimPrefix(line, "import "))
			for _, spec := range strings.Split(remainder, ",") {
				root := pythonImportRoot(spec)
				if shouldIncludeImport(root, scriptPath) {
					candidates[root] = struct{}{}
				}
			}
		case strings.HasPrefix(line, "from "):
			if strings.HasPrefix(line, "from .") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 4 || fields[2] != "import" {
				continue
			}
			root := pythonImportRoot(fields[1])
			if shouldIncludeImport(root, scriptPath) {
				candidates[root] = struct{}{}
			}
		}
	}

	out := make([]string, 0, len(candidates))
	for candidate := range candidates {
		out = append(out, candidate)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func stripPythonComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		return strings.TrimSpace(line[:idx])
	}
	return line
}

func pythonImportRoot(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}

	fields := strings.Fields(spec)
	if len(fields) == 0 {
		return ""
	}

	module := strings.TrimSpace(fields[0])
	if module == "" || strings.HasPrefix(module, ".") {
		return ""
	}
	return strings.Split(module, ".")[0]
}

func shouldIncludeImport(root string, scriptPath string) bool {
	if root == "" {
		return false
	}
	if _, ok := pythonStdlibModules[root]; ok {
		return false
	}
	if isLocalModule(root, scriptPath) {
		return false
	}
	return true
}

func isLocalModule(root string, scriptPath string) bool {
	scriptPath = strings.TrimSpace(scriptPath)
	if scriptPath == "" {
		return false
	}

	scriptDir := filepath.Dir(scriptPath)
	roots := []string{scriptDir}
	projectRoot := findPythonProjectRoot(scriptDir)
	if projectRoot != "" && projectRoot != scriptDir {
		roots = append(roots, projectRoot)
	}

	for _, base := range roots {
		if pathExists(filepath.Join(base, root+".py")) {
			return true
		}
		if isPythonPackageDir(filepath.Join(base, root)) {
			return true
		}
	}
	return false
}

func findPythonProjectRoot(dir string) string {
	current := dir
	for {
		for _, marker := range []string{"pyproject.toml", "requirements.txt", "setup.py", ".git"} {
			if pathExists(filepath.Join(current, marker)) {
				return current
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return dir
		}
		current = parent
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isPythonPackageDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	return pathExists(filepath.Join(path, "__init__.py")) || pathExists(filepath.Join(path, "py.typed"))
}

var pythonStdlibModules = map[string]struct{}{
	"__future__":       {},
	"__hello__":        {},
	"_abc":             {},
	"_aix_support":     {},
	"_ast":             {},
	"_asyncio":         {},
	"_bisect":          {},
	"_blake2":          {},
	"_bootsubprocess":  {},
	"_bz2":             {},
	"_codecs":          {},
	"_collections":     {},
	"_collections_abc": {},
	"_compat_pickle":   {},
	"_compression":     {},
	"_contextvars":     {},
	"_crypt":           {},
	"_csv":             {},
	"_ctypes":          {},
	"_curses":          {},
	"_datetime":        {},
	"_dbm":             {},
	"_decimal":         {},
	"_elementtree":     {},
	"_functools":       {},
	"_gdbm":            {},
	"_hashlib":         {},
	"_heapq":           {},
	"_imp":             {},
	"_io":              {},
	"_json":            {},
	"_locale":          {},
	"_lsprof":          {},
	"_lzma":            {},
	"_markupbase":      {},
	"_md5":             {},
	"_msi":             {},
	"_multibytecodec":  {},
	"_multiprocessing": {},
	"_opcode":          {},
	"_operator":        {},
	"_osx_support":     {},
	"_pickle":          {},
	"_posixshmem":      {},
	"_posixsubprocess": {},
	"_py_abc":          {},
	"_pydecimal":       {},
	"_pyio":            {},
	"_queue":           {},
	"_random":          {},
	"_sha1":            {},
	"_sha256":          {},
	"_sha3":            {},
	"_sha512":          {},
	"_signal":          {},
	"_sitebuiltins":    {},
	"_socket":          {},
	"_sqlite3":         {},
	"_sre":             {},
	"_ssl":             {},
	"_stat":            {},
	"_statistics":      {},
	"_string":          {},
	"_strptime":        {},
	"_struct":          {},
	"_symtable":        {},
	"_sysconfigdata":   {},
	"_thread":          {},
	"_threading_local": {},
	"_tkinter":         {},
	"_tokenize":        {},
	"_tracemalloc":     {},
	"_uuid":            {},
	"_warnings":        {},
	"_weakref":         {},
	"_weakrefset":      {},
	"abc":              {},
	"aifc":             {},
	"antigravity":      {},
	"argparse":         {},
	"array":            {},
	"ast":              {},
	"asynchat":         {},
	"asyncio":          {},
	"asyncore":         {},
	"atexit":           {},
	"audioop":          {},
	"base64":           {},
	"bdb":              {},
	"binascii":         {},
	"binhex":           {},
	"bisect":           {},
	"builtins":         {},
	"bz2":              {},
	"cProfile":         {},
	"calendar":         {},
	"cgi":              {},
	"cgitb":            {},
	"chunk":            {},
	"cmath":            {},
	"cmd":              {},
	"code":             {},
	"codecs":           {},
	"codeop":           {},
	"collections":      {},
	"colorsys":         {},
	"compileall":       {},
	"concurrent":       {},
	"configparser":     {},
	"contextlib":       {},
	"contextvars":      {},
	"copy":             {},
	"copyreg":          {},
	"crypt":            {},
	"csv":              {},
	"ctypes":           {},
	"curses":           {},
	"dataclasses":      {},
	"datetime":         {},
	"dbm":              {},
	"decimal":          {},
	"difflib":          {},
	"dis":              {},
	"distutils":        {},
	"doctest":          {},
	"email":            {},
	"encodings":        {},
	"ensurepip":        {},
	"enum":             {},
	"errno":            {},
	"faulthandler":     {},
	"fcntl":            {},
	"filecmp":          {},
	"fileinput":        {},
	"fnmatch":          {},
	"fractions":        {},
	"ftplib":           {},
	"functools":        {},
	"gc":               {},
	"genericpath":      {},
	"getopt":           {},
	"getpass":          {},
	"gettext":          {},
	"glob":             {},
	"graphlib":         {},
	"grp":              {},
	"gzip":             {},
	"hashlib":          {},
	"heapq":            {},
	"hmac":             {},
	"html":             {},
	"http":             {},
	"idlelib":          {},
	"imaplib":          {},
	"imghdr":           {},
	"imp":              {},
	"importlib":        {},
	"inspect":          {},
	"io":               {},
	"ipaddress":        {},
	"itertools":        {},
	"json":             {},
	"keyword":          {},
	"lib2to3":          {},
	"linecache":        {},
	"locale":           {},
	"logging":          {},
	"lzma":             {},
	"mailbox":          {},
	"mailcap":          {},
	"marshal":          {},
	"math":             {},
	"mimetypes":        {},
	"modulefinder":     {},
	"msilib":           {},
	"msvcrt":           {},
	"multiprocessing":  {},
	"netrc":            {},
	"nis":              {},
	"nntplib":          {},
	"nt":               {},
	"ntpath":           {},
	"nturl2path":       {},
	"numbers":          {},
	"opcode":           {},
	"operator":         {},
	"optparse":         {},
	"os":               {},
	"pathlib":          {},
	"pdb":              {},
	"pickle":           {},
	"pickletools":      {},
	"pipes":            {},
	"pkgutil":          {},
	"platform":         {},
	"plistlib":         {},
	"poplib":           {},
	"posix":            {},
	"posixpath":        {},
	"pprint":           {},
	"profile":          {},
	"pstats":           {},
	"pty":              {},
	"pwd":              {},
	"py_compile":       {},
	"pyclbr":           {},
	"pydoc":            {},
	"pydoc_data":       {},
	"pyexpat":          {},
	"queue":            {},
	"quopri":           {},
	"random":           {},
	"re":               {},
	"readline":         {},
	"reprlib":          {},
	"resource":         {},
	"rlcompleter":      {},
	"runpy":            {},
	"sched":            {},
	"secrets":          {},
	"select":           {},
	"selectors":        {},
	"shelve":           {},
	"shlex":            {},
	"shutil":           {},
	"signal":           {},
	"site":             {},
	"smtpd":            {},
	"smtplib":          {},
	"sndhdr":           {},
	"socket":           {},
	"socketserver":     {},
	"spwd":             {},
	"sqlite3":          {},
	"sre_compile":      {},
	"sre_constants":    {},
	"sre_parse":        {},
	"ssl":              {},
	"stat":             {},
	"statistics":       {},
	"string":           {},
	"stringprep":       {},
	"struct":           {},
	"subprocess":       {},
	"sunau":            {},
	"symtable":         {},
	"sys":              {},
	"sysconfig":        {},
	"syslog":           {},
	"tabnanny":         {},
	"tarfile":          {},
	"telnetlib":        {},
	"tempfile":         {},
	"termios":          {},
	"test":             {},
	"textwrap":         {},
	"this":             {},
	"threading":        {},
	"time":             {},
	"timeit":           {},
	"tkinter":          {},
	"token":            {},
	"tokenize":         {},
	"tomllib":          {},
	"trace":            {},
	"traceback":        {},
	"tracemalloc":      {},
	"tty":              {},
	"turtle":           {},
	"turtledemo":       {},
	"types":            {},
	"typing":           {},
	"unicodedata":      {},
	"unittest":         {},
	"urllib":           {},
	"uu":               {},
	"uuid":             {},
	"venv":             {},
	"warnings":         {},
	"wave":             {},
	"weakref":          {},
	"webbrowser":       {},
	"winreg":           {},
	"winsound":         {},
	"wsgiref":          {},
	"xdrlib":           {},
	"xml":              {},
	"xmlrpc":           {},
	"zipapp":           {},
	"zipfile":          {},
	"zipimport":        {},
	"zlib":             {},
	"zoneinfo":         {},
}
