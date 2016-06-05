package main

type Work interface {
	GetID() string
	GetType() string
	GetRespValue() string
}
