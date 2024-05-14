package util

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"rngAPI/model"
	"time"
)

func ErrorHandler(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ApiErrorHandler(c *gin.Context, status int, message string, params ...any) {
	var errorJson model.Error
	if len(params) == 0 {
		errorJson = model.Error{
			Message:   message,
			Timestamp: time.Now(),
		}
	} else {
		errorJson = model.Error{
			Message:   fmt.Sprintf(message, params),
			Timestamp: time.Now(),
		}
	}
	c.AbortWithStatusJSON(status, errorJson)
}
