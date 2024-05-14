package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log"
	"math/big"
	"net/http"
	"os"
	"rngAPI/docs"
	"rngAPI/model"
	"rngAPI/util"
	"strconv"
	"strings"
	"time"
)

var database *sql.DB
var configuration model.Configuration

// Ping godoc
// @Summary Ping Endpoint
// @Description Responds to a Ping with Pong
// @Accept json
// @Produce json
// @Success 200 {string} Pong
// @Router /api/v1/Ping [get]
func Ping(g *gin.Context) {
	g.JSON(http.StatusOK, "Pong")
}

// RandomFloat0to1 godoc
// @Summary Generate a float between 0 and 1
// @Description Generates a float64 between 0 and 1 using crypto/rand to ensure cryptographically robust randomness
// @Accept json
// @Produce json
// @Success 200 {string} model.RNG Result
// @Failure 405 {string} string "Disallowed Username"
// @Param username path string false "Username"
// @Router /api/v1/RandomFloat0to1/{username} [get]
// @Security ApiKeyAuth
func RandomFloat0to1(c *gin.Context) {
	username := c.Param("username")
	if len(username) == 0 || strings.ToLower(username) == "undefined" {
		suffix, err := rand.Int(rand.Reader, big.NewInt(9))
		util.ErrorHandler(err)
		username = "Debug" + suffix.String()
	}

	if strings.ToLower(username) == "all" {
		util.ApiErrorHandler(c, http.StatusMethodNotAllowed, "The username '%s' is reserved.", username)
		return
	}

	var result = model.RNG{
		ID:        uuid.New().String(),
		User:      username,
		RNG:       util.RandomFloat64(),
		Timestamp: time.Now().UTC(),
	}

	tx, err := database.Begin()
	util.ErrorHandler(err)

	stmt, err := tx.Prepare("INSERT INTO RNG(id, username, rng, timestamp) VALUES(?, ?, ?, ?)")
	util.ErrorHandler(err)

	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		util.ErrorHandler(err)
	}(stmt)

	_, err = stmt.Exec(result.ID, result.User, result.RNG, result.Timestamp.Format(time.RFC3339))
	util.ErrorHandler(err)

	err = tx.Commit()
	util.ErrorHandler(err)

	c.IndentedJSON(http.StatusOK, result)
}

// GetUserAverageRNG godoc
// @Summary Gets the average of generated numbers so far
// @Description Returns the average of the generated numbers for a given user or all
// @Accept json
// @Produce json
// @Success 200 {string} model.Average Result
// @Failure 404 {string} string "Username not found"
// @Param username path string false "Username"
// @Router /api/v1/GetUserAverageRNG/{username} [get]
// @Security ApiKeyAuth
func GetUserAverageRNG(c *gin.Context) {
	username := c.Param("username")
	if len(username) == 0 || strings.ToLower(username) == "undefined" {
		username = "All"
	}

	stmt, err := database.Prepare("SELECT username, average, count FROM AverageRNG WHERE username LIKE ?")
	util.ErrorHandler(err)

	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		util.ErrorHandler(err)
	}(stmt)

	var user string
	var average float64
	var count int
	err = stmt.QueryRow(username).Scan(&user, &average, &count)
	if err != nil {
		util.ApiErrorHandler(c, http.StatusNotFound, "Could not find user '%s'", username)
		return
	}

	var result = model.Average{
		User:    username,
		Average: average,
		Count:   count,
	}

	c.IndentedJSON(http.StatusOK, result)
}

// GetAllAveragesRNG godoc
// @Summary Gets the averages of generated numbers so far
// @Description Returns the average of the generated numbers for all users as a list
// @Accept json
// @Produce json
// @Success 200 {string} model.Averages Result
// @Router /api/v1/GetAllAveragesRNG [get]
// @Security ApiKeyAuth
func GetAllAveragesRNG(c *gin.Context) {
	rows, err := database.Query("SELECT username, average, count FROM AverageRNG WHERE username NOT LIKE 'all'")
	util.ErrorHandler(err)
	defer func(rows *sql.Rows) {
		err := rows.Close()
		util.ErrorHandler(err)
	}(rows)

	var averages []model.Average
	for rows.Next() {
		var username string
		var average float64
		var count int
		err = rows.Scan(&username, &average, &count)
		util.ErrorHandler(err)
		averages = append(averages, model.Average{User: username, Average: average, Count: count})
	}

	err = rows.Err()
	util.ErrorHandler(err)

	var result = model.Averages{List: averages}

	c.IndentedJSON(http.StatusOK, result)
}

// GetUsers godoc
// @Summary Gets the list of users
// @Description Returns a list of all users who had made calls to the RNG endpoint
// @Accept json
// @Produce json
// @Success 200 {string} model.Users Result
// @Router /api/v1/GetUsers [get]
// @Security ApiKeyAuth
func GetUsers(c *gin.Context) {
	rows, err := database.Query("SELECT username FROM RNG GROUP BY username")
	util.ErrorHandler(err)
	defer func(rows *sql.Rows) {
		err := rows.Close()
		util.ErrorHandler(err)
	}(rows)

	var users []string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		util.ErrorHandler(err)
		users = append(users, name)
	}

	err = rows.Err()
	util.ErrorHandler(err)

	var result = model.Users{Users: users}

	c.IndentedJSON(http.StatusOK, result)
}

// GetGenerationDetails godoc
// @Summary Returns a paged list of results
// @Description Returns a list in the specified paging range of generated numbers so far
// @Accept json
// @Produce json
// @Success 200 {string} model.RNGs Result
// @Param page query string false "Page"
// @Param count query string false "Count"
// @Router /api/v1/GetGenerationDetails [get]
// @Security ApiKeyAuth
func GetGenerationDetails(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		util.ApiErrorHandler(c, http.StatusBadRequest, "Conversion failed for page '%s'", pageStr)
		return
	}
	page -= 1
	countStr := c.DefaultQuery("count", "15")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		util.ApiErrorHandler(c, http.StatusBadRequest, "Conversion failed for count '%s'", countStr)
		return
	}

	stmt, err := database.Prepare("SELECT id, username, rng, timestamp FROM RNG ORDER BY timestamp DESC LIMIT ? OFFSET ?")
	util.ErrorHandler(err)

	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		util.ErrorHandler(err)
	}(stmt)

	rows, err := stmt.Query(count, count*page)
	if err != nil {
		err = c.AbortWithError(http.StatusInternalServerError, err)
		util.ErrorHandler(err)
		return
	}

	var rngList []model.RNG
	for rows.Next() {
		var id string
		var username string
		var rng float64
		var timestamp string
		err = rows.Scan(&id, &username, &rng, &timestamp)
		util.ErrorHandler(err)

		parsedTime, err := time.Parse(time.RFC3339, timestamp)
		util.ErrorHandler(err)

		rngList = append(rngList, model.RNG{ID: id, User: username, RNG: rng, Timestamp: parsedTime})
	}

	err = rows.Err()
	util.ErrorHandler(err)

	var result = model.RNGs{RNGs: rngList}

	c.IndentedJSON(http.StatusOK, result)
}

func ValidateAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		APIKey := c.Request.Header.Get("Authorization")
		if APIKey != configuration.APIKey {
			util.ApiErrorHandler(c, http.StatusUnauthorized, "API Key does not match")
			return
		}

		return
	}
}

func GenerateDefaultConfig() {
	var configuration = model.Configuration{
		Port:            9999,
		Host:            "0.0.0.0",
		APIKey:          uuid.New().String(),
		AllowedOrigins:  []string{"http://localhost", "http://localhost:8080", "http://localhost:5173"},
		AllowAllOrigins: false,
	}

	content, err := json.Marshal(configuration)
	util.ErrorHandler(err)

	err = os.WriteFile("config.json", content, 0644)
	util.ErrorHandler(err)
}

func ReadConfiguration() {
	_, openErr := os.OpenFile("config.json", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(openErr, os.ErrNotExist) {
		GenerateDefaultConfig()

		config, readErr := os.ReadFile("config.json")
		util.ErrorHandler(readErr)

		unmarshalErr := json.Unmarshal(config, &configuration)
		util.ErrorHandler(unmarshalErr)
	} else {
		config, readErr := os.ReadFile("config.json")
		util.ErrorHandler(readErr)

		unmarshalErr := json.Unmarshal(config, &configuration)
		if unmarshalErr != nil {
			removeErr := os.Remove("config.json")
			util.ErrorHandler(removeErr)

			GenerateDefaultConfig()
			log.Fatal("Invalid configuration found, file has been reset to defaults. Please rerun the application.")
		}
	}
}

// @title RNG Service API
// @version 1.0
// @description This is a WebAPI providing cryptographically secure RNG
// @license.name MPL-2.0 License
// @license.url https://www.mozilla.org/en-US/MPL/2.0/
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description API Key defined in the configuration file
func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	ReadConfiguration()

	db, err := sql.Open("sqlite3", "./RNG.sqlite")
	util.ErrorHandler(err)
	database = db

	defer func(db *sql.DB) {
		err := db.Close()
		util.ErrorHandler(err)
	}(database)

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS RNG (id TEXT NOT NULL PRIMARY KEY, username TEXT, rng FLOAT, timestamp TEXT);
	`
	_, err = database.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	sqlStmt = `
	CREATE VIEW IF NOT EXISTS AverageRNG AS
	SELECT username,
		   AVG(rng) AS average,
		   COUNT(username) AS count
	FROM RNG
	GROUP BY username
	UNION ALL
	SELECT
		'All',
		AVG(rng),
		COUNT(username)
	FROM RNG;
	`
	_, err = database.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     configuration.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH"},
		AllowHeaders:     []string{"Origin", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowAllOrigins:  configuration.AllowAllOrigins,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	docs.SwaggerInfo.BasePath = "/"
	v1 := router.Group("/api/v1")
	{
		v1.GET("/Ping", Ping)
		v1.GET("/GetUsers", ValidateAPIKey(), GetUsers)
		v1.GET("/GetGenerationDetails", ValidateAPIKey(), GetGenerationDetails)
		v1.GET("/RandomFloat0to1", ValidateAPIKey(), RandomFloat0to1)
		v1.GET("/GetAverageRNG", ValidateAPIKey(), GetUserAverageRNG)
		v1.GET("/GetAllAveragesRNG", ValidateAPIKey(), GetAllAveragesRNG)
		v1.GET("/RandomFloat0to1/:username", ValidateAPIKey(), RandomFloat0to1)
		v1.GET("/GetAverageRNG/:username", ValidateAPIKey(), GetUserAverageRNG)
	}
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	err = router.Run(fmt.Sprintf("%s:%d", configuration.Host, configuration.Port))
	util.ErrorHandler(err)
}
