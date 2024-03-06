package api

import (
	"crypto/rand"
	"log"
	"path/filepath"
	"strconv"
	"golang.org/x/crypto/bcrypt"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"github.com/golang-jwt/jwt/v4"
	"time"
	"fmt"
)
//creaet random key
var jwtKey = []byte("1asf12vsr2agrasg892yh780gahe780g0sbh8")
type User struct {
	gorm.Model
	UserName     string
	Email        string 	`gorm:"unique"`
	PasswordHash string	
	Offers       []Offer    `gorm:"foreignKey:UserID"`
	Requests     []Request  `gorm:"foreignKey:UserID"`
	ProfilePhoto *Photo     `gorm:"foreignKey:UserID"`
	Communities  []Community `gorm:"many2many:user_communities;"`
}
type Photo struct {
	gorm.Model
	Path       string  `gorm:"unique"`
	OfferID    *uint
	RequestID  *uint
	UserID     uint
}

type Offer struct {
	gorm.Model
	Title       string	`json:"title"`
	Description string	`json:"description"`
	Photos      []Photo     `gorm:"foreignKey:OfferID"`
	UserID      uint 	`json:"user_id"`
	CommunityID uint 	`json:"community_id"`	
}

type Request struct {
	gorm.Model
	Title       string
	Description string
	Photos      []Photo   `gorm:"foreignKey:RequestID"`
	UserID      uint 
	CommunityID uint
}

type Community struct {
	gorm.Model
	Name       string
	Lat        float64
	Lon        float64
	Users      []User    `gorm:"many2many:user_communities;"`
	Offers     []Offer   `gorm:"foreignKey:CommunityID"`
	Requests   []Request `gorm:"foreignKey:CommunityID"`
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
}

func InsertTestData(db *gorm.DB) {
	// Create a Community 
	community := Community{Name: "test", Lat: 0.0, Lon: 0.0}
	result := db.Create(&community)
	if result.Error != nil {
		log.Fatal("Error creating community: ", result.Error)
	}
	// Create a User
	user := User{UserName: "test", Email: "sas@gmail.com", PasswordHash: "test" }
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
	err = db.AutoMigrate(&User{}, &Photo{}, &Offer{}, &Request{})
	if err != nil {
		log.Fatal("Error Migrating the database: ", err)
	}
	log.Println("Inserting test data")
	//InsertTestData(db)
	return db
}

func CreateImages(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		imageFiles := form.File["images"]
		var photos []Photo
		for _, file := range imageFiles {
			var newRand int
			_, err := rand.Read([]byte(strconv.Itoa(newRand)))
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			path := filepath.Join("images", strconv.Itoa(newRand)+file.Filename)
			photo := Photo{Path: path}
			photos = append(photos, photo)
			err = c.SaveUploadedFile(file, path)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			result := db.Create(&photo)
			if result.Error != nil {
				c.JSON(400, gin.H{"error": result.Error.Error()})
				return
			}
		}
		c.JSON(200, photos)
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

func validateJWT(tokenString string) (string, error ){
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

func CreateOffer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		log.Println("Creating offer")
		var offer OfferInput
		err = c.BindJSON(&offer)
		if err != nil {
			fmt.Printf("error binding json: %v\n", err)
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


func DropAllTables(db *gorm.DB) {
	log.Println("Droping all tables")
	err := db.Migrator().DropTable(&User{}, &Photo{}, &Offer{}, &Request{}, &Community{})
	if err != nil {
		log.Fatal("Error Dropping the tables: ", err)
	}
}

func SetupRoutes(db *gorm.DB, router *gin.Engine) {
	router.POST("/images", CreateImages(db))
	router.POST("/signup", SignUp(db))
	router.POST("/signin", SignIn(db))
	router.POST("/offers", CreateOffer(db))
	router.GET("/offers/:id", GetOffersByCommunityId(db))

}
