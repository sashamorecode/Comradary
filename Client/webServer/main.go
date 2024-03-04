package main

import (
	"fmt"
	"net/http"
	"github.com/a-h/templ"
	"bytes"
)

type Offer struct {
	Title       string
	Description string
	User        string
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err)
		return
	}
	fmt.Printf("Login: %v\n", r.Form)
	rVals := map[string]string{
		"username": r.Form.Get("username"),
		"password": r.Form.Get("password"),
	}
	jstring := map2json(rVals)
	req, err := http.NewRequest("POST", "http://localhost:8000/login", bytes.NewBuffer(jstring))
	if err != nil {
		fmt.Println(err)
	}
	defer req.Body.Close()

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.Redirect(w, r, "/login", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}


func map2json(m map[string]string) []byte {
	
	json := "{"
	for k, v := range m {
		json += fmt.Sprintf("\"%v\": \"%v\",", k, v)
	}
	json = json[:len(json)-1]
	json += "}"
	return []byte(json)
}


func handleSignup(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil { 
		fmt.Println(err)
	}
	fmt.Printf("Signup: %v\n", r.Form)
	rVals := map[string]string{
		"username": r.Form.Get("username"),
		"password": r.Form.Get("password"),
		"email": r.Form.Get("email"),
	}
	jstring := map2json(rVals)
	req, err := http.NewRequest("POST", "http://localhost:8000/signup", bytes.NewBuffer(jstring))
	if err != nil {
		fmt.Println(err)
	}
	defer req.Body.Close()

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.Redirect(w, r, "/signup", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

	
func main() {
	offers := []Offer{
		{Title: "Offer 1", Description: "Description 1", User: "User 1"},
		{Title: "Offer 2", Description: "Description 2", User: "User 2"},
		{Title: "Offer 3", Description: "Description 3", User: "User 3"},
	}
	http.Handle("/", templ.Handler(offerPage(offers)))
	http.Handle("/signup", templ.Handler(userSignupPage()))
	http.Handle("/login", templ.Handler(userLoginPage()))
	http.HandleFunc("/handelSignup", handleSignup)
	http.HandleFunc("/handelLogin", handleLogin)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}

