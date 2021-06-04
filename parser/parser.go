package parser

import (
	"google.golang.org/protobuf/encoding/protojson"
	"log"
)

func ParseSessions(json []byte) []*Session {
	sessions := &Sessions{}
	err := protojson.Unmarshal(json, sessions)
	if err != nil {
		log.Fatalln(err)
	}
	return sessions.GetSessions()
}