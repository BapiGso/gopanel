package file

import (
	"log"
	"os"
	"path/filepath"
)

type (
	FileManage struct {
		path    string
		filearr []os.FileInfo
		opera   opera
	}
	opera struct {
		path string
	}
)

var (
	F = FileManage{}
)

func (f *FileManage) setpath(p string) {
	f.path = p
}

func (f *FileManage) read() {
	err := filepath.Walk(f.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		f.filearr = append(f.filearr, info)
		return nil
	})
	if err != nil {
		log.Println(err)
		return
	}
}

func (o *opera) zip() {

}

func (o *opera) unzip() {

}

func (o *opera) copy() {

}

func (o *opera) cut() {

}

func (o *opera) paste() {

}
