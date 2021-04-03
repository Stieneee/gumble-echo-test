package main

import (
	"log"

	"layeh.com/gumble/gumble"
)

// MumbleListener Handle mumble events
type MumbleListener struct{}

var pingSeenCount = 0
var pingSentCount = 0

var pongSeenCount = 0
var pongSentCount = 0

func (l *MumbleListener) mumbleConnect(e *gumble.ConnectEvent) {
	log.Println("Mumble Connected")
}

func (l *MumbleListener) mumbleUserChange(e *gumble.UserChangeEvent) {
	log.Println("UserChange", e)
}

func (l *MumbleListener) onTextMessage(e *gumble.TextMessageEvent) {
	log.Println("textMessage", e)
	if e.TextMessage.Message == "Ping" {
		pingSeenCount++
		e.Sender.Channel.Send("Pong", false)
		pongSentCount++
		// e.Sender.Send("Pong")
	} else if e.TextMessage.Message == "Pong" {
		pongSeenCount++
	}

	log.Println(pingSentCount, pingSeenCount, pongSeenCount, pongSentCount)
}
