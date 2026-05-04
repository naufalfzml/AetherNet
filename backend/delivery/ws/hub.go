package ws

import "github.com/aethernet-0g/aethernet/backend/domain"

type Hub struct {
	Broadcast chan domain.Post
}

func NewHub() *Hub {
	return &Hub{Broadcast: make(chan domain.Post, 64)}
}
