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
	"sort"
	"strings"
	"syscall"
	"time"
)

//go:embed templates/*.html
var htmlFiles embed.FS

type File struct {
	Path         string
	Name         string
	ModifiedDate time.Time
	Size         string
	IsDir        bool
}

var SortBy string = "date"

func ByModifiedDate(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[j].ModifiedDate.Before(files[i].ModifiedDate)
	})
}

func ByName(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	home, err := htmlFiles.ReadFile("templates/home.html")
	if err != nil {
		log.Fatal(err)
	}

	tmpl, err := template.New("home").Funcs(template.FuncMap{"stripLast": getDirectory}).Parse(string(home))
	if err != nil {
		log.Fatal(err)
	}

	var input string
	homeDir, _ := os.UserHomeDir()
	fmt.Printf("enter path of book: (press enter for default home)  %s/", homeDir)
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
	localIP = fmt.Sprintf("%s:%s", localIP, port)

	mux := http.NewServeMux()

	mux.Handle("/{name}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this will serve /Book_Name.epub (because Kobo browser does not support my headers)
		// name is just the base name of the book
		// a path will be in the query featuring the actual full path of the book

		fullpath := r.URL.Query().Get("path")
		if fullpath == "" {
			return
		}

		// // ensure they don't go past root directory
		if !strings.HasPrefix(fullpath, homeDir) {
			return
		}

		// // remove the home directory
		prefixLen := len(homeDir)

		newRequest := r.WithContext(r.Context())
		urlPath := filepath.ToSlash(fullpath[prefixLen:])

		newRequest.URL.Path = urlPath

		http.FileServer(http.Dir(homeDir)).ServeHTTP(w, newRequest)
	}))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		booksDir := input
		query := r.URL.Query().Get("path")
		if query != "" {
			booksDir = query
		}

		// ensure they don't go past root directory
		if !strings.HasPrefix(booksDir, homeDir) {
			booksDir = homeDir
		}

		files, err := displayFilesFromDir(booksDir)
		if err != nil {
			log.Println(err)
			return
		}

		sortBy := r.URL.Query().Get("sort_by")
		if sortBy != "" {
			SortBy = sortBy
		}
		if SortBy == "alphabetical" {
			ByName(files)
		} else {
			ByModifiedDate(files)
		}

		err = tmpl.Execute(w, struct {
			Root        string
			CurrentPath string
			Files       []File
			Addr        string
			Sorted      string
		}{Files: files, Addr: localIP, CurrentPath: booksDir, Root: homeDir, Sorted: SortBy})
		if err != nil {
			panic(err)
		}
	})

	addr := fmt.Sprintf("0.0.0.0:%s", port)

	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	go func() {
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	asciiWelcome := `
__        __   _                                    
\ \      / /__| | ___ ___  _ __ ___   ___    
 \ \ /\ / / _ \ |/ __/ _ \| '_ ' _ \ / _ \    
  \ V  V /  __/ | (_| (_) | | | | | |  __/
   \_/\_/ \___|_|\___\___/|_| |_| |_|\___|

`
	fmt.Println(asciiWelcome)
	fmt.Println("server started on ", srv.Addr)
	fmt.Printf("Enter http://%s in your browser\n", localIP)

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

	<-stopSig
	fmt.Println("\rstop signal received, terminating...")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}

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
	return "", fmt.Errorf("ip not found. Please connect to a network")
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

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)

	switch {
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d Bytes", size)
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
			ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
			retrievedSize := getFolderSize(fullEntryPath, ctx)
			if retrievedSize != 0 {
				size = retrievedSize
			}
		}

		file := File{
			Path:         fullEntryPath,
			Name:         name,
			ModifiedDate: info.ModTime(),
			Size:         formatFileSize(size),
			IsDir:        info.IsDir(),
		}
		files = append(files, file)
	}

	return files, nil
}

func getDirectory(dir string) string {
	return filepath.Dir(dir)
}
