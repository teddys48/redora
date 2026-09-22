package models

import "time"

type Connection struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Host              string    `json:"host"`
	Port              int       `json:"port"`
	Username          string    `json:"username,omitempty"`
	PasswordEncrypted string    `json:"-"`
	DB                int       `json:"db"`
	TLSEnabled        bool      `json:"tlsEnabled"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type ConnectionCreateInput struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	DB         int    `json:"db"`
	TLSEnabled bool   `json:"tlsEnabled"`
}

type ConnectionUpdateInput struct {
	Name       string  `json:"name"`
	Host       string  `json:"host"`
	Port       int     `json:"port"`
	Username   *string `json:"username,omitempty"`
	Password   *string `json:"password,omitempty"`
	DB         int     `json:"db"`
	TLSEnabled bool    `json:"tlsEnabled"`
}
