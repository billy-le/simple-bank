package gapi

import (
	"context"
	"database/sql"
	"testing"
	"time"

	mockdb "github.com/billy-le/simple-bank/db/mock"
	db "github.com/billy-le/simple-bank/db/sqlc"
	"github.com/billy-le/simple-bank/pb"
	"github.com/billy-le/simple-bank/token"
	"github.com/billy-le/simple-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRpcUpdateUser(t *testing.T) {
	newUser, _ := createRandomUser(t)

	newName := util.RandomOwner()
	newEmail := util.RandomEmail()
	invalidArg := ""

	testCases := []struct {
		name           string
		req            *pb.UpdateUserRequest
		buildStubs     func(store *mockdb.MockStore)
		buildContext   func(t *testing.T, tokenMaker token.Maker) context.Context
		checkResponses func(t *testing.T, res *pb.UpdateUserResponse, err error)
	}{
		{
			name: "Ok",
			req: &pb.UpdateUserRequest{
				Username: newUser.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					Username: newUser.Username,
					FullName: sql.NullString{
						String: newName,
						Valid:  true,
					},
					Email: sql.NullString{
						String: newEmail,
						Valid:  true,
					},
				}

				updatedUser := db.User{
					Username:          newUser.Username,
					HashedPassword:    newUser.HashedPassword,
					FullName:          newName,
					Email:             newEmail,
					PasswordChangedAt: newUser.PasswordChangedAt,
					CreatedAt:         newUser.CreatedAt,
					IsEmailVerified:   newUser.IsEmailVerified,
				}

				store.EXPECT().UpdateUser(gomock.Any(), gomock.Eq(arg)).Times(1).Return(updatedUser, nil)

			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, newUser.Username, time.Minute)
			},
			checkResponses: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				updatedUser := res.GetUser()
				require.Equal(t, newUser.Username, updatedUser.Username)
				require.Equal(t, newName, updatedUser.FullName)
				require.Equal(t, newEmail, updatedUser.Email)
			},
		},
		{
			name: "UserNotFound",
			req: &pb.UpdateUserRequest{
				Username: newUser.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(1).Return(db.User{}, sql.ErrNoRows)

			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, newUser.Username, time.Minute)
			},
			checkResponses: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.NotFound)
			},
		},
		{
			name: "ExpiredToken",
			req: &pb.UpdateUserRequest{
				Username: newUser.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, newUser.Username, -time.Minute)
			},
			checkResponses: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.Unauthenticated)
			},
		},
		{
			name: "InvalidEmail",
			req: &pb.UpdateUserRequest{
				Username: newUser.Username,
				FullName: &newName,
				Email:    &invalidArg,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, newUser.Username, time.Minute)
			},
			checkResponses: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.InvalidArgument)
			},
		},
		{
			name: "InvalidFullName",
			req: &pb.UpdateUserRequest{
				Username: newUser.Username,
				FullName: &invalidArg,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, newUser.Username, time.Minute)
			},
			checkResponses: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.InvalidArgument)
			},
		},
		{
			name: "InvalidPassword",
			req: &pb.UpdateUserRequest{
				Username: newUser.Username,
				Password: &invalidArg,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, newUser.Username, time.Minute)
			},
			checkResponses: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.InvalidArgument)
			},
		},
		{
			name: "PermissionDenied",
			req: &pb.UpdateUserRequest{
				Username: "da",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, newUser.Username, time.Minute)
			},
			checkResponses: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.PermissionDenied)
			},
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			storeCtrl := gomock.NewController(t)
			defer storeCtrl.Finish()

			store := mockdb.NewMockStore(storeCtrl)
			testCase.buildStubs(store)

			server := newTestServer(t, store, nil)

			ctx := testCase.buildContext(t, server.tokenMaker)
			res, err := server.UpdateUser(ctx, testCase.req)

			testCase.checkResponses(t, res, err)
		})
	}
}
