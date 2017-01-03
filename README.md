# turtle
[![](https://godoc.org/github.com/stacktitan/turtle?status.svg)](http://godoc.org/github.com/stacktitan/turtle)

HandlerFunc all the way down.


### Example
The following uses gorilla router, but anything taking a `HandlerFunc` will work with `turtle.Bundle`.
```go
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/stacktitan/turtle"
	"github.com/stacktitan/turtle/schemes"
)

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

	bundle := turtle.NewBundler()
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
	router := mux.NewRouter()
	router.HandleFunc("/token", bundle.New(turtle.O{
		Allow:    []string{"application/json"}, // Only allow JSON.
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
	})).Methods("POST")
	router.HandleFunc("/me", bundle.New(turtle.O{
		AuthMode: "required",
		Roles:    []string{"user"}, // Roles can be used to restrict access.
		Schemes:  []string{"jwt"},  // Schemes can be set per HandleFunc.
		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			// Authentication schemes mount credentials in the request context.
			u := r.Context().Value(turtle.CtxCredentials{}).(User)
			fmt.Fprintf(w, "username: %s", u.Username)
		},
	})).Methods("GET")

	log.Fatal(http.ListenAndServe(":3000", router))
}
```

`/token` requires JSON.
```
$ curl http://localhost:3000/token -X POST -H "Content-Type: application/json"
token: eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFsaWNlIn0.DIUMnDYOs1tti1aAEHXBdmdzqqrWYWGYSVWy4Q63RxeiCSLAXaJPXHWDQ-fi8tsuv1TdhIar3J14PtG5b8TKOw
```
All errors are written by default using JSON and `boom.Error` using the `DefaultErrorWriter`.

Bad content-type:
```
$ curl -ki http://localhost:3000/token -X POST -H "Content-Type: text/plain"
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=UTF-8
Date: Sat, 31 Dec 2016 06:01:31 GMT
Content-Length: 104

{"status_code":400,"error":"Bad Request","message":"invalid request content-type: text/plain","data":{}}
```
Authentication on all handlers will be required using the default scheme. And all routes require an `AuthMode`.
```
$ curl http://localhost:3000/me -H 'Authorization: bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFsaWNlIn0.DIUMnDYOs1tti1aAEHXBdmdzqqrWYWGYSVWy4Q63RxeiCS
LAXaJPXHWDQ-fi8tsuv1TdhIar3J14PtG5b8TKOw'
username: alice
```
If authentication or validation fails:
```
$ curl -ki http://localhost:3000/me
HTTP/1.1 401 Unauthorized
Content-Type: application/json; charset=UTF-8
Date: Sat, 31 Dec 2016 06:02:08 GMT
Content-Length: 65

{"status_code":401,"error":"Unauthorized","message":"","data":{}}
```
