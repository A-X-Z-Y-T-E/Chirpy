package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/A-X-Z-Y-T-E/Chirpy/internal/auth"
	"github.com/A-X-Z-Y-T-E/Chirpy/internal/database"
)

type User struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	Email         string `json:"email"`
	Is_Chirpy_Red bool   `json:"is_chirpy_red"`
}
type authUser struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	Email         string `json:"email"`
	Token         string `json:"token"`
	RefreshToken  string `json:"refresh_token"`
	Is_Chirpy_Red bool   `json:"is_chirpy_red"`
}

func (cfg *apiConfig) Reset() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits = atomic.Int32{}
		if cfg.platform != "dev" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		err := cfg.db.DeleteAllUsers(r.Context())
		if err != nil {
			fmt.Println(err)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (cfg *apiConfig) create_user() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}
		var params parameters
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		hashed_password, err := auth.HashPassword(params.Password)
		user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
			Email:          params.Email,
			HashedPassword: hashed_password,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		data, err := json.Marshal(User{
			ID:            user.ID.String(),
			CreatedAt:     user.CreatedAt.String(),
			UpdatedAt:     user.UpdatedAt.String(),
			Email:         user.Email,
			Is_Chirpy_Red: user.IsChirpyRed,
		})
		w.Write(data)
	})
}

func (cfg *apiConfig) login() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}
		var params parameters
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		user, err := cfg.db.GetUser(r.Context(), params.Email)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ok, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
		if err != nil || !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		accessToken, _ := auth.MakeJWT(user.ID, cfg.secret)

		refresh_token, _ := auth.MakeRefreshToken()

		cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			Token:     refresh_token,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		data, err := json.Marshal(authUser{
			ID:            user.ID.String(),
			CreatedAt:     user.CreatedAt.String(),
			UpdatedAt:     user.UpdatedAt.String(),
			Email:         user.Email,
			Token:         accessToken,
			RefreshToken:  refresh_token,
			Is_Chirpy_Red: user.IsChirpyRed,
		})
		w.Write(data)
	})
}

func (cfg *apiConfig) refresh() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			fmt.Println("GetBearerToken error:", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		fmt.Println("Refresh token:", refreshToken)

		dbToken, err := cfg.db.GetRefreshToken(r.Context(), refreshToken)
		if err != nil {
			fmt.Println("GetRefreshToken DB error:", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		fmt.Println("Found token for user:", dbToken.UserID)

		accessToken, err := auth.MakeJWT(dbToken.UserID, cfg.secret)
		if err != nil {
			fmt.Println("MakeJWT error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			Token string `json:"token"`
		}{
			Token: accessToken,
		})
	})
}

func (cfg *apiConfig) revoke() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func (cfg *apiConfig) UpdateCredentials() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		UserID, err := auth.ValidateJWT(accessToken, cfg.secret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		UserDetails, _ := cfg.db.GetUserFromId(r.Context(), UserID)

		type Credentials struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		var creds Credentials
		err = json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		hashed_password, _ := auth.HashPassword(creds.Password)
		err = cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
			Email:          creds.Email,
			HashedPassword: hashed_password,
			ID:             UserID,
		})

		json.NewEncoder(w).Encode(User{
			ID:            UserID.String(),
			CreatedAt:     UserDetails.CreatedAt.String(),
			UpdatedAt:     time.Now().String(),
			Email:         creds.Email,
			Is_Chirpy_Red: UserDetails.IsChirpyRed,
		})
	})
}
func (cfg *apiConfig) DeleteUser() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("=== DeleteUser called ===")

		// Since route is /api/chirps/{chirpID}, use "chirpID"
		ChirpIDStr := r.PathValue("chirpID")
		ChirpID, _ := convert_to_uuid(ChirpIDStr)
		fmt.Println("DeleteID from path:", ChirpID)

		accessToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			fmt.Println("GetBearerToken error:", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Println("Access token:", accessToken)

		UserID, err := auth.ValidateJWT(accessToken, cfg.secret)
		if err != nil {
			fmt.Println("ValidateJWT error:", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Println("Authenticated UserID:", UserID.String())
		Chirp, err := cfg.db.GetChirpByID(r.Context(), ChirpID)

		if Chirp.UserID != UserID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Println("Authorization passed")

		err = cfg.db.DeleteChirp(r.Context(), ChirpID)
		if err != nil {
			fmt.Println("DeleteUser DB error:", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		fmt.Println("User deleted successfully")
		w.WriteHeader(http.StatusNoContent)
	})
}

func (cfg *apiConfig) Upgrade_User() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// accessToken, err := auth.GetBearerToken(r.Header)
		// if err != nil {
		// 	w.WriteHeader(http.StatusUnauthorized)
		// 	return
		// }
		// UserID, err := auth.ValidateJWT(accessToken, cfg.secret)
		// if err != nil {
		// 	w.WriteHeader(http.StatusUnauthorized)
		// 	return
		// }
		received_API_Key, err := auth.GetAPIKEY(r.Header)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if received_API_Key != cfg.Polka_key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		type UpgradeRequest struct {
			Event string `json:"event"`
			Data  struct {
				UserID string `json:"user_id"`
			} `json:"data"`
		}
		var upgradeRequest UpgradeRequest
		err = json.NewDecoder(r.Body).Decode(&upgradeRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if upgradeRequest.Event != "user.upgraded" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		user_id, _ := convert_to_uuid(upgradeRequest.Data.UserID)
		err = cfg.db.UpgradeUserToChirpyRed(r.Context(), user_id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
