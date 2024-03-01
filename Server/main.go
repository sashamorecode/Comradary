package main

import (
	"fmt"
	"net/http"
	"github.com/gin-gonic/gin"
	"database/sql"
	"log"
	_ "github.com/go-sql-driver/mysql"
	"math/rand"
	"crypto/sha256"
	"strconv"
)

type Offer struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ImageId     int     `json:"image_ids"`
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



func setUpDB(db *sql.DB) {
	var err error
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS images (
				id INT AUTO_INCREMENT PRIMARY KEY,
				user_id INT, FOREIGN KEY (user_id) REFERENCES users(id),
				post_id INT, FOREIGN KEY (post_id) REFERENCES offers(id),
				image_path TEXT
			)`)
	if err != nil { log.Fatal("Error creating image Table: ", err) }

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				username TEXT,
				password_hash TEXT,
				password_salt TEXT
			)`)
	if err != nil { log.Fatal("Error creating users table: ", err) }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS offers (
				id INT AUTO_INCREMENT PRIMARY KEY,
				title TEXT,
				description TEXT,
				user_id INT, FOREIGN KEY (user_id) REFERENCES users(id),
				group_image_id INT, FOREIGN KEY (group_image_id) REFERENCES images(group_id)
			)`)
	if err != nil { log.Fatal("Error creating offers table: ", err) }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comunities (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name TEXT,
				location BIGINT,
				description TEXT
			)`)
	if err != nil { log.Fatal("Error creating comunit_users table: ", err) }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comunities_users (
				comunity_id INT,
				FOREIGN KEY (comunity_id) REFERENCES comunities(id),
				user_id INT,
				FOREIGN KEY (user_id) REFERENCES users(id)
			)`)
	if err != nil { log.Fatal("Error creating table: ", err) }
	
}

func getOffer(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		comunity_id := c.Param("comunity_id")
		rows, err := db.Query(`SELECT id, title, description, user_id, image_id
					FROM offers WHERE comunity_id = ?`, comunity_id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var offers []Offer
		for rows.Next() {
			var post Offer
			err := rows.Scan(&post.ID, &post.Title, &post.Description, &post.UserID, &post.ImageId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			offers = append(offers, post)
		}
		c.JSON(http.StatusOK, offers)
	}
}

func createOffer(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newOffer Offer
		err := c.BindJSON(&newOffer)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_, err = db.Exec("INSERT INTO offers (title, description, user_id, image_group_id) VALUES (?, ?)", newOffer.Title, newOffer.Description)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusCreated, newOffer)
	}
}

func createImage(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		form, _ := c.MultipartForm()
		
		images := form.File["image"]
		if len(images) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please upload a single image"})
			return
		}
		image := images[0]
		user_ids := form.Value["user_id"]
		if len(user_ids) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a user_id"})
			return
		}
		user_id, err := strconv.Atoi(user_ids[0])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid user_id"})
			return
		}
		post_ids := form.Value["post_id"]
		post_id := "NULL"
		if len(post_ids) != 0 {
			post_id = post_ids[0]
		}
		img_path := "./images/" + fmt.Sprint(rand.Intn(1000000)) + image.Filename
		err = c.SaveUploadedFile(image, img_path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_, err = db.Exec(`INSERT INTO images (user_id, image_path) 
				  SELECT id "?" FROM users WHERE
				  	id = ?
				  LIMIT 1`, img_path, user_id)
		if err != nil {
			fmt.Println(user_id, post_id, img_path)
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

func createUser(db *sql.DB) gin.HandlerFunc {
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

func connectDB() *sql.DB {
	db, err := sql.Open("mysql", "root:secure@tcp(127.0.0.1:3306)/mysql")
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func dropAllTables(db *sql.DB) {
	var err error
	_, err = db.Exec("DROP TABLE IF EXISTS offers")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS comunities")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS comunities_users")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS images")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		log.Fatal(err)
	}
}
func main() {
	db := connectDB()
	defer db.Close()
	setUpDB(db)
	router := gin.Default()
	router.GET("/offers:comunity_id", getOffer(db))
	router.POST("/offers", createOffer(db))
	router.POST("/images", createImage(db))
	router.POST("/users", createUser(db))
	err := router.Run("127.0.0.1:8000")
	if err != nil {
		fmt.Println("Error: ", err)
	}
}
