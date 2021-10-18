package buildscriptutils

import (
	"os"
	"io"
	"log"
	"bufio"
)

func ConfirmHeader(file, expectedHeader string) {
	buildmain, err := os.Open(file)
	if err != nil {
		log.Fatalf("While confirming that %s is being run from its own directory, it could not open %s.  Are you running it from the correct directory?  Error opening the file: %s", file, file, err)
	}
	foundHeader, err := bufio.NewReader(buildmain).ReadString('\n')
	if err != nil {
		log.Fatalf("While confirming that %s is being run from its own directory, it could not read the first line.  Are you running it from the correct directory?  Error opening the file: %s", file, err)
	}

	if foundHeader != expectedHeader {
		log.Printf("Expected Header: \"%s\"", expectedHeader)
		log.Printf("Header found in %s: \"%s\"", file, foundHeader)
		log.Fatalf("While confirming that %s is being run from its own directory, it did not find the expected header.  Are you running it from the correct directory? ", file)
	}	
}


func CopyFile(src, dst string) error {
	fs, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fs.Close()
	fd, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = io.Copy(fd, fs)
	if err != nil {
		return err
	}

	return fd.Close()
}

func ErrChecker(f func(error)) (chkerr func(error), deferedCall func()) {
	errFound := false

	chkerr = func(err error) {
		if err != nil {
			errFound = true
			panic(err)
		}
	}

	deferedCall = func() {
		if errFound {
			v := recover()
			err, ok := v.(error)
			if ok {
			f(err)	
			} else {
				panic(v)
			}
		}
	}

	return
}
