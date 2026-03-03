package main

import (
	"log"
	"net/http"
	"net/http/cgi"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: git-server <port> <repos_dir>")
	}
	port := os.Args[1]
	reposDir := os.Args[2]

	backend := "/usr/lib/git-core/git-http-backend"
	if _, err := os.Stat(backend); err != nil {
		log.Fatalf("git-http-backend not found at %s", backend)
	}

	handler := &cgi.Handler{
		Path: backend,
		Env: []string{
			"GIT_PROJECT_ROOT=" + reposDir,
			"GIT_HTTP_EXPORT_ALL=1",
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		handler.ServeHTTP(w, r)
	})

	log.Printf("Serving git repos in %s on :%s", reposDir, port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, nil))
}
