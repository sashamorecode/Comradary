package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"github.com/a-h/templ"
	"encoding/json"
	"mime/multipart"
	"io"
)

type Photo struct {
	ID int
}

type Offer struct {
	Title       string
	Description string
	User        string
	Photos      []Photo
}


func handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusNotFound)
		fmt.Println(err)
		return
	}
	fmt.Printf("Login: %v\n", r.Form)
	rVals := map[string]string{
		"email": r.Form.Get("email"),
		"password": r.Form.Get("password"),
	}
	jstring := map2json(rVals)
	req, err := http.NewRequest("POST", "http://localhost:8000/signin", bytes.NewBuffer(jstring))
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	defer req.Body.Close()

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	token := resp.Header.Get("token")
	id := resp.Header.Get("token_id")
	fmt.Printf("Token: %v, ID: %v", token, id)
	http.SetCookie(w, &http.Cookie{Name: "token", Value: string(token)})
	http.SetCookie(w, &http.Cookie{Name: "token_id", Value: string(id)})
	log.Println("Login successful")
	http.Redirect(w, r, "/", http.StatusPermanentRedirect)

}

func handelLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "token", Value: "", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: "token_id", Value: "", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusPermanentRedirect)
}

type ImgResponse struct {
	ImageID string `json:"imageID"`
}

func createOffer(w http.ResponseWriter, r *http.Request) {
	var resp *http.Response
	var err error
	var req *http.Request
	var imgresp ImgResponse
	client := &http.Client{}
	MAX_FORM_SIZE := int64(10^6 * 50) // 50MB
	//parse multipart Form

	err = r.ParseMultipartForm(MAX_FORM_SIZE)
	fmt.Println("Create Offer")
	if err != nil {
		fmt.Println("error parsing multipart form: ", err)
		http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
		return
	}
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	id, err := r.Cookie("token_id")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	fmt.Printf("Token: %v, ID: %v", token.Value, id)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
		return
	}
	//Upload image to server
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	imageWriter, err := writer.CreateFormFile("image", r.MultipartForm.File["image"][0].Filename)
	if err == nil {
		image, err := r.MultipartForm.File["image"][0].Open()
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
			return
		}
		_, err = io.Copy(imageWriter, image)
		image.Close()
		if err != nil { 
			fmt.Println(err)
			http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
			return
		}
		writer.Close()
		req, err = http.NewRequest("POST", "http://localhost:8000/image", &b)
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
			return
		}
		defer req.Body.Close()
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("token", token.Value)
		resp, err = client.Do(req)
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Println("Error: ", resp.Status)
			http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
			return
		}
		fmt.Println("Image uploaded")
		// get image id
		//bind response to struct
		err = json.NewDecoder(resp.Body).Decode(&imgresp)
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
			return
		}
		fmt.Printf("Image ID: %v\n", imgresp.ImageID)
	} else {
		fmt.Println(err)
	}

	// get image id 

	// create Request
	fmt.Printf("Form: %v\n", r.Body)
	// convert id from string to uint
	payload := map[string]string{
		"title": r.MultipartForm.Value["title"][0],
		"description": r.MultipartForm.Value["description"][0],
		"user_id": id.Value,
		"community_id": r.MultipartForm.Value["community_id"][0],
		"user_token": token.Value,
		"image_id": imgresp.ImageID,
	}
	fmt.Printf("Payload: %v\n", payload)
	encodedPayload := map2json(payload)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
		return
	}

	req, err = http.NewRequest("POST", "http://localhost:8000/offers", bytes.NewBuffer(encodedPayload))
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
		return
	}
	defer req.Body.Close()
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.Redirect(w, r, "/createOffer", http.StatusTemporaryRedirect)
		return
	}
	log.Println("Offer created")
	http.Redirect(w, r, "/", http.StatusPermanentRedirect)

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
		fmt.Println("Error parsing form: ", err)
		http.Redirect(w, r, "/signup", http.StatusTemporaryRedirect)
		return
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
		http.Redirect(w, r, "/signup", http.StatusTemporaryRedirect)
		return
	}
	defer req.Body.Close()

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/signup", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.Redirect(w, r, "/signup", http.StatusTemporaryRedirect)
		return
	}
	log.Println("Signup successful")
	http.Redirect(w, r, "/login", http.StatusPermanentRedirect)
}

func getOffers(w http.ResponseWriter, r *http.Request) []Offer {
	//client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8000/offers/1", bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer req.Body.Close()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		return nil
	}
	// Decode JSON
	offers := []Offer{}
	err = json.NewDecoder(resp.Body).Decode(&offers)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return offers
}

func getMyOffers(w http.ResponseWriter, r *http.Request) []Offer {
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}
	id, err := r.Cookie("token_id")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}

	req, err := http.NewRequest("GET", "http://localhost:8000/myOffers/"+id.Value, bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer req.Body.Close()
	req.Header.Set("Authorization", token.Value)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		return nil
	}
	offers := []Offer{}
	err = json.NewDecoder(resp.Body).Decode(&offers)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return offers
}
	


func offerPagehandler(w http.ResponseWriter, r *http.Request) {
	offers := getMyOffers(w, r)
	if offers == nil {
		fmt.Println("Error getting offers")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	
	err := offerPage(offers).Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func handleJoinCommunity(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/joinCommunity", http.StatusTemporaryRedirect)
		return
	}
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	id, err := r.Cookie("token_id")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	fmt.Printf("Token: %v, ID: %v", token.Value, id)
	payload := map[string]string{
		"community_id": r.Form.Get("community_id"),
		"user_id": id.Value,
		"user_token": token.Value,
	}
	encodedPayload := map2json(payload)
	req, err := http.NewRequest("POST", "http://localhost:8000/joinCommunity", bytes.NewBuffer(encodedPayload))
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/joinCommunity", http.StatusTemporaryRedirect)
		return
	}
	defer req.Body.Close()
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/joinCommunity", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.Redirect(w, r, "/joinCommunity", http.StatusTemporaryRedirect)
		return
	}
	log.Println("Community joined")
	http.Redirect(w, r, "/", http.StatusPermanentRedirect)
}
func main() {
	http.HandleFunc("/", offerPagehandler)
	http.Handle("/signup", templ.Handler(userSignupPage()))
	http.Handle("/login", templ.Handler(userLoginPage()))
	http.Handle("/createOffer", templ.Handler(createOfferPage()))
	http.Handle("/joinCommunity", templ.Handler(joinCommunityPage()))
	http.HandleFunc("/handelSignup", handleSignup)
	http.HandleFunc("/handelLogin", handleLogin)
	http.HandleFunc("/handelLogout", handelLogout)
	http.HandleFunc("/handelCreateOffer", createOffer)
	http.HandleFunc("/handelJoinCommunity", handleJoinCommunity)
	fmt.Println("Server started at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}

