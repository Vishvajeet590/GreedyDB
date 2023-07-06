package main

import (
	"GreedyDB/db"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type input struct {
	Command string `json:"command"`
}
type response struct {
	Code    int         `json:"code"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
}

func CommandHandler(c *gin.Context) {
	var command input
	err := c.BindJSON(&command)
	if err != nil {
		FormatResponseMessage(nil, http.StatusBadRequest, fmt.Errorf("json format error"), c)
		return
	}

	parsedQuery, err := db.ParseCommand(strings.TrimSpace(command.Command))
	if err != nil {
		FormatResponseMessage(nil, http.StatusBadRequest, err, c)
		return
	}

	switch parsedQuery.Type {
	case "SET":
		err = Store.Set(parsedQuery)
		if err != nil {
			FormatResponseMessage(nil, http.StatusInternalServerError, err, c)
			return
		}
		c.Status(http.StatusCreated)

	case "GET":
		value, err := Store.Get(parsedQuery)
		if err != nil {
			FormatResponseMessage(nil, http.StatusNotFound, err, c)
			return
		}
		FormatResponseMessage(value, http.StatusOK, nil, c)

	case "QPUSH":
		err := Store.QPush(parsedQuery)
		if err != nil {
			FormatResponseMessage(nil, http.StatusInternalServerError, err, c)
			return
		}
		c.Status(http.StatusCreated)
	case "QPOP":
		poppedValue, err := Store.QPop(parsedQuery)
		if err != nil {
			FormatResponseMessage(nil, http.StatusInternalServerError, err, c)
			return
		}
		FormatResponseMessage(poppedValue, http.StatusOK, nil, c)

	case "BQPOP":
		poppedValue, err := Store.BQPop(parsedQuery)
		if err != nil {
			return
		}
		if err != nil {
			FormatResponseMessage(nil, http.StatusInternalServerError, err, c)
			return
		}
		FormatResponseMessage(poppedValue, http.StatusOK, nil, c)

	}
}

// FormatResponseMessage utility function
func FormatResponseMessage(res interface{}, code int, err error, c *gin.Context) {
	if err != nil {
		c.JSON(code, &response{
			Code:    code,
			Value:   nil,
			Message: err.Error(),
		})

	} else {
		c.AbortWithStatusJSON(code, &response{
			Code:    code,
			Value:   res,
			Message: "Success",
		})
	}
}
