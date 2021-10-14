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

type checkedError struct {
	err error
}

func (err *checkedError) Error() string {
	return err.err.Error()
}

func (err *checkedError) Unwrap() error {
	return err.err
}

func CheckErr(err error) {
	if err != nil {
		panic(&checkedError{err})
	}
}

func HandleCheckedError(f func(error)) {
	err := recover()
	if err == nil {
		return
	}
	err2, ok := err.(checkedError)
	if !ok {
		panic(err)
	}
	f(err2.err)
}