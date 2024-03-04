package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
	"bytes"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"fmt"
)

type homePage struct {
	app.Compo
}

type SignUpPage struct {
	app.Compo
}

type SignUpUser struct {
	UserName string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}


type Image struct {
	app.Compo
	Source string
	Alt string
}

type Offer struct {
	app.Compo
	Title       string
	Description string
	Photos      []Image
	PostedAt    time.Time
	UserID      uint
	CommunityID uint
}
const apiUrl = "http://127.0.0.1:8000"

func (s *SignUpPage) OnPost(ctx app.Context, e app.Event) {
	//make a request to the API      
	log.Println("Signing up user")
	var user SignUpUser
	log.Println(ctx.Dispatcher().Context().Value("username"))
	user.UserName = ctx.Dispatcher().Context().JSSrc().String()
	user.Email = ctx.Dispatcher().Context().JSSrc().String()
	user.Password = ctx.Dispatcher().Context().JSSrc().String()
	fmt.Printf("User: %v\n", user.Email)
	userJson, err := json.Marshal(user)
	if err != nil {
		log.Fatal("Error marshalling user", err)
	}
	resp, err := http.Post(apiUrl + "/signup", "multipart/form-data", bytes.NewBuffer(userJson))
	if err != nil {
		log.Fatal("Error signing up user", err)
	}
	defer resp.Body.Close()
}

func (s *SignUpPage) Render() app.UI {
	//signup form stores the user's username, email, and password in ctx
	return app.Div().Body(
		app.H1().Text("Sign Up"),
		app.Form().Body(
			app.Input().Type("text").Placeholder("Username").Name("username"),
			app.Input().Type("email").Placeholder("Email").Name("email"),
			app.Input().Type("password").Placeholder("Password").Name("password"),
			app.Button().Text("Sign Up").OnClick(s.OnPost),

		),
	)
	
}

func (o *Offer) Render() app.UI {
	//make a request to the API to get Offer
	return app.Div().Body(
		app.H3().Text(o.Title),
		app.P().Text(o.Description),
		app.P().Text("Posted by " + string(o.UserID) + " at " + o.PostedAt.Format(time.RFC3339)),
	)
}

func (h *homePage) Render() app.UI {
	offers := getOffers(1)
	log.Println(offers)
	offerCompos := make([]app.UI, len(offers))
	for i, offer := range offers {
		offerCompos[i] = &offer
	}
	return app.Div().Body(
		app.H1().Text("Offers"),
		app.Div().Body(offerCompos...),
	)
}

	

func getOffers(communityId uint) []Offer {
	//make a request to the API to get Offers
	communityIdStr := strconv.FormatUint(uint64(communityId), 10)
	resp, err := http.Get(apiUrl + "/offers/" + communityIdStr)
	if err != nil {
		log.Fatal("Error getting offers", err)
	}
	defer resp.Body.Close()
	log.Println(resp.Body)
	//parse the response
	var offers []Offer
	if err := json.NewDecoder(resp.Body).Decode(&offers); err != nil {
		log.Fatal("Error decoding offers", err)
	}
	return offers
}

func main() {
	app.Route("/", &homePage{})
	app.Route("/signup", &SignUpPage{})
	app.RunWhenOnBrowser()
	http.Handle("/", &app.Handler{
		Name:        "homePage",
		Description: "An Hello World! example",
	})
	if err := http.ListenAndServe(":7777", nil); err != nil {
		log.Fatal(err)
	}
}
