package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Configures the serve mux for the application.
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

			case req.Method == http.MethodOptions:
				return // status 200 with cors headers

			default:
				http.NotFound(w, req)
			}
		})))

	app.Routes.Handle("/", addHeaders(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>see <a href="https://github.com/quillaja/kvss">https://github.com/quillaja/kvss</a></body></html>`))
		})))
}

// Applies standard headers to all responses.
func addHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET, POST, PUT")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Content-Type", "application/json")
		h.ServeHTTP(w, req)
	})
}

// A type for simpler JSONifing of arbitary data.
type dict map[string]interface{}

// Handler to create a new API key.
func (app *Application) newAPIKey(w http.ResponseWriter, req *http.Request) {

	// get user name, email, and note from req body
	var user User
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := dec.Decode(&user)
	if err != nil {
		app.Log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	// generate key and set time fields
	user.Key = generateKey()
	user.Created = time.Now().UTC()
	user.Modified = time.Now().UTC()

	// insert
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

	// return response
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	err = enc.Encode(user)
	if err != nil {
		app.Log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// returns a JSON array of key-value pairs associated with the given apikey.
func (app *Application) listAllForKey(apikey string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// get user
		var user User
		err := app.DB.Get(&user, getUser, apikey)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		// get kv pair data
		pairs := []Pair{}
		err = app.DB.Select(&pairs, selectPairs, user.ID)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		// write response
		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		err = enc.Encode(pairs)
		if err != nil {
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// returns a single key-value pair for the given apikey and key.
func (app *Application) getValue(apikey, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// get user
		var user User
		err := app.DB.Get(&user, getUser, apikey)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		// get pair
		var pair Pair
		err = app.DB.Get(&pair, getPair, user.ID, key)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		// write response
		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		err = enc.Encode(dict{
			"key":      pair.Key,
			"value":    pair.Value,
			"apikey":   user.Key, // could probably get rid of this
			"created":  pair.Created,
			"modified": pair.Modified,
		})
		if err != nil {
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// will update or create the key-value pair for the given apikey and key.
func (app *Application) putValue(apikey, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// get user
		var user User
		err := app.DB.Get(&user, getUser, apikey)
		if err != nil {
			app.Log.Println(err)
			http.NotFound(w, req)
			return
		}

		// get pair
		// set a flag to "create" if the db says the key isn't found
		var pair Pair
		update := true
		err = app.DB.Get(&pair, getPair, user.ID, key)
		switch {
		case err == sql.ErrNoRows:
			// add new value
			update = false
			pair.Created = time.Now().UTC()
			pair.OwnerID = user.ID
			pair.Key = key
		case err != nil:
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// attempt to read the value from the request body
		body := dict{}
		dec := json.NewDecoder(req.Body)
		defer req.Body.Close()
		err = dec.Decode(&body)
		if err != nil {
			app.Log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// value must be a string and less than a certain size to be accepted
		// by the API.
		value, ok := body["value"].(string)
		if !ok || len(value) > maxValueSize {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("value is not a string or is longer than 4096 bytes"))
			return
		}
		pair.Value = value
		pair.Modified = time.Now().UTC()

		// do the update or create
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

		// write the response
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
