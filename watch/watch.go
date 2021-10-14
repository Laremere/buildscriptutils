package watch

import (
	"time"
	"path/filepath"
	"io/fs"
)

type BuildFunc func(shouldRestart func()bool)

func WatchAndBuild(build BuildFunc, filesAndFolders ...string) {
	t := time.NewTicker(time.Second)

	walk := &walker{
		folders: filesAndFolders,
		last: make(map[string]time.Time),
		next: make(map[string]time.Time),
	}

	buildImmediate := false
	shouldRestart := func() bool {
		buildImmediate = walk.poll()
		return buildImmediate
	}

	for {
		for range t.C {
			if walk.poll() {
				break
			}
		}
		buildImmediate = true
		for buildImmediate {
			buildImmediate = false
			build(shouldRestart)
		}
	}
} 

// This should probably be using some proper file watcher, but this works so whatever.
type walker struct {
	folders []string
	last, next map[string]time.Time
	change     bool
}


func (w *walker) poll() bool {
	w.change = false

	if len(w.next) != 0 {
		panic("walker.next should be empty at start of poll")
	}

	for _, folder := range w.folders {
		err := filepath.Walk(folder, w.walkFunc)
		if err != nil {
			panic(err)
		}
	}

	if len(w.last) > 0 {
		w.change = true
		for k := range w.last {
			delete(w.last, k)
		}
	}

	w.last, w.next = w.next, w.last
	return w.change
}

func (w *walker) walkFunc(path string, info fs.FileInfo, err error) error {
	nextModTime := info.ModTime()
	w.next[path] = nextModTime
	lastModTime, ok := w.last[path]
	if ok {
		if !nextModTime.Equal(lastModTime) {
			w.change = true
		}
		delete(w.last, path)
	} else {
		w.change = true
	}

	return err
}
