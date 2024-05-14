package model

import (
	"time"
)

type Error struct {
	Message   string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

type Configuration struct {
	Port            int      `json:"port"`
	Host            string   `json:"host"`
	APIKey          string   `json:"apiKey"`
	AllowedOrigins  []string `json:"allowedOrigins"`
	AllowAllOrigins bool     `json:"allowAllOrigins"`
}

type RNG struct {
	ID        string    `json:"id"`
	User      string    `json:"user"`
	RNG       float64   `json:"rng"`
	Timestamp time.Time `json:"timestamp"`
}

type RNGs struct {
	RNGs []RNG `json:"rngs"`
}

type Average struct {
	User    string  `json:"user"`
	Count   int     `json:"count"`
	Average float64 `json:"average"`
}

type Averages struct {
	List []Average `json:"averages"`
}

type Users struct {
	Users []string `json:"users"`
}
