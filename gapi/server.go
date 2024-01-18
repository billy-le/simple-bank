package gapi

import (
	"fmt"

	db "github.com/billy-le/simple-bank/db/sqlc"
	"github.com/billy-le/simple-bank/pb"
	"github.com/billy-le/simple-bank/token"
	"github.com/billy-le/simple-bank/util"
)

type Server struct {
	pb.UnimplementedSimpleBankServer
	store      db.Store
	config     util.Config
	tokenMaker token.Maker
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token mater: %w", err)
	}

	server := &Server{store: store, tokenMaker: tokenMaker, config: config}

	return server, nil
}
