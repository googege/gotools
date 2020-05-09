package gotools

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"sync"
	"os"
)
// put a root path,and return whole file path.
func ReadFile(root string) ([]string, error) {
	ma, err := Read(root)
	var path []string
	for k := range ma {
		path = append(path, k)

	}
	sort.Strings(path)
	for k,v := range path {
		t := fmt.Sprintf("%x",ma[v])
		path[k]= t+":--->"+v
	}
	return path, err
}

// return the paths.
func WalkFile(done chan bool, root string) (<-chan string, <-chan error) {
	paths := make(chan string)
	err := make(chan error, 1)
	go func() {
		defer close(paths)
		err <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// if is not file ,return nil.
			if !info.Mode().IsRegular() {
				return nil
			}
			select {
			case paths <- path:
			case <-done:
				return fmt.Errorf("done")
			}
			return nil
		})
	}()
	return paths, err
}

// return result .
type resultValue struct {
	path string
	data [md5.Size]byte
	err error
}

// file recipient.
func FileRecipient(done <-chan bool, paths <-chan string, result chan *resultValue) {
	for path := range paths {
		data, err := ioutil.ReadFile(path)
		select {
		case result <- &resultValue{
			path: path,
			data: md5.Sum(data),
			err:  err,
		}:
		case <-done:
			return
		}
	}
}
// return a map[string][md5.Size]byte
// string is path
// [md5.Size]byte mean the file's only Identifier
func Read(root string) (map[string][md5.Size]byte, error) {
	done := make(chan bool)
	defer close(done)
	paths, err := WalkFile(done, root)
	c := make(chan *resultValue)
	var wg = sync.WaitGroup{}
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			FileRecipient(done, paths, c)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c)
	}()
	ma := make(map[string][md5.Size]byte)
	for k := range c {
		if k.err != nil {
			return nil, k.err
		}
		ma[k.path] = k.data
	}
	if err := <-err; err != nil {
		return nil, err
	}
	return ma, nil
}

