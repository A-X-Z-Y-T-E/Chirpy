package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/A-X-Z-Y-T-E/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
	secret         string
	Polka_key      string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resW http.ResponseWriter, req *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(resW, req)
	})
}
func (cfg *apiConfig) printMetrics() http.Handler {
	return http.HandlerFunc(func(resW http.ResponseWriter, req *http.Request) {
		hits := cfg.fileServerHits.Load()
		resW.Write([]byte(fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>`, hits)))
		resW.Header().Set("Content-Type", "text/html")
	})
}

func main() {
	err := godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if err != nil {
		fmt.Println("Warning no .env file found")
	}
	db, err := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)

	if err != nil {
		fmt.Println("Couldnt connet to DB")
	}
	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}
	apiCfg := apiConfig{
		fileServerHits: atomic.Int32{},
		db:             dbQueries,
		platform:       os.Getenv("PLATFORM"),
		secret:         os.Getenv("tokenSecret"),
		Polka_key:      os.Getenv("POLKA_KEY"),
	}

	file_server_handler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	assets_file_handler := http.StripPrefix("/app/assets", http.FileServer(http.Dir("./assets")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(file_server_handler))
	mux.Handle("/app/assets", assets_file_handler)

	mux.Handle("GET /admin/metrics", apiCfg.printMetrics())
	mux.Handle("POST /admin/reset", apiCfg.Reset())

	//API
	mux.Handle("GET /api/chirps", apiCfg.ReturnChirps())
	mux.Handle("GET /api/chirps/{chirpID}", apiCfg.GetChirp())
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.Handle("POST /api/polka/webhooks", apiCfg.Upgrade_User())
	mux.Handle("POST /api/chirps", apiCfg.add_chirp())
	mux.Handle("POST /api/users", apiCfg.create_user())
	mux.Handle("POST /api/login", apiCfg.login())
	mux.Handle("POST /api/refresh", apiCfg.refresh())
	mux.Handle("POST /api/revoke", apiCfg.revoke())

	mux.Handle("PUT /api/users", apiCfg.UpdateCredentials())

	mux.Handle("DELETE /api/chirps/{chirpID}", apiCfg.DeleteUser())

	err = server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		return
	}

}
