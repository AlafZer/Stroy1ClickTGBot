package dbController

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

var dummyToken = "Bd{cD#h%;6oXi$nY2!_hvjcRK:5Yw55^M5UFV*L#;:KGICsJYjUy[5k,wk|*3JpV"

func (dbCtrl *DatabaseController) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRow := r.Header.Get("Authorization")
		tokenArr := strings.Split(tokenRow, " ")
		if len(tokenArr) != 2 || tokenArr[0] != "Bearer" {
			_ = dbCtrl.errorJSON(w, errors.New("wrong authentication credentials"))
			return
		}

		token := tokenArr[1]

		if dbCtrl.validToken(token) == 0 {
			http.Error(w, "Wrong access token, GET OUT DUMBASS!!!", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (dbCtrl *DatabaseController) validToken(token string) int {
	if len(dbCtrl.token) != len(token) {
		return subtle.ConstantTimeCompare([]byte(dummyToken), []byte(token))
	}

	return subtle.ConstantTimeCompare([]byte(dbCtrl.token), []byte(token))
}
