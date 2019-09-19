package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (app *Application) setupRoutes() {
	app.Routes = http.NewServeMux()

	app.Routes.Handle("/api/newapikey/",
		addHeaders(http.HandlerFunc(app.newAPIKey)))

	app.Routes.Handle("/api/", addHeaders(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")[1:]

			switch {
			case len(parts) == 1 && req.Method == http.MethodGet:
				app.listAllForKey(parts[0])(w, req)

			case len(parts) == 2 && req.Method == http.MethodGet:
				app.getValue(parts[0], parts[1])(w, req)

			case len(parts) == 2 && req.Method == http.MethodPut:
				app.putValue(parts[0], parts[1])(w, req)

			default:
				http.NotFound(w, req)
			}
		})))
}

func addHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET, POST, PUT")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Content-Type", "application/json")
		h.ServeHTTP(w, req)
	})
}

type dict map[string]interface{}

func (app *Application) newAPIKey(w http.ResponseWriter, req *http.Request) {

	var user User
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := dec.Decode(&user)
	if err != nil {
		app.Log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	user.Key = generateKey()
	user.Created = time.Now().UTC()
	user.Modified = time.Now().UTC()

	_, err = app.DB.Exec(insertUser,
		user.Created,
		user.Modified,
		user.Name,
		user.Email,
		user.Key,
		user.Note)
	if err != nil {
		app.Log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(dict{
		"name":    user.Name,
		"email":   user.Email,
		"note":    user.Note,
		"apikey":  user.Key,
		"created": user.Created,
	})
	if err != nil {
		app.Log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *Application) listAllForKey(apikey string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var user User
		err := app.DB.Get(&user, getUser, apikey)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		pairs := []Pair{}
		err = app.DB.Select(&pairs, selectPairs, user.ID)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		err = enc.Encode(pairs)
		if err != nil {
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func (app *Application) getValue(apikey, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var user User
		err := app.DB.Get(&user, getUser, apikey)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		var pair Pair
		err = app.DB.Get(&pair, getPair, user.ID, key)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		err = enc.Encode(dict{
			"key":      pair.Key,
			"value":    pair.Value,
			"apikey":   user.Key,
			"created":  pair.Created,
			"modified": pair.Modified,
		})
		if err != nil {
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func (app *Application) putValue(apikey, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var user User
		err := app.DB.Get(&user, getUser, apikey)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		var pair Pair
		update := true
		err = app.DB.Get(&pair, getPair, user.ID, key)
		switch {
		case err == sql.ErrNoRows:
			// add new value
			update = false
			pair.Created = time.Now()
			pair.OwnerID = user.ID
			pair.Key = key
		case err != nil:
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		body := dict{}
		dec := json.NewDecoder(req.Body)
		defer req.Body.Close()
		err = dec.Decode(&body)
		if err != nil {
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		value, ok := body["value"].(string)
		if !ok || len(value) > maxValueSize {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("value is not a string or is longer than 4096 bytes"))
			return
		}
		pair.Value = value
		pair.Modified = time.Now().UTC()

		// var res sql.Result
		switch update {
		case true:
			// UPDATE
			_, err = app.DB.Exec(updatePair, pair.Value, pair.Modified, pair.ID)

		case false:
			// INSERT
			_, err = app.DB.Exec(insertPair,
				pair.Created,
				pair.Modified,
				pair.OwnerID,
				pair.Key,
				pair.Value)

		}
		if err != nil {
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		err = enc.Encode(dict{
			"key":      pair.Key,
			"value":    pair.Value,
			"apikey":   user.Key,
			"created":  pair.Created,
			"modified": pair.Modified,
		})
		if err != nil {
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}

	}
}

// max length (bytes) allowed for "value"
const maxValueSize = 4096

// queries
const (
	getUser    = "SELECT * FROM apikey WHERE key=?"
	insertUser = "INSERT INTO apikey (created, modified, name, email, key, note) VALUES (?, ?, ?, ?, ?, ?)"

	selectPairs = "SELECT * FROM kvpair WHERE owner_id=?"
	getPair     = "SELECT * FROM kvpair WHERE owner_id=? AND key=?"
	updatePair  = "UPDATE kvpair SET value=?, modified=? WHERE id=?"
	insertPair  = "INSERT INTO kvpair (created, modified, owner_id, key, value) VALUES (?,?,?,?,?)"
)
