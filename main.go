package main

import (
	"GreedyDB/db"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var Store db.Store

func main() {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.Default())
	router.GET("/execute", CommandHandler)

	Store = db.NewDataStore()
	router.Run(":8080")
}
