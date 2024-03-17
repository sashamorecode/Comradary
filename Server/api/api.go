package api

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"time"
	"image/png"
	"image/jpeg"
	"os"
	"github.com/nfnt/resize"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// creaet random key
var jwtKey = []byte("1asf12vsr2agrasg892yh780gahe780g0sbh8")

type User struct {
	gorm.Model
	UserName         string
	Email            string `gorm:"unique"`
	PasswordHash     string
	Offers           []Offer     `gorm:"foreignKey:UserID"`
	Requests         []Request   `gorm:"foreignKey:UserID"`
	ProfilePhoto     *Photo      `gorm:"foreignKey:UserID"`
	Communities      []Community `gorm:"many2many:user_communities;"`
	OwnedCommunities []Community `gorm:"foreignKey:OwnerID"`
	MessagesInbox    []Message   `gorm:"foreignKey:ReciverID"`
	MessagesOutbox   []Message   `gorm:"foreignKey:SenderID"`
}

type Photo struct {
	gorm.Model
	Path      string `gorm:"unique" json:"path"`
	OfferID   *uint
	RequestID *uint
	UserID    uint `json:"user_id"`
}

type Offer struct {
	gorm.Model
	Title       	 string    `json:"title"`
	Description 	 string    `json:"description"`
	Photos      	 []Photo   `gorm:"foreignKey:OfferID"`
	UserID      	 uint      `json:"user_id"`
	CommunityID      uint      `json:"community_id"`
	Messages         []Message `gorm:"foreignKey:OfferID"`
	CreatedAt        time.Time
	MessagesInbox    []Message   `gorm:"foreignKey:OfferID"`
}

type Request struct {
	gorm.Model
	Title       string
	Description string
	Photos      []Photo `gorm:"foreignKey:RequestID"`
	UserID      uint
	CommunityID uint
	Messages    []Message `gorm:"foreignKey:RequestID"`
	CreatedAt   time.Time
}

type Community struct {
	gorm.Model
	Name     string    `gorm:"unique"`
	Country  string  
	City     string
	OwnerID  *uint
	Users    []User    `gorm:"many2many:user_communities;"`
	Offers   []Offer   `gorm:"foreignKey:CommunityID"`
	Requests []Request `gorm:"foreignKey:CommunityID"`
}

type Message struct {
	gorm.Model 
	Text       string
	SenderID   uint 
	ReciverID  uint
	OfferID    *uint
	RequestID  *uint 
}
type SignUpInput struct {
	UserName string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SignInInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type OfferInput struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	UserID      string `json:"user_id" binding:"required"`
	CommunityID string `json:"community_id" binding:"required"`
	Token       string `json:"user_token" binding:"required"`
	ImageID     string `json:"image_id" binding:"required"`
}


func SetupRoutes(db *gorm.DB, router *gin.Engine) {
	router.POST("/image", CreateImage(db))
	router.POST("/signup", SignUp(db))
	router.POST("/signin", SignIn(db))
	router.POST("/joinCommunity", JoinCommunity(db))
	router.POST("/createCommunity", createCommunity(db))
	router.POST("/offers", CreateOffer(db))
	router.GET("/offers/:id", GetOffersByCommunityId(db))
	router.GET("/myOffers", GetOffersByUserId(db))
	router.GET("/images/:id", GetImageById(db))
	router.GET("/communities/:country", GetCommunityByCountry(db))
	router.GET("/userCommunities", GetUserCommunities(db))
	router.GET("/offer/:id", GetOfferById(db))

	router.POST("/messages", SendMesssage(db))
	router.GET("/messages", GetMessages(db))
	router.GET("/username/:id", GetUserById(db))
	router.GET("/offerResp/:id", GetOfferResp(db))
}


func InsertTestData(db *gorm.DB) {
	// Create a Community
	community := Community{Name: "test", Country: "test", City: "testcity"}
	result := db.Create(&community)
	if result.Error != nil {
		log.Fatal("Error creating community: ", result.Error)
	}
	// Create a User
	user := User{UserName: "test", Email: "sas@gmail.com", PasswordHash: "test"}
	result = db.Create(&user)
	if result.Error != nil {
		log.Fatal("Error creating user: ", result.Error)
	}
	// Create an offer
	offer := Offer{Title: "test", Description: "test", UserID: user.ID, CommunityID: community.ID}
	result = db.Create(&offer)
	if result.Error != nil {
		log.Fatal("Error creating offer: ", result.Error)
	}
	// Create a Request
	request := Request{Title: "test", Description: "test", UserID: user.ID, CommunityID: community.ID}
	result = db.Create(&request)
	if result.Error != nil {
		log.Fatal("Error creating request: ", result.Error)
	}
}
func ConnectDB() *gorm.DB {
	dsn := "root:password@tcp(127.0.0.1:3306)/mysql?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	//DropAllTables(db)
	err = db.AutoMigrate(&User{}, &Photo{}, &Offer{}, &Request{}, &Community{}, &Message{})
	if err != nil {
		log.Fatal("Error Migrating the database: ", err)
	}
	//log.Println("Inserting test data")
	//InsertTestData(db)
	return db
}

func CreateImage(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		err = c.Request.ParseMultipartForm(10^6 *5) // 5MB
		if err != nil {
			c.JSON(399, gin.H{"error": err.Error()})
			return
		}
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}
		if form.File["image"] == nil {
			c.JSON(402, gin.H{"error": "no image file"})
			return
		}
		fileName := form.File["image"][0].Filename
		fileType := fileName[len(fileName)-4:]
		if fileType != ".jpg" && fileType != ".png" && fileType != "jpeg" {
			c.JSON(403, gin.H{"error": "file type not supported"})
			return
		}
		tokenString := c.Request.Header.Get("token")
		userIDstr, err := validateJWT(tokenString)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}
		userID, err := strconv.Atoi(userIDstr)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		image := form.File["image"][0]
		src, err := image.Open()
		if err != nil {
			c.JSON(405, gin.H{"error": err.Error()})
			return
		}
		defer src.Close()
		fmt.Printf("image: %v\n", src)
		img, err := jpeg.Decode(src)
		if err != nil {
			fmt.Printf("error decoding image: %e\n", err)
			img, err = png.Decode(src)
			if err != nil {
				fmt.Printf("error decoding image: %e\n", err)
				c.JSON(406, gin.H{"error": ""})
				return
			}
		}
		img = resize.Resize(512, 0, img, resize.Lanczos3)
		if err != nil {
			c.JSON(407, gin.H{"error": err.Error()})
			return
		}

		timeString := time.Now().String()
		filename := timeString + image.Filename
		imgFile, err := os.Create(filepath.Join("./images", filename))
		if err != nil {
			c.JSON(408, gin.H{"error": err.Error()})
			return
		}
		defer imgFile.Close()
		err = jpeg.Encode(imgFile, img, nil)
		if err != nil {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.JSON(406, gin.H{"error": err.Error()})
			return
		}
		photo := Photo{Path: filename, UserID: uint(userID)}
		result := db.Create(&photo)
		if result.Error != nil {
			fmt.Printf("path: %v\n", photo.Path)
			c.JSON(406, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(200, gin.H{"imageID": strconv.Itoa(int(photo.ID))})
	}
}

func GetImageById(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var photo Photo
		id := c.Param("id")
		_, err := strconv.Atoi(id)
		if err != nil {
			c.JSON(400, gin.H{"error": "id is not a number"})
			return
		}
		result := db.First(&photo, id)
		if result.Error != nil {
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		c.File("./images/" + photo.Path)
	}
}

func HashPassword(password string) (string, error) {
	var passwordHash string
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	passwordHash = string(hash)
	return passwordHash, nil
}

func CheckPassword(password string, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func SignUp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input SignUpInput
		err := c.BindJSON(&input)
		if err != nil {
			log.Println("Error binding json: ", err)
			log.Println("json: ", input)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		log.Printf("username: %s, email: %s, password: %s", input.UserName, input.Email, input.Password)
		passwordHash, err := HashPassword(input.Password)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		user := User{UserName: input.UserName, Email: input.Email, PasswordHash: passwordHash}
		result := db.Create(&user)
		if result.Error != nil {
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(200, user)
	}
}

func generateJWT(userid string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = userid
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		err := fmt.Errorf("error signing token: %v", err)
		return "", err
	}
	fmt.Printf("token: %v\n", tokenString)
	_, err = validateJWT(tokenString)
	if err != nil {
		err := fmt.Errorf("error validating token on creation: %v", err)
		return "", err
	}
	return tokenString, nil
}

func validateJWT(tokenString string) (string, error) {
	fmt.Printf("token: %v\n", tokenString)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		fmt.Printf("token: %v\n", tokenString)
		err := fmt.Errorf("error parsing token: %v", err)
		return "", err
	}
	if !token.Valid {
		err := fmt.Errorf("token is not valid")
		return "", err
	}
	tokenClaims := token.Claims.(jwt.MapClaims)
	userID := tokenClaims["user_id"]
	fmt.Printf("user id: %v\n", userID)
	return userID.(string), nil
}

func SignIn(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input SignInInput
		err := c.BindJSON(&input)
		if err != nil {
			log.Println("Error binding json: ", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var user User
		result := db.Where("email = ?", input.Email).First(&user)
		if result.Error != nil {
			log.Println("Error finding user: ", result.Error)
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		if !CheckPassword(input.Password, user.PasswordHash) {
			log.Println("Incorrect password")
			c.JSON(400, gin.H{"error": "incorrect password"})
			return
		}

		userID := strconv.Itoa(int(user.ID))
		token, err := generateJWT(string(userID))
		if err != nil {
			log.Println("Error generating JWT: ", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.Header("token", token)
		c.Header("token_id", strconv.Itoa(int(user.ID)))
		c.JSON(200, gin.H{"user": user})

	}
}

func GetUserById(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user User
		id := c.Param("id")
		result := db.First(&user, id)
		if result.Error != nil {
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(200, user)
	}
}

type joinCommunityInput struct {
	UserID        string `json:"user_id" binding:"required"`
	CommunityID   string `json:"community_id" binding:"required"`
	UserToken     string `json:"user_token" binding:"required"`
}

func JoinCommunity(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user User
		var community Community
		var input joinCommunityInput
		err := c.BindJSON(&input)
		if err != nil {
			fmt.Printf("error binding json: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
		}
		tokenOwnID, err := validateJWT(input.UserToken)
		if err != nil {
			fmt.Printf("error validating token1: %v\n", err)
			fmt.Printf("token: %v\n", input.UserToken)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if tokenOwnID != input.UserID {
			fmt.Printf("token id: %v, offer user id: %v\n", tokenOwnID, input.UserID)
			c.JSON(400, gin.H{"error": "token id does not match offer user id"})
			return
		}
		userID := input.UserID
		communityID := input.CommunityID
		if userID == "" || communityID == "" {
			fmt.Printf("user_id: %v, community_name:%v", userID, communityID )
			c.JSON(400, gin.H{"error": "user_id or community_id is empty"})
			return
		}
		userResult := db.First(&user, userID)
		if userResult.Error != nil {
			c.JSON(400, gin.H{"error": userResult.Error.Error()})
			return
		}
		communityResult := db.First(&community, communityID)
		if communityResult.Error != nil {
			c.JSON(400, gin.H{"error": communityResult.Error.Error()})
			return
		}
		err = db.Model(&user).Association("Communities").Append(&community)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"user": user, "community": community})
	}
}

type createCommunityInput struct {
	Name    string `json:"name" binding:"required"`
	Country string `json:"country" binding:"required"`
	City    string `json:"city" binding:"required"`
}

func createCommunity(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		var owner User
		tokenString := c.Request.Header.Get("token")
		userID, err := validateJWT(tokenString)
		if err != nil {
			c.JSON(400, gin.H{"error token valdiation": err.Error()})
			return
		}
		result := db.First(&owner, userID)
		if result.Error != nil {
			c.JSON(401, gin.H{"error finding user": result.Error.Error()})
			return
		}

		var input createCommunityInput
		err = c.BindJSON(&input)
		if err != nil {
			c.JSON(402, gin.H{"error parsing json": err.Error()})
			return
		}
		community := Community{Name: input.Name, Country: input.Country, City: input.City}
		result = db.Create(&community)
		if result.Error != nil {
			c.JSON(403, gin.H{"error creating community": result.Error.Error()})
			return
		}
		err = db.Model(&owner).Association("OwnedCommunities").Append(&community)
		if err != nil {
			c.JSON(404, gin.H{"error creatings assisiation to owner": err.Error()})
			return
		}
		err = db.Model(&owner).Association("Communities").Append(&community)
		if err != nil {
			c.JSON(405, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, community)
	}
}




func GetCommunityByCountry(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var communitys []Community 
		country := c.Param("country")
		fmt.Printf("country: %v\n", country)
		if country == "" {
			result := db.Find(&communitys)
			if result.Error != nil {
				c.JSON(400, gin.H{"error": result.Error.Error()})
				return
			}
			c.JSON(200, communitys)
			return 
		}
		if country == "ALL" {
			result := db.Find(&communitys)
			if result.Error != nil {
				c.JSON(400, gin.H{"error": result.Error.Error()})
				return 
			}
			c.JSON(200, communitys)
			return 
		}
		result := db.Find(&communitys, "country = ?", country)
		if result.Error != nil {
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(200, communitys)
	}
}

func GetUserCommunities(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user User
		var uCom []Community
		tokenString := c.Request.Header.Get("token")
		userID, err := validateJWT(tokenString)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		result := db.First(&user, userID)
		if result.Error != nil {
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		err = db.Model(&user).Association("Communities").Find(&uCom)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, uCom)
	}
}



func userBelongsToCommunity(db *gorm.DB, userID string, communityID string) (bool, error) {
	var user User
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		return false, fmt.Errorf("error parsing user id: %v", err)
	}
	communityIDInt, err := strconv.Atoi(communityID)
	if err != nil {
		return false, fmt.Errorf("error parsing community id: %v", err)
	}
	result := db.First(&user, userIDInt)
	if result.Error != nil {
		return false, result.Error
	}
	fmt.Printf("community id: %v\n", communityIDInt)
	var userCommunities []Community
	err = db.Model(&user).Association("Communities").Find(&userCommunities)
	if err != nil {
		return false, err
	}
	fmt.Printf("userCommunities: %v\n", userCommunities)
	for _, community := range userCommunities {
		if community.ID == uint(communityIDInt) {
			return true, nil
		}
	}
	return false, fmt.Errorf("user does not belong to community")
}

func CreateOffer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		log.Println("Creating offer")
		var offer OfferInput
		err = c.BindJSON(&offer)
		if err != nil {
			fmt.Printf("error binding json: %v\n", err)
			fmt.Printf("data: %v\n", offer)
			c.JSON(400, gin.H{"binding error": err.Error()})
			return
		}
		tokenOwnID, err := validateJWT(offer.Token)
		if err != nil {
			fmt.Printf("error validating token1: %v\n", err)
			fmt.Printf("token: %v\n", offer.Token)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if tokenOwnID != offer.UserID {
			fmt.Printf("token id: %v, offer user id: %v\n", tokenOwnID, offer.UserID)
			c.JSON(400, gin.H{"error": "token id does not match offer user id"})
			return
		}
		isInCommunity, err := userBelongsToCommunity(db, offer.UserID, offer.CommunityID)
		if err != nil {
			fmt.Printf("error checking if user is in community: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if !isInCommunity {
			c.JSON(400, gin.H{"error": "user does not belong to community"})
			return
		}

		var dbOffer Offer
		OfferID, err := strconv.Atoi(offer.UserID)
		if err != nil {
			fmt.Printf("error parsing user id: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		dbOffer.UserID = uint(OfferID)
		CommunityID, err := strconv.Atoi(offer.CommunityID)
		if err != nil {
			fmt.Printf("error parsing community id: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		dbOffer.CommunityID = uint(CommunityID)
		dbOffer.Title = offer.Title
		dbOffer.Description = offer.Description
		result := db.Create(&dbOffer)
		if result.Error != nil {
			log.Println("Error creating offer: ", result.Error)
			log.Println("Offer: ", offer)
			c.JSON(400, gin.H{"error": result.Error.Error(), "offer": offer.UserID})
			return
		}
		photoID, err := strconv.Atoi(offer.ImageID)
		if err != nil {
			fmt.Printf("error parsing photo id: %v\n", err)
			c.JSON(200, offer)
			return
		}
		var photo Photo
		result = db.First(&photo, uint(photoID))
		if result.Error != nil {
			fmt.Printf("error finding photo: %v\n", result.Error)
			c.JSON(200, offer)
			return
		}
		err = db.Model(&dbOffer).Association("Photos").Append(&photo)
		if err != nil {
			fmt.Printf("error associating photo with offer: %v\n", err)
			c.JSON(200, offer)
			return
		}
		c.JSON(200, offer)
	}
}

func GetOffersByCommunityId(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var offers []Offer
		param, err := strconv.Atoi(c.Param("id"))
		id := uint(param)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		results := db.Find(&offers, "community_id = ?", id)
		if results.Error != nil {
			c.JSON(400, gin.H{"error": results.Error.Error()})
			return
		}
		c.JSON(200, offers)

	}
}

func GetOffersByUserId(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user User
		var userCommunities []Community
		tokenString := c.Request.Header.Get("token")
		ret, err := validateJWT(tokenString)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		tokenID, err := strconv.Atoi(ret)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		id := uint(tokenID)
		results := db.Find(&user, id)
		if results.Error != nil {
			c.JSON(400, gin.H{"error": results.Error.Error()})
			return
		}

		err = db.Model(&user).Association("Communities").Find(&userCommunities)
		if err != nil {
			c.JSON(400, gin.H{"error": err})
			return
		}
		for i := range userCommunities {
			err = db.Model(&userCommunities[i]).Association("Offers").Find(&userCommunities[i].Offers)
			if err != nil {
				c.JSON(400, gin.H{"error": err})
				return 
			}
			for j, offer := range userCommunities[i].Offers {
				err = db.Model(&offer).Association("Photos").Find(&userCommunities[i].Offers[j].Photos)
				if err != nil {
					c.JSON(400, gin.H{"error": err})
					return
				}
			}
		}

		c.JSON(200, userCommunities)
	}
}

func isIn(user User, offer Offer) bool {
	for _, community := range user.Communities {
		if community.ID == offer.CommunityID {
			return true
		}
	}
	return false
}
func GetOfferById(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var offer Offer
		var user User
		id := c.Param("id")
		tokenString := c.Request.Header.Get("token")
		userID, err := validateJWT(tokenString)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		results := db.First(&offer, id)
		if results.Error != nil {
			c.JSON(400, gin.H{"error": results.Error.Error()})
			return
		}
		results = db.First(&user, userID)
		if results.Error != nil {
			c.JSON(400, gin.H{"error": results.Error.Error()})
			return
		}
		err = db.Model(&user).Association("Communities").Find(&user.Communities)
		if err != nil {
			c.JSON(400, gin.H{"error": err })
			return 
		}
		if !isIn(user, offer) {
			c.JSON(400, gin.H{"error": "user does not belong to community"})
			return 
		}
		err = db.Model(&offer).Association("Photos").Find(&offer.Photos)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, offer)
	}
}


type MessageInput struct {
	Text      string `json:"text" binding:"required"`
	ReciverID string `json:"reciver_id" binding:"required"`
	OfferID   string `json:"offer_id"`
}

func SendMesssage(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var message Message
		var messageInput MessageInput
		var offer Offer
		tokenString := c.Request.Header.Get("token")
		userID, err := validateJWT(tokenString) 
		
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err = c.BindJSON(&messageInput)
		if err != nil {
			fmt.Printf("error binding json: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		senderID, err := strconv.Atoi(userID)
		if err != nil {
			fmt.Printf("error parsing user id: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		reciverID, err := strconv.Atoi(messageInput.ReciverID)
		if err != nil {
			fmt.Printf("error parsing reciver id: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		offerID, err := strconv.Atoi(messageInput.OfferID)
		if err != nil {
			fmt.Printf("error parsing offer id: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		message = Message{Text: messageInput.Text, SenderID: uint(senderID), ReciverID: uint(reciverID)}
		result := db.Create(&message)
		if result.Error != nil {
			fmt.Printf("error creating message: %v\n", result.Error)
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return 
		}
		result = db.First(&offer, offerID)
		if result.Error != nil {
			c.JSON(200, message)
			return 
		}
		err = db.Model(&offer).Association("Messages").Append(&message)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return 
		}
		c.JSON(200, message)

	
	}
}

func GetMessages(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var messages []Message
		tokenString := c.Request.Header.Get("token")
		userID, err := validateJWT(tokenString)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		otherUserID := c.Request.Header.Get("otherUserID")
		if otherUserID == "" {
			fmt.Printf("otherUserID: %v\n", otherUserID)
			c.JSON(400, gin.H{"error": "otherUserID is empty"})
			return
		}
		
		result := db.Find(&messages, "sender_id = ? AND reciver_id = ?", userID, otherUserID)
		if result.Error != nil {
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return 
		}
		fmt.Printf("user id: %v, other user id: %v\n", userID, otherUserID)
		result = db.Where("reciver_id = ? AND sender_id = ?", userID, otherUserID).
			    Or("reciver_id = ? AND sender_id = ?", otherUserID, userID).
			    Find(&messages)
		if result.Error != nil {
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(200, messages)
	}
}

func ResolveUserName(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user User
		userID := c.Param("id")
		result := db.First(&user, userID)
		if result.Error != nil {
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(200, user.UserName)
	}
}

func GetOfferResp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var offer Offer
		offerID := c.Param("id")
		tokenString := c.Request.Header.Get("token")
		userID, err := validateJWT(tokenString)
		if err != nil {
			fmt.Printf("error validating token: %v\n", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		result := db.First(&offer, offerID)
		if result.Error != nil {
			fmt.Printf("error finding offer: %v\n", result.Error)
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		if fmt.Sprint(offer.UserID) != userID {
			fmt.Printf("offer user id: %v, token user id: %v\n", offer.UserID, userID)
			c.JSON(400, gin.H{"error": "user does not own offer"})
			return
		}
		//get users who have messaged the offer 
		var messages []Message 
		result = db.Find(&messages, "offer_id = ?", offerID)
		if result.Error != nil {
			fmt.Printf("error finding messages: %v\n", result.Error)
			c.JSON(400, gin.H{"error": result.Error.Error()})
			return
		}
		var users []User 
		for _, message := range messages {
			var user User
			result = db.First(&user, message.SenderID)
			if result.Error != nil {
				fmt.Printf("error finding user: %v\n", result.Error)
				c.JSON(400, gin.H{"error": result.Error.Error()})
				return 
			}
			if !user.isIn(users) {
				users = append(users, user)
			}
		}
		c.JSON(200, users)
	}
}

func (user User) isIn(users []User) bool {
	for _, u := range users {
		if u.ID == user.ID {
			return true
		}
	}
	return false
}

func DropAllTables(db *gorm.DB) {
	log.Println("Droping all tables")
	err := db.Migrator().DropTable(&User{}, &Photo{}, &Offer{}, &Request{}, &Community{}, &Message{})
	if err != nil {
		log.Fatal("Error Dropping the tables: ", err)
	}
}

