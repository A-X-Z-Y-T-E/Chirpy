package main

import (
	"encoding/json"
	"net/http"

	"github.com/A-X-Z-Y-T-E/Chirpy/internal/auth"
	"github.com/A-X-Z-Y-T-E/Chirpy/internal/database"
	"github.com/google/uuid"
)

type Chirp struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Body      string `json:"body"`
	User_id   string `json:"user_id"`
}

func convert_to_uuid(user_id string) (uuid.UUID, error) {
	User_id, err := uuid.Parse(user_id)
	if err != nil {
		return uuid.UUID{}, err
	}
	return User_id, nil
}
func (cfg *apiConfig) add_chirp() http.Handler {
	return http.HandlerFunc(func(resW http.ResponseWriter, req *http.Request) {
		resW.Header().Set("Content-Type", "application/json")

		bearer_token, err := auth.GetBearerToken(req.Header)
		if err != nil {
			resW.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(resW).Encode(struct {
				Error string `json:"error"`
			}{
				Error: "Unauthorized",
			})
			return
		}
		User_id, err := auth.ValidateJWT(bearer_token, cfg.secret)
		if err != nil {
			resW.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(resW).Encode(struct {
				Error string `json:"error"`
			}{
				Error: "Unauthorized",
			})
			return
		}

		type parameters struct {
			Body string `json:"body"`
			// User_id string `json:"user_id"`
		}
		params := parameters{}
		err = json.NewDecoder(req.Body).Decode(&params)

		if err != nil {
			resW.WriteHeader(http.StatusInternalServerError)
			data, _ := json.Marshal(struct {
				Error string `json:"error"`
			}{
				Error: "Something went wrong",
			})
			resW.Write(data)
			return
		}

		if len(params.Body) > 140 {
			resW.WriteHeader(http.StatusBadRequest)
			data, _ := json.Marshal(struct {
				Error string `json:"error"`
			}{
				Error: "Chirp is too long",
			})
			resW.Write(data)
			return
		}
		// Convert string to UUID
		// User_id, err := convert_to_uuid(params.User_id)

		// if err != nil {
		// 	resW.WriteHeader(http.StatusBadRequest)
		// 	json.NewEncoder(resW).Encode(struct {
		// 		Error string `json:"error"`
		// 	}{
		// 		Error: "Invalid user ID",
		// 	})
		// }
		chirp, err := cfg.db.CreateChirp(req.Context(), database.CreateChirpParams{
			Body:   params.Body,
			UserID: User_id,
		})

		resW.WriteHeader(http.StatusCreated)
		json.NewEncoder(resW).Encode(Chirp{
			ID:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			User_id:   chirp.UserID.String(),
		})
	})
}

func (cfg *apiConfig) ReturnChirps() http.Handler {
	return http.HandlerFunc(func(resW http.ResponseWriter, req *http.Request) {
		resW.Header().Set("Content-Type", "application/json")

		authorID := req.URL.Query().Get("author_id")
		sort := req.URL.Query().Get("sort")
		var chirps []database.Chirp
		var err error
		if authorID != "" {
			userID, err := convert_to_uuid(authorID)
			if err != nil {
				resW.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(resW).Encode(struct {
					Error string `json:"error"`
				}{
					Error: "Invalid author_id",
				})
				return
			}
			chirps, err = cfg.db.GetChirpByUserID(req.Context(), userID)
			// chirps = []database.Chirp{chirp}
		} else {
			chirps, err = cfg.db.ReturnChirps(req.Context())
		}

		if err != nil {
			resW.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(resW).Encode(struct {
				Error string `json:"error"`
			}{
				Error: "Internal Server Error.",
			})
			return
		}

		if sort == "desc" {
			for i := len(chirps)/2 - 1; i >= 0; i-- {
				opp := len(chirps) - 1 - i
				chirps[i], chirps[opp] = chirps[opp], chirps[i]
			}
		}

		chirps_out := []Chirp{}
		for _, chirp := range chirps {
			chirps_out = append(chirps_out, Chirp{
				ID:        chirp.ID.String(),
				CreatedAt: chirp.CreatedAt.String(),
				UpdatedAt: chirp.UpdatedAt.String(),
				Body:      chirp.Body,
				User_id:   chirp.UserID.String(),
			})
		}
		resW.WriteHeader(http.StatusOK)
		json.NewEncoder(resW).Encode(chirps_out)
	})
}

func (cfg *apiConfig) GetChirp() http.Handler {
	return http.HandlerFunc(func(resW http.ResponseWriter, req *http.Request) {
		resW.Header().Set("Content-Type", "application/json")

		chirpIDStr := req.PathValue("chirpID")

		chirpID, err := convert_to_uuid(chirpIDStr)
		if err != nil {
			resW.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(resW).Encode(struct {
				Error string `json:"error"`
			}{
				Error: "Invalid chirp ID",
			})
			return
		}
		chirp, err := cfg.db.GetChirpByID(req.Context(), chirpID)
		if err != nil {
			resW.WriteHeader(http.StatusNotFound)
			json.NewEncoder(resW).Encode(struct {
				Error string `json:"error"`
			}{
				Error: "Chirp not found",
			})
			return
		}
		resW.WriteHeader(http.StatusOK)
		json.NewEncoder(resW).Encode(Chirp{
			ID:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			User_id:   chirp.UserID.String(),
		})
	})
}
