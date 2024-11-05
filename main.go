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
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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

	var input string
	homeDir, _ := os.UserHomeDir()
	fmt.Printf("enter path of book (relative to %s): ", homeDir)
	fmt.Scanln(&input)

	input = filepath.Join(homeDir, input)
	_, err = os.Stat(input)
	if err != nil {
		log.Fatal(err)
	}

	localIP, err := getLocalIpAddr()
	if err != nil {
		log.Fatal(err)
	}
	localIP = fmt.Sprintf("%s:%s", localIP, cfg.Port)

	mux := http.NewServeMux()
	mux.Handle("/download/", http.StripPrefix("/download", http.FileServer(http.Dir("/"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		booksDir := input
		files, err := displayFilesFromDir(booksDir)
		if err != nil {
			log.Println(err)
			return
		}

		err = tmpl.Execute(w, struct {
			Files []File
			Addr  string
		}{Files: files, Addr: localIP})
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

func getLocalIpAddr() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			ipStr := ip.String()
			if ip != nil && ip.To4() != nil && !strings.HasPrefix(ipStr, "127") {
				return ipStr, nil
			}
		}
	}
	return "", fmt.Errorf("ip not found")
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
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		size := info.Size()
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
