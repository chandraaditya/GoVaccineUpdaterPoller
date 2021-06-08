package parser

import (
	"google.golang.org/protobuf/encoding/protojson"
	"log"
)

func ParseSessions(json []byte) ([]*Session, error) {
	sessions := &Sessions{}
	err := protojson.Unmarshal(json, sessions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return sessions.GetSessions(), nil
}
