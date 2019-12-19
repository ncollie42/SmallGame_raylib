package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	myProto "game/proto"

	"google.golang.org/grpc"
)

var playerMap = myProto.AllPlayers{}

type server struct {
	myProto.UnimplementedUpdateStateServer
}

func (s *server) ConstUpdate(stream myProto.UpdateState_ConstUpdateServer) error {
	for {
		t := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-t.C:
				stream.Send(&playerMap)
			}
		}
	}
	return nil
}

func (s *server) Join(ctx context.Context, p *myProto.Player) (*myProto.AllPlayers, error) {
	log.Printf("player %s joined", p.GetName())
	playerMap.PlayerMap[p.GetName()] = p
	return &playerMap, nil
}

func (s *server) Leave(ctx context.Context, p *myProto.Player) (*myProto.AllPlayers, error) {
	log.Printf("player %s left", p.GetName())
	delete(playerMap.PlayerMap, p.GetName())
	return &playerMap, nil
}

func (s *server) Update(ctx context.Context, p *myProto.Player) (*myProto.AllPlayers, error) {
	// log.Printf("updating: player %s", p.GetName())
	playerMap.PlayerMap[p.GetName()].Location = p.GetLocation()
	return &playerMap, nil
}

func main() {
	playerMap.PlayerMap = make(map[string]*myProto.Player)
	go func() {
		tic := time.NewTicker(time.Second * 2)
		for {
			select {
			case <-tic.C:
				log.Println("------------------")
				// for name, tt := range playerMap.PlayerMap {
				// 	log.Println(name, tt.GetLocation())
				// }
				log.Println(playerMap.PlayerMap)
			}
		}
	}()
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	myProto.RegisterUpdateStateServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to server: %v", err)
	}
	fmt.Println("DID I MAKE IT HERE?")
}
