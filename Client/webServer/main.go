package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
)

var (
	apiURL = "http://127.0.0.1:8000"
)

type Photo struct {
	ID int
}

type Offer struct {
	ID            int
	Title         string
	Description   string
	User          string
	Photos        []Photo
	CommunityName string
	CreatedAt     string
	UserID        int `json:"user_id"`
}

type Community struct {
	Name    string
	Country string
	ID      int
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
		"email":    r.Form.Get("email"),
		"password": r.Form.Get("password"),
	}
	jstring := map2json(rVals)
	req, err := http.NewRequest("POST", apiURL+"/signin", bytes.NewBuffer(jstring))
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
	//delete cookies
	http.SetCookie(w, &http.Cookie{Name: "token", Value: "", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: "token_id", Value: "", MaxAge: -1})
	//render login offerPage
	err := userLoginPage().Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	}
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
	MAX_FORM_SIZE := int64(10 ^ 6*10) // 10MB
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

	//Upload image to server
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	if len(r.MultipartForm.File["image"]) > 0 {
		//check if file is an image
		imageWriter, err := writer.CreateFormFile("image", r.MultipartForm.File["image"][0].Filename)
		image, fileErr := r.MultipartForm.File["image"][0].Open()
		if err != nil || fileErr != nil {
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
		req, err = http.NewRequest("POST", apiURL+"/image", &b)
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
		imgresp.ImageID = " "
	}

	// get image id

	// create Request
	fmt.Printf("Form: %v\n", r.Body)
	// convert id from string to uint
	payload := map[string]string{
		"title":        r.MultipartForm.Value["title"][0],
		"description":  r.MultipartForm.Value["description"][0],
		"user_id":      id.Value,
		"community_id": r.MultipartForm.Value["community_id"][0],
		"user_token":   token.Value,
		"image_id":     imgresp.ImageID,
	}
	fmt.Printf("Payload: %v\n", payload)
	encodedPayload := map2json(payload)

	req, err = http.NewRequest("POST", apiURL+"/offers", bytes.NewBuffer(encodedPayload))
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
	json, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
		return []byte("{}")
	}
	return json
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
		"email":    r.Form.Get("email"),
	}
	jstring := map2json(rVals)
	req, err := http.NewRequest("POST", apiURL+"/signup", bytes.NewBuffer(jstring))
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

type communityOffer struct {
	Name   string
	Offers []Offer
}

func getMyOffers(w http.ResponseWriter, r *http.Request) []Offer {
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}

	req, err := http.NewRequest("GET", apiURL+"/myOffers/", bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer req.Body.Close()
	req.Header.Set("token", token.Value)
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
	comOff := []communityOffer{}
	err = json.NewDecoder(resp.Body).Decode(&comOff)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	var allOffers []Offer
	for _, co := range comOff {
		offers := co.Offers
		for _, o := range offers {
			o.CommunityName = co.Name
			o.CreatedAt = convertTime(o.CreatedAt)
			allOffers = append([]Offer{o}, allOffers...)
		}
	}
	return allOffers
}

func convertTime(t string) string {
	time, err := time.Parse(time.RFC3339, t)
	if err != nil {
		fmt.Println(t)
		fmt.Println(err)
		return ""
	}
	return time.Format("06/01/02 15:04")
}
func offerPagehandler(w http.ResponseWriter, r *http.Request) {
	comOff := getMyOffers(w, r)
	if comOff == nil {
		fmt.Println("Error getting offers")
		comOff = []Offer{}
		comOff = append(comOff, Offer{Title: "No offers found"})
	}

	err := offerPage(comOff).Render(r.Context(), w)
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
		"user_id":      id.Value,
		"user_token":   token.Value,
	}
	encodedPayload := map2json(payload)
	req, err := http.NewRequest("POST", apiURL+"/joinCommunity", bytes.NewBuffer(encodedPayload))
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

func handleCreateCommunity(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/createCommunity", http.StatusTemporaryRedirect)
		return
	}
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	payload := map[string]string{
		"name":    r.Form.Get("name"),
		"country": r.Form.Get("country"),
		"city":    r.Form.Get("city"),
	}
	encodedPayload := map2json(payload)
	req, err := http.NewRequest("POST", apiURL+"/createCommunity", bytes.NewBuffer(encodedPayload))
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/createCommunity", http.StatusTemporaryRedirect)
		return
	}
	defer req.Body.Close()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", token.Value)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/createCommunity", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.Redirect(w, r, "/createCommunity", http.StatusTemporaryRedirect)
		return
	}
	log.Println("Community created")
	http.Redirect(w, r, "/", http.StatusPermanentRedirect)
}

func getUserCommunities(w http.ResponseWriter, r *http.Request) []Community {
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil
	}
	req, err := http.NewRequest("GET", apiURL+"/userCommunities", bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer req.Body.Close()
	req.Header.Set("token", token.Value)
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
	communities := []Community{}
	err = json.NewDecoder(resp.Body).Decode(&communities)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return communities
}

func getCommunities(w http.ResponseWriter, r *http.Request) ([]Community, string) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return nil, ""
	}
	country := r.Form.Get("country")
	fmt.Println("Country: ", country)
	req, err := http.NewRequest("GET", apiURL+"/communities/"+country, bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}
	defer req.Body.Close()
	req.Header.Set("token", token.Value)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		return nil, ""
	}
	communities := []Community{}
	err = json.NewDecoder(resp.Body).Decode(&communities)
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}
	return communities, country
}

func generateCommunityList(w http.ResponseWriter, r *http.Request) {
	communities, _ := getCommunities(w, r)
	if communities == nil {
		communities = []Community{}
	}
	err := communityOptions(communities).Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
	}
}

func generateUserCommunityList(w http.ResponseWriter, r *http.Request) {
	communities := getUserCommunities(w, r)
	if communities == nil {
		communities = []Community{}
	}
	err := communityOptions(communities).Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
	}
}

func generateOffer(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusPermanentRedirect)
		return
	}
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	id := r.URL.Query().Get("offerID")
	req, err := http.NewRequest("GET", apiURL+"/offer/"+id, bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer req.Body.Close()
	req.Header.Set("token", token.Value)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	offer := Offer{}
	err = json.NewDecoder(resp.Body).Decode(&offer)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	offer.CreatedAt = convertTime(offer.CreatedAt)
	offer.CommunityName = r.URL.Query().Get("communityName")

	user, err := getUser(strconv.Itoa(offer.UserID), client)
	if err != nil {
		fmt.Println("Error getting user: ", err)
	}
	offer.User = user.Username
	err = viewOfferPage(offer).Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
		fmt.Println(id, offer)
		http.NotFound(w, r)
	}

}

type Message struct {
	Text    string
	isMyMsg bool
	OfferID int
}

type MessageFromServer struct {
	Text      string
	SenderID  int
	ReciverID int
	OfferID   int
}
type User struct {
	ID       int
	Username string
}

func getUser(id string, client *http.Client) (User, error) {
	req, err := http.NewRequest("GET", apiURL+"/user/"+id, bytes.NewBuffer([]byte("")))
	if err != nil {
		return User{}, err
	}
	defer req.Body.Close()
	resp, err := client.Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("error: %v", resp.Status)
	}
	var user User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func renderInboxOptions(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
	}
	offerID := r.Form.Get("offerID")
	posterID := r.Form.Get("posterID")
	posterIDInt, err := strconv.Atoi(posterID)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
	}
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	tokenID, err := r.Cookie("token_id")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	if posterID != tokenID.Value {
		err = selectChatBox([]User{{ID: posterIDInt, Username: "Poster"}}).Render(r.Context(), w)
		if err != nil {
			fmt.Println(err)
			http.NotFound(w, r)
		}
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL+"/offerResp/"+offerID, bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	defer req.Body.Close()
	req.Header.Set("token", token.Value)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.NotFound(w, r)
		return
	}
	var users []User
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	for i, u := range users {
		if strconv.Itoa(u.ID) == tokenID.Value {
			users = append(users[:i], users[i+1:]...)
			break
		}
	}
	if len(users) == 0 {
		err = selectChatBox([]User{{ID: posterIDInt, Username: "No Messages Yet"}}).Render(r.Context(), w)
		if err != nil {
			fmt.Println(err)
			http.NotFound(w, r)
			return
		}
	} else {
		err = selectChatBox(users).Render(r.Context(), w)
	}
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
	}
}

func renderMessageBox(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	client := &http.Client{}
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
	}
	offerID := r.Form.Get("offerID")
	posterID := r.Form.Get("posterID")
	otherUserID := r.Form.Get("otherUserID")

	fmt.Printf("OfferID: %v, PosterID: %v, OtherUserID: %v\n", offerID, posterID, otherUserID)

	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	req, err := http.NewRequest("GET", apiURL+"/messages", bytes.NewBuffer([]byte("")))
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	defer req.Body.Close()
	req.Header.Set("token", token.Value)
	req.Header.Set("otherUserID", otherUserID)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.NotFound(w, r)
		return
	}
	messages := []MessageFromServer{}
	err = json.NewDecoder(resp.Body).Decode(&messages)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	var relevantMessages []Message
	for _, m := range messages {
		if fmt.Sprint(m.OfferID) == offerID &&
			(fmt.Sprint(m.SenderID) == otherUserID || fmt.Sprint(m.ReciverID) == otherUserID) {
			msg := Message{Text: m.Text, OfferID: m.OfferID}
			msgSenderID := fmt.Sprint(m.SenderID)
			if msgSenderID == otherUserID {
				msg.isMyMsg = false
			} else {
				msg.isMyMsg = true
			}
			relevantMessages = append(relevantMessages, msg)
		}

	}
	err = chatBox(relevantMessages).Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)

	}
}
func handelSendMessage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	token, err := r.Cookie("token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	offerID := r.Form.Get("offerID")

	payload := map[string]string{
		"text":       r.Form.Get("message"),
		"reciver_id": r.Form.Get("otherUserID"),
		"offer_id":   offerID,
	}
	encodedPayload := map2json(payload)
	req, err := http.NewRequest("POST", apiURL+"/messages", bytes.NewBuffer(encodedPayload))
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	defer req.Body.Close()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", token.Value)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, r)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: ", resp.Status)
		http.NotFound(w, r)
		return
	}
	log.Println("Message sent")
	http.Redirect(w, r, "/chatBox?offerID="+offerID, http.StatusPermanentRedirect)
}

func main() {
	http.HandleFunc("/", offerPagehandler)
	http.Handle("/signup", templ.Handler(userSignupPage()))
	http.Handle("/login", templ.Handler(userLoginPage()))
	http.Handle("/createOffer", templ.Handler(createOfferPage()))
	http.Handle("/createCommunity", templ.Handler(createCommunityPage()))
	http.Handle("/joinCommunity", templ.Handler(joinCommunityPage()))
	http.HandleFunc("/handelSignup", handleSignup)
	http.HandleFunc("/handelLogin", handleLogin)
	http.HandleFunc("/handelLogout", handelLogout)
	http.HandleFunc("/handelCreateOffer", createOffer)
	http.HandleFunc("/handelJoinCommunity", handleJoinCommunity)
	http.HandleFunc("/handelCreateCommunity", handleCreateCommunity)
	http.HandleFunc("/communitiesList", generateCommunityList)
	http.HandleFunc("/userCommunitiesList", generateUserCommunityList)
	http.HandleFunc("/viewOffer", generateOffer)
	http.HandleFunc("/chatBox", renderMessageBox)
	http.HandleFunc("/handelSendMessage", handelSendMessage)
	http.HandleFunc("/offerInbox", renderInboxOptions)
	ip := "127.0.0.1"
	fmt.Printf("Server running on: %v:8080\n", ip)
	err := http.ListenAndServe(ip+":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}
