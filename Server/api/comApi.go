package api

import (
	"fmt"
	"net/http"
	"github.com/gin-gonic/gin"
	"database/sql"
	"log"
	_ "github.com/go-sql-driver/mysql"
	"math/rand"
	"crypto/sha256"
	"os"
	"strconv"
)

type Offer struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ImageSetID  int     `json:"image_set_id"`
	ComunityID  int     `json:"comunity_id"`
	UserID      int     `json:"user_id"`
}


type Request struct {
	ID          int    `json:"id"`
	Title  	    string `json:"title"`
	Description string `json:"description"`
	UserID      int    `json:"user_id"`
}

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	ProfileImageId int  `json:"profile_image_id"`
	PasswordHash string `json:"password_hash"`
	PasswordSalt string `json:"password_salt"`
}

type Comunity struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Location     int64  `json:"location"`
	Description  string `json:"description"`
}

type Image struct {
	ID           int    `json:"id"`
	ImagePath    string `json:"image_path"`
}



func SetUpDB(db *sql.DB) {
	var err error
	
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS images (
				id INT AUTO_INCREMENT PRIMARY KEY,
				image_path TEXT NOT NULL
			)`)
	if err != nil { log.Fatal("Error creating image Table: ", err) }

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS image_sets (
				id INT AUTO_INCREMENT PRIMARY KEY,
				set_id INT NOT NULL,
				image_id INT NOT NULL,
				FOREIGN KEY (image_id) REFERENCES images(id))`)
	if err != nil { log.Fatal("Error creating image_sets table: ", err) }

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				username TEXT,
				password_hash TEXT,
				password_salt TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				profile_image_id INT DEFAULT -1 REFERENCES images(id)
			)`)
	if err != nil { log.Fatal("Error creating users table: ", err) }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comunities (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name TEXT,
				location BIGINT,
				description TEXT,
				image_set_id INT DEFAULT -1,
				FOREIGN KEY (image_set_id) REFERENCES image_sets(id)
			)`)
	if err != nil { log.Fatal("Error creating comunities table: ", err) }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS offers (
				id INT AUTO_INCREMENT PRIMARY KEY,
				title TEXT,
				description TEXT,
				image_set_id INT DEFAULT -1,
				FOREIGN KEY (image_set_id) REFERENCES image_sets(id),
				user_id INT NOT NULL,
				FOREIGN KEY (user_id) REFERENCES users(id),
				comunity_id INT NOT NULL,
				FOREIGN KEY (comunity_id) REFERENCES comunities(id)
				)`)
	if err != nil { log.Fatal("Error creating offers table: ", err) }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comunities_users (
				comunity_id INT,
				FOREIGN KEY (comunity_id) REFERENCES comunities(id),
				user_id INT,
				FOREIGN KEY (user_id) REFERENCES users(id),
				create_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`)
	if err != nil { log.Fatal("Error creating comunities_users table: ", err) }
	
}

func GetOfferByCommunityId(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		comunity_id := c.Param("comunity_id")
		rows, err := db.Query(`SELECT id, title, description, user_id, comunity_id, image_set_id
					FROM offers WHERE comunity_id = ?`, comunity_id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var offers []Offer
		for rows.Next() {
			var post Offer
			err := rows.Scan(&post.ID, &post.Title, &post.Description, &post.UserID, &post.ComunityID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			offers = append(offers, post)
		}
		c.JSON(http.StatusOK, offers)
	}
}

func CreateOffer(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newOffer Offer
		form, err := c.MultipartForm()
		if err != nil { 
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) 
		}
		newOffer.UserID, err = strconv.Atoi(form.Value["user_id"][0])
		if err != nil { 
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid user id"})
		}
		newOffer.ComunityID, err = strconv.Atoi(form.Value["comunity_id"][0])
		if err != nil { 
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid comunity id"})
		}
		newOffer.Title = form.Value["title"][0]
		newOffer.Description = form.Value["description"][0]
		newOffer.ImageSetID, err = strconv.Atoi(form.Value["image_set_id"][0])
		if err != nil { 
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid image set id"})
		}
		_, err = db.Exec(`INSERT INTO offers 
				(title, description, user_id, comunity_id, image_set_id) VALUES (?, ?, ?, ?, ?)`,
				newOffer.Title, newOffer.Description, newOffer.UserID,
				newOffer.ComunityID, newOffer.ComunityID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"SQL error": err.Error()})
			return
		}
		result := db.QueryRow("SELECT LAST_INSERT_ID() FROM offers")
		err = result.Scan(&newOffer.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting last insert id"})
			return
		}
		
		c.JSON(http.StatusCreated, newOffer)
	}
}

func GetImageById(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		image_id := c.Param("image_id")
		log.Println(image_id)
		_, err := strconv.Atoi(image_id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid image id"})
			return
		}
		results := db.QueryRow("SELECT id, image_path FROM images WHERE id = ?", image_id)
		var image Image
		err = results.Scan(&image.ID, &image.ImagePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error querying database": err.Error()})
			return
		}
		imageData, err := os.Open(image.ImagePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError,
			gin.H{"error loading image form drive": err.Error()})
		}
		defer imageData.Close()
		c.File(image.ImagePath)
	}
}

func CreateImage(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		form, _ := c.MultipartForm()
		
		images := form.File["image"]
		if len(images) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please upload a single image"})
			return
		}
		image := images[0]
		img_path := "./images/" + fmt.Sprint(rand.Intn(1000000)) + image.Filename
		err := c.SaveUploadedFile(image, img_path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_, err = db.Exec(`INSERT INTO images (image_path) VALUES (?)`, img_path)
		if err != nil {
			fmt.Println(img_path)
			c.JSON(http.StatusInternalServerError, gin.H{"SQL error": err.Error()})
			return
		}
		result := db.QueryRow("SELECT LAST_INSERT_ID() FROM images")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var img_id int
		err = result.Scan(&img_id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"image_id": img_id})
	}
}

func createImageGroup(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		imageids := form.File["image_ids"]
		res, err := db.Query("SELECT MAX(set_id) FROM image_sets")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var set_id int
		err = res.Scan(&set_id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		set_id++
		for _, imageid := range imageids {
			res := db.QueryRow("SELECT id FROM images WHERE id = ?", imageid)
			err := res.Scan()
			if err != nil {
				c.JSON(http.StatusInternalServerError,
					gin.H{"error image not found": err.Error()})
				return
			}
			_, err = db.Exec(`INSERT INTO image_sets 
					(set_id, image_id) VALUES (?, ?)`,
					set_id, imageid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"set_id": set_id})
		}
	}
}

func GetUserById(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user_id := c.Param("user_id")
		results := db.QueryRow("SELECT id, username FROM users WHERE id = ?", user_id)
		var user User
		err := results.Scan(&user.ID, &user.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

func CreateUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newUser User
		form, _ := c.MultipartForm()
		if len(form.Value["username"]) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a username"})
			return
		}
		newUser.Username = form.Value["username"][0]
		results := db.QueryRow("SELECT id FROM users WHERE username = ?", newUser.Username)
		err := results.Scan()
		if err != sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
			return
		}
		if len(form.Value["password"]) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a password"})
		}
		password := form.Value["password"][0]
		newUser.PasswordSalt = fmt.Sprint(rand.Intn(1000000))
		newUser.PasswordHash = fmt.Sprintf("%x", sha256.Sum256([]byte(password + newUser.PasswordSalt)))
		
		
		_, err = db.Exec("INSERT INTO users (username, password_hash, password_salt) VALUES (?, ?, ?)", newUser.Username, newUser.PasswordHash, newUser.PasswordSalt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result := db.QueryRow("SELECT LAST_INSERT_ID() FROM users")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var user_id int
		err = result.Scan(&user_id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"user_id": user_id})
	}
}

func ConnectDB() *sql.DB {
	db, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/mysql")
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging the database: ", err)
	}
	return db
}

func DropAllTables(db *sql.DB) {
	var err error
	_, err = db.Exec("DROP TABLE IF EXISTS comunities_users")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS offers")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS comunities")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS image_sets")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS images")
	if err != nil {
		log.Fatal(err)
	}
}
//func main() {
//	db := connectDB()
//	defer db.Close()
//	dropAllTables(db)
//	setUpDB(db)
//	router := gin.Default()
//	router.GET("/offers:comunity_id", getOffer(db))
//	router.POST("/offers", createOffer(db))
//	router.POST("/images", createImage(db))
//	router.POST("/users", createUser(db))
//	err := router.Run("127.0.0.1:8000")
//	if err != nil {
//		fmt.Println("Error: ", err)
//	}
//}
