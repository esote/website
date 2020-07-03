package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/esote/openshim2"
)

type Page struct {
	Title     string
	Integrity string `json:"-"`
	Kind      uint8  `json:"-"`
}

type Project struct {
	Page
	Desc   string
	Badges []Badge

	// Optional custom path (such as external link).
	Path string

	Subprojects []Project `json:"-"`
}

type Badge struct {
	Name, Style string
}

type Log struct {
	Page
	Date string `json:"-"`
	Path string `json:"-"`
}

const (
	out = "gen/"
	in  = "tmpl/"
	css = "static/css/main.css"

	flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	fmode = 0604
	dmode = 0705

	kProject = 0
	kLog     = 1

	tProject = "project.tmpl"
	tLayout  = "layout.tmpl"
	tHome    = "home.tmpl"

	tLog   = "log.tmpl"
	tLogs  = "logs.tmpl"
	newest = 10
)

var (
	inProjects = filepath.Join(in, "projects")
	inLogs     = filepath.Join(in, "logs")

	projectTmpl *template.Template
	logTmpl     *template.Template
	integrity   string
)

func main() {
	var err error

	if err = os.MkdirAll(filepath.Dir(out), dmode); err != nil {
		log.Fatal(err)
	}
	if err = openshim2.Unveil(out, "rwc"); err != nil {
		log.Fatal(err)
	}
	if err = openshim2.Unveil(in, "r"); err != nil {
		log.Fatal(err)
	}
	if err = openshim2.Unveil(css, "r"); err != nil {
		log.Fatal(err)
	}
	if err = openshim2.Pledge("stdio rpath cpath wpath", ""); err != nil {
		log.Fatal(err)
	}

	projectTmpl, err = template.ParseFiles(filepath.Join(in, tProject),
		filepath.Join(in, tLayout))
	if err != nil {
		log.Fatal(err)
	}
	logTmpl, err = template.ParseFiles(filepath.Join(in, tLog),
		filepath.Join(in, tLayout))
	if err != nil {
		log.Fatal(err)
	}
	if integrity, err = cssIntegrity(); err != nil {
		log.Fatal(err)
	}

	var projects []Project
	if err = recurseProjects(inProjects, &projects); err != nil {
		log.Fatal(err)
	}
	var logs []Log
	if err = traverseLogs(inLogs, &logs); err != nil {
		log.Fatal(err)
	}
	if err = writeLogs(logs); err != nil {
		log.Fatal(err)
	}
	if err = writeHome(projects, &logs[0]); err != nil {
		log.Fatal(err)
	}
}

func cssIntegrity() (string, error) {
	f, err := os.Open(css)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sha := sha512.New()
	if _, err = io.Copy(sha, f); err != nil {
		return "", err
	}
	return "sha512-" + base64.StdEncoding.EncodeToString(sha.Sum(nil)), nil
}

func recurseProjects(dir string, projects *[]Project) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	infos, err := d.Readdir(-1)
	if err != nil {
		return err
	}
	if infos, err = sortInfos(infos); err != nil {
		return err
	}
	for _, info := range infos {
		path := filepath.Join(dir, info.Name())
		var project Project
		project.Page.Kind = kProject
		switch {
		case info.Mode().IsRegular():
			tmpl, err := parsePage(&project, path)
			if err != nil {
				return err
			}
			if tmpl == "" {
				if !isDescriptor(path) {
					break
				}
				continue
			}
			t, err := projectTmpl.Clone()
			if err != nil {
				return err
			}
			if t, err = t.Parse(tmpl); err != nil {
				return err
			}
			if path, err = normalizePath(path); err != nil {
				return err
			}
			if project.Path == "" {
				project.Path = path
			}
			project.Page.Integrity = integrity
			path = filepath.Join(out, path)
			if err = writePage(path, t, &project); err != nil {
				return err
			}
		case info.IsDir():
			desc := path + ".tmpl"
			if _, err = os.Stat(desc); err != nil {
				return err
			}
			if _, err = parsePage(&project, desc); err != nil {
				return err
			}
			err = recurseProjects(path, &project.Subprojects)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected file '%s'", path)
		}
		*projects = append(*projects, project)
	}
	return nil
}

func isDescriptor(path string) bool {
	info, err := os.Stat(strings.TrimSuffix(path, filepath.Ext(path)))
	return err == nil && info.IsDir()
}

func sortInfos(infos []os.FileInfo) ([]os.FileInfo, error) {
	index := make(map[os.FileInfo]int64, len(infos))
	var err error
	for i := 0; i < len(infos); i++ {
		prefix := strings.SplitN(infos[i].Name(), "-", 2)[0]
		if prefix != "" && prefix[0] == '.' {
			infos = append(infos[:i], infos[i+1:]...)
			i--
			continue
		}
		index[infos[i]], err = strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(infos, func(i, j int) bool {
		return index[infos[i]] > index[infos[j]]
	})
	return infos, nil
}

func normalizePath(path string) (string, error) {
	path, err := filepath.Rel(inProjects, path)
	if err != nil {
		return "", err
	}
	parts := strings.Split(path, string(os.PathSeparator))
	for i, part := range parts {
		split := strings.SplitN(part, "-", 2)
		if len(split) == 2 {
			parts[i] = split[1]
		}
	}
	path = filepath.Join(parts...)
	path = strings.TrimSuffix(path, filepath.Ext(path)) + ".html"
	return path, nil
}

func writePage(file string, t *template.Template, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(file), dmode); err != nil {
		return err
	}
	f, err := os.OpenFile(file, flags, fmode)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.ExecuteTemplate(f, tLayout, data)
}

func parsePage(meta interface{}, file string) (string, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	split := bytes.SplitN(b, []byte("\n\n"), 2)
	if len(split) == 0 {
		return "", errors.New("file format invalid")
	}
	if err = json.Unmarshal(split[0], meta); err != nil {
		return "", err
	}
	if len(split) == 1 {
		return "", nil
	}
	return string(split[1]), nil
}

func writeHome(projects []Project, latestLog *Log) error {
	f, err := os.OpenFile(filepath.Join(out, "index.html"), flags, fmode)
	if err != nil {
		return err
	}
	defer f.Close()
	t, err := template.ParseFiles(filepath.Join(in, tLayout),
		filepath.Join(in, tHome))
	if err != nil {
		return err
	}
	var pof bytes.Buffer
	if _, err = io.Copy(&pof, os.Stdin); err != nil {
		return err
	}
	data := struct {
		Page
		Projects []Project
		Log      *Log
		Time     string
		Pof      string
	}{
		Page:     Page{Integrity: integrity},
		Log:      latestLog,
		Projects: projects,
		Time:     time.Now().UTC().Format("Jan 2, 2006"),
		Pof:      pof.String(),
	}
	return t.Execute(f, data)
}

func traverseLogs(dir string, logs *[]Log) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	infos, err := d.Readdir(-1)
	if err != nil {
		return err
	}
	if infos, err = sortInfos(infos); err != nil {
		return err
	}
	for _, info := range infos {
		path := filepath.Join(dir, info.Name())
		var log Log
		log.Page.Kind = kLog
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unexpected file '%s'", path)
		}
		tmpl, err := parsePage(&log, path)
		if err != nil {
			return err
		}
		if tmpl == "" {
			return fmt.Errorf("log without content '%s'", path)
		}
		t, err := logTmpl.Clone()
		if err != nil {
			return err
		}
		if t, err = t.Parse(tmpl); err != nil {
			return err
		}
		path = info.Name()
		prefix := strings.SplitN(path, "-", 2)[0]
		date, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return err
		}
		if date == 0 {
			log.Date = "aeons ago..."
		} else {
			log.Date = time.Unix(date, 0).UTC().Format("Jan 2, 2006")
		}
		log.Page.Integrity = integrity
		path = filepath.Join("logs", path)
		path = strings.TrimSuffix(path, filepath.Ext(path)) + ".html"
		log.Path = path
		path = filepath.Join(out, path)
		if err = writePage(path, t, &log); err != nil {
			return err
		}
		*logs = append(*logs, log)
	}
	return nil
}

func writeLogs(logs []Log) error {
	f, err := os.OpenFile(filepath.Join(out, "logs.html"), flags, fmode)
	if err != nil {
		return err
	}
	defer f.Close()
	t, err := template.ParseFiles(filepath.Join(in, tLayout),
		filepath.Join(in, tLogs))
	if err != nil {
		return err
	}
	data := struct {
		Page
		Logs []Log
	}{
		Page: Page{
			Title:     "Logs",
			Integrity: integrity,
		},
		Logs: logs,
	}
	return t.Execute(f, data)
}
