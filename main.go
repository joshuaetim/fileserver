/*
This script creates a file server in the specified directory
It also displays all the files sorting them by date added
*/
package main

import (
	"context"
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
	Path         string
	Name         string
	ModifiedDate time.Time
	Size         int
	IsDir        bool
}

func getFolderSize(dir string, ctx context.Context) int64 {
	var totalSize int64
	done := make(chan struct{})

	go func() {
		defer close(done)
		filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if err != nil {
				return err
			}
			if !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})
	}()

	select {
	case <-done:
		return totalSize
	case <-ctx.Done():
		return 0
	}
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
		size := info.Size()
		name := entry.Name()
		fullEntryPath := filepath.Join(fullpath, name)

		if info.IsDir() {
			ctx, _ := context.WithTimeout(context.Background(), 50*time.Millisecond)
			retrievedSize := getFolderSize(fullEntryPath, ctx)
			if retrievedSize != 0 {
				size = retrievedSize
			}
		}

		file := File{
			Path:         fullEntryPath,
			Name:         name,
			ModifiedDate: info.ModTime(),
			Size:         int(size),
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

	var input string
	fmt.Print("enter path of book (press enter for current directory): ")
	fmt.Scanln(&input)

	if input == "" {
		input = "."
	}

	mux.Handle("/download", http.StripPrefix("/download", http.FileServer(http.Dir("/"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// print file details

		// booksDir := "/Users/joshuaetim/Documents/books"
		booksDir := input
		files, err := displayFilesFromDir(booksDir)
		if err != nil {
			log.Println(err)
			return
		}

		err = tmpl.Execute(w, struct{ Files []File }{Files: files})
		if err != nil {
			panic(err)
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
