package main

import (
	"fmt"
	"github.com/sashamorecode/Comradery/Server/api"
	"github.com/gin-gonic/gin"
)

// example image Post: curl -X POST 127.0.0.1:8000/images -F "user_id=100" -F "image=@./image.png" -H "Content-Type: multipart/form-data"
// example user Post: curl -X POST 127.0.0.1:8000/users -F "username=testUser" -F "password=testPass" -H "Content-Type: multipart/form-data"
//example get image id=1: curl -X GET '127.0.0.1:8000/images1' > imgtest.jpg
//example get user id=1: curl -X GET '127.0.0.1:8000/users1'
func main() {
	db := api.ConnectDB()
	router := gin.Default()
	api.SetupRoutes(db, router)
	err := router.Run("127.0.0.1:8000")
	if err != nil {
		fmt.Println("Error: ", err)
	}
}
