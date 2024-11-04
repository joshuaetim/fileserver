/*
This script creates a file server in the specified directory
It also displays all the files sorting them by date added
*/
package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
)

//go:embed templates/*.html
var htmlFiles embed.FS

type config struct {
	Port string `env:"PORT" envDefault:"3000"`
}

type File struct {
	Name         string
	ModifiedDate time.Time
	Size         int
	IsDir        bool
}

func getFolderSize(dir string) int64 {
	var totalSize int64
	filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize
}

func displayFilesFromDir(dir string) ([]File, error) {
	fullpath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(fullpath)
	if err != nil {
		return nil, err
	}

	var files []File
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		file := File{
			Name:         entry.Name(),
			ModifiedDate: info.ModTime(),
			Size:         int(info.Size()),
			IsDir:        info.IsDir(),
		}
		files = append(files, file)
	}

	return files, nil
}

func main() {
	cfg, err := env.ParseAs[config]()
	if err != nil {
		log.Fatal(err)
	}

	home, err := htmlFiles.ReadFile("templates/home.html")
	if err != nil {
		log.Fatal(err)
	}

	tmpl, err := template.New("home").Parse(string(home))
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.Execute(w, nil)
		if err != nil {
			panic(err)
		}

		// print file details
		booksDir := "/Users/joshuaetim/Documents/books"
		files, err := displayFilesFromDir(booksDir)
		if err != nil {
			log.Println(err)
			return
		}
		if files == nil {
			log.Println("no files found")
			return
		}

		for _, file := range files {
			fmt.Println("name:", file.Name)
			fmt.Println("size:", file.Size)
			fmt.Println("date:", file.ModifiedDate)
			fmt.Println("dir:", file.IsDir)

		}
	})

	addr := fmt.Sprintf(":%s", cfg.Port)

	go func() {
		err = http.ListenAndServe(addr, mux)
		if err != nil {
			log.Fatal(err)
		}
	}()
	fmt.Println("server started on ", addr)

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

	<-stopSig
	fmt.Println("\rstop signal received, terminating...")

}
