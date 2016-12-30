package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/tomsteele/turtle"
	"github.com/tomsteele/turtle/schemes"
	"goji.io"
	"goji.io/pat"
)

type EW struct{}

func (e EW) Unauthorized(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Fprintf(w, "unauthorized")
}

func (e EW) ServerError(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Fprintf(w, "serverError")
}

func (e EW) Forbidden(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Fprintf(w, "forbidden")
}

func (e EW) BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Fprintf(w, "bad request")
}

type User struct {
	Username string
	Roles    []string
}

// Implements Roler.
func (u User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

var user = User{
	Username: "alice",
	Roles:    []string{"user"},
}

func main() {

	bundle := turtle.NewBundler(EW{})
	bundle.RegisterScheme("jwt", &schemes.JWTScheme{
		Secret: []byte("password"),
		ValidateFunc: func(claims jwt.MapClaims) (interface{}, error) {
			username, ok := claims["username"].(string)
			if !ok {
				return nil, errors.New("no username in token")
			}
			if username != user.Username {
				return nil, errors.New("user not found")
			}
			return user, nil
		},
	})
	bundle.SetDefaultScheme("jwt") // Every request will require jwt scheme, unless AuthMode none.
	mux := goji.NewMux()

	mux.HandleFunc(pat.Post("/token"), bundle.New(turtle.O{
		Allow:    []string{"application/json"}, // Only allow JSON.After
		AuthMode: "none",                       // Disable authentication for this route.
		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			// Here is where you would probably decode username and password and validate.
			token := jwt.New(jwt.SigningMethodHS512)
			claims := jwt.MapClaims{}
			claims["username"] = user.Username
			token.Claims = claims
			s, _ := token.SignedString([]byte("password"))
			fmt.Fprintf(w, "token: %s", s)
		},
	}))

	mux.HandleFunc(pat.Get("/me"), bundle.New(turtle.O{
		AuthMode: "required",
		Roles:    []string{"user"}, // Roles can be used to restrict access.
		Schemes:  []string{"jwt"},  // Schemes can be set per HandleFunc.
		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			// Authentication schemes mount credentials in the request context.
			u := r.Context().Value(turtle.CtxCredentials{}).(User)
			fmt.Fprintf(w, "username: %s", u.Username)
		},
	}))

	log.Fatal(http.ListenAndServe(":3000", mux))
}
