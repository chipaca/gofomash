// © 2021 John Lenton. MIT licensed. from https://chipaca.com/gofomash
package main // import "chipaca.com/gofomash"

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

const version = "0.0.2"

var (
	writeMode   bool
	listMode    bool
	showRaw     bool
	showRules   bool
	carryOn     bool
	dryRun      bool
	showVersion bool
	maxEll      int
	rootDir     string
	ruleFile    string
	extraRules  multi
	excludes    multi
	binary      string
)

func init() {
	log.SetFlags(0)

	flag.BoolVar(&showVersion, "v", false, "print version and exit")
	flag.BoolVar(&writeMode, "w", false, "write fixes to files instead of printing the diff and failing")
	flag.BoolVar(&listMode, "l", false, "list failing files instead of printing the diff")
	flag.BoolVar(&showRaw, "show-raw", false, "show raw rules, and exit")
	flag.BoolVar(&showRules, "show", false, "show post-processed rules, and exit")
	flag.BoolVar(&carryOn, "c", false, "continue past the first failure")
	flag.BoolVar(&dryRun, "n", false,
		"run all the rules, but against an empty go file\n(useful for validating your rules)")
	flag.IntVar(&maxEll, "m", 7, "expand ellipsised rules up to this many vars")
	flag.StringVar(&rootDir, "root", ".", "from where to start the walk")
	flag.StringVar(&ruleFile, "f", "", "file from which to get rules")
	flag.Var(&extraRules, "r", "add an individual rule; can be specified multiple times")
	flag.Var(&excludes, "x", `path to exclude, relative to root; can be specified multiple times
(default {"vendor", ".git", "build"})`)
	flag.StringVar(&binary, "binary", "gofmt", `binary to use (e.g. "goimports" or "gofumpt")`)
}

const (
	OK  = "\r\033[38;5;034m✓\033[0m"
	NOK = "\r\033[38;5;124m×\033[0m"
	ERR = "\r\033[38;5;124mℯ\033[0m"
)

// globals are cool
var failed bool

func shorten(s string) string {
	if len(s) <= 40 {
		return s
	}
	w, _, _ := term.GetSize(2)
	w -= 13
	if w < 40 {
		w = 40
	}
	if len(s) <= w {
		return s
	}
	return s[:w-1] + "…"
}

func run(args []string, rule string) {
	if rule == "-s" {
		fmt.Fprintf(os.Stderr, "› %s -s", binary)
	} else {
		fmt.Fprintf(os.Stderr, "› %s -r '%s'", binary, shorten(rule))
		args[1] = rule
	}

	cmd := exec.Command(binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, ERR)
		os.Stderr.Write(out)
		log.Fatal(err)
	}
	if len(out) > 0 {
		fmt.Fprintln(os.Stderr, NOK)
		os.Stdout.Write(out)
		if !carryOn {
			log.Fatalf("rule '%s' failed", rule)
		}
		failed = true
	} else {
		fmt.Fprintln(os.Stderr, OK)
	}
}

func getRawRules() []string {
	var lines [][]byte
	if ruleFile != "" {
		buf, err := ioutil.ReadFile(ruleFile)
		if err != nil {
			log.Fatal(err)
		}
		lines = bytes.Split(buf, []byte{'\n'})
	}

	rawRules := make([]string, len(lines)+len(extraRules))
	for i, line := range lines {
		rawRules[i] = string(line)
	}
	copy(rawRules[len(lines):], extraRules)

	return rawRules
}

var rx = regexp.MustCompile(`^(.*\PL)(\p{Ll})…(.* -> .*\PL)(\p{Ll})…(.*)$`)

func cook(rule string) []string {
	rule = strings.TrimSpace(rule)
	if rule == "" || rule[0] == '#' {
		return nil
	}
	subs := rx.FindStringSubmatch(rule)
	if len(subs) == 0 {
		// not an ellipsised rule
		return []string{rule}
	}
	if subs[2] != subs[4] {
		log.Fatalf("Bad rule %q: ellipsised character should be the same on both sides.", rule)
	}
	start, _ := utf8.DecodeRuneInString(subs[2])
	if start < 'a' || start > 'z' {
		log.Fatalf("Bad rule %q: ellipsised character should be in [a-z] (for now).", rule)
	}
	end := start + rune(maxEll)
	if end > 'z' {
		end = 'z'
	}
	rules := make([]string, int(end-start))
	pat := string(start)
	for i := range rules {
		rules[i] = subs[1] + pat + subs[3] + pat + subs[5]
		pat += ", " + string(start+rune(i+1))
	}
	return rules
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if excludes == nil {
		excludes = []string{"vendor", ".git", "build"}
	}

	if maxEll > 25 {
		maxEll = 25
	}

	mode := "-d"
	if writeMode {
		mode = "-w"
	}
	if listMode {
		mode = "-l"
	}

	rawRules := getRawRules()
	if showRaw {
		for _, r := range rawRules {
			fmt.Println(r)
		}
		os.Exit(0)
	}

	var rules []string
	for _, rule := range rawRules {
		rules = append(rules, cook(rule)...)
	}

	if showRules {
		for _, rule := range rules {
			fmt.Println(rule)
		}
		os.Exit(0)
	}

	excluded := make(map[string]bool, len(excludes))
	for _, x := range excludes {
		excluded[filepath.Join(rootDir, x)] = true
	}

	args := []string{"-s", "-s", mode}
	if dryRun {
		f, err := ioutil.TempFile("", "")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		defer os.Remove(f.Name())

		fmt.Fprintln(f, "package foo")
		args = append(args, f.Name())
	} else if extraArgs := flag.Args(); len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	} else {
		n := 6
		if err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if excluded[path] {
					return filepath.SkipDir
				}
			} else {
				if excluded[path] {
					return nil
				}
				if path[0] != '.' && filepath.Ext(path) == ".go" {
					args = append(args, path)
					n += len(path)
				}
			}
			return nil
		}); err != nil {
			log.Fatal(err)
		}
		n += len(args)
		if n > 2_000_000 {
			// limit obtained from 'xargs --show-limits'; ymmv
			log.Fatal("argument length dangeroulsy close to limit, this code needs work!")
		}
	}

	run(args, "-s")

	args[0] = "-r"
	for _, rule := range rules {
		run(args, rule)
	}

	if failed {
		fmt.Fprintln(os.Stderr, "Crushing failure and despair.")
		os.Exit(1)
	}
}
