package main

import (
	"fmt"
	"github.com/sashamorecode/comradary/server/api"
	"github.com/gin-gonic/gin"
)

// example image Post: curl -X POST 127.0.0.1:8000/images -F "user_id=100" -F "image=@./image.png" -H "Content-Type: multipart/form-data"
// example user Post: curl -X POST 127.0.0.1:8000/users -F "username=testUser" -F "password=testPass" -H "Content-Type: multipart/form-data"
//example get image id=1: curl -X GET '127.0.0.1:8000/images1' > imgtest.jpg
//example get user id=1: curl -X GET '127.0.0.1:8000/users1'
func main() {
	db := api.ConnectDB()
	defer db.Close()
	api.DropAllTables(db)
	api.SetUpDB(db)
	router := gin.Default()
	router.GET("/offers:comunity_id", api.GetOffer(db))
	router.POST("/offers", api.CreateOffer(db))
	router.GET("/images:image_id", api.GetImageById(db))
	router.POST("/images", api.CreateImage(db))
	router.GET("/users:user_id", api.GetUserById(db))
	router.POST("/users", api.CreateUser(db))
	err := router.Run("127.0.0.1:8000")
	if err != nil {
		fmt.Println("Error: ", err)
	}
}
