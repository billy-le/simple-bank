package gapi

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mockdb "github.com/billy-le/simple-bank/db/mock"
	db "github.com/billy-le/simple-bank/db/sqlc"
	"github.com/billy-le/simple-bank/pb"
	"github.com/billy-le/simple-bank/util"
	"github.com/billy-le/simple-bank/worker"
	mockwk "github.com/billy-le/simple-bank/worker/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eqCreateUserTxParamsMatcher struct {
	arg      db.CreateUserTxParams
	password string
	user     db.User
}

func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	actualArg, ok := x.(db.CreateUserTxParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(expected.password, actualArg.HashedPassword)
	if err != nil {
		return false
	}
	expected.arg.HashedPassword = actualArg.HashedPassword

	if !reflect.DeepEqual(expected.arg.CreateUserParams, actualArg.CreateUserParams) {
		return false
	}

	err = actualArg.AfterCreate(expected.user)

	return err == nil
}

func (expected eqCreateUserTxParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", expected.arg, expected.password)
}

func EqCreateUserTxParams(arg db.CreateUserTxParams, password string, user db.User) gomock.Matcher {
	return eqCreateUserTxParamsMatcher{arg, password, user}
}

func TestRpcCreateUser(t *testing.T) {
	newUser, password := createRandomUser(t)

	testCases := []struct {
		name           string
		req            *pb.CreateUserRequest
		buildStubs     func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor)
		checkResponses func(t *testing.T, res *pb.CreateUserResponse, err error)
	}{
		{
			name: "Ok",
			req: &pb.CreateUserRequest{
				Username: newUser.Username,
				FullName: newUser.FullName,
				Email:    newUser.Email,
				Password: password,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				arg := db.CreateUserTxParams{
					CreateUserParams: db.CreateUserParams{
						Username: newUser.Username,
						FullName: newUser.FullName,
						Email:    newUser.Email,
					},
				}

				store.EXPECT().CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password, newUser)).Times(1).Return(db.CreateUserTxResult{User: newUser}, nil)

				taskPayload := &worker.PayloadSendVerifyEmail{
					Username: newUser.Username,
				}
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any()).Times(1).Return(nil)
			},
			checkResponses: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				createdUser := res.GetUser()
				require.Equal(t, newUser.Username, createdUser.Username)
				require.Equal(t, newUser.FullName, createdUser.FullName)
				require.Equal(t, newUser.Email, createdUser.Email)

			},
		},
		{
			name: "InternalError",
			req: &pb.CreateUserRequest{
				Username: newUser.Username,
				FullName: newUser.FullName,
				Email:    newUser.Email,
				Password: password,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(1).Return(db.CreateUserTxResult{}, db.ErrRecordNotFound)
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.Internal)
			},
		},
		{
			name: "BadRequestUsername",
			req: &pb.CreateUserRequest{
				Username: "",
				FullName: newUser.FullName,
				Email:    newUser.Email,
				Password: password,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(0)
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.InvalidArgument)
			},
		},
		{
			name: "BadRequestFullName",
			req: &pb.CreateUserRequest{
				Username: newUser.Username,
				FullName: "123",
				Email:    newUser.Email,
				Password: password,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(0)
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.InvalidArgument)
			},
		},
		{
			name: "BadRequestEmail",
			req: &pb.CreateUserRequest{
				Username: newUser.Username,
				FullName: newUser.FullName,
				Email:    "@example.com",
				Password: password,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(0)
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.InvalidArgument)
			},
		},
		{
			name: "BadRequestPassword",
			req: &pb.CreateUserRequest{
				Username: newUser.Username,
				FullName: newUser.FullName,
				Email:    newUser.Email,
				Password: "12345",
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(0)
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.InvalidArgument)
			},
		},
		{
			name: "AlreadyExists",
			req: &pb.CreateUserRequest{
				Username: newUser.Username,
				FullName: newUser.FullName,
				Email:    newUser.Email,
				Password: password,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(1).Return(db.CreateUserTxResult{}, db.ErrUniqueViolation)
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, st.Code(), codes.AlreadyExists)
			},
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			storeCtrl := gomock.NewController(t)
			taskCtrl := gomock.NewController(t)
			defer storeCtrl.Finish()
			defer taskCtrl.Finish()

			store := mockdb.NewMockStore(storeCtrl)
			taskDistributor := mockwk.NewMockTaskDistributor(taskCtrl)
			testCase.buildStubs(store, taskDistributor)

			server := newTestServer(t, store, taskDistributor)

			res, err := server.CreateUser(context.Background(), testCase.req)

			testCase.checkResponses(t, res, err)
		})
	}

}

func createRandomUser(t *testing.T) (db.User, string) {
	password := util.RandomString(8)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.User{
		Username:       util.RandomOwner(),
		Email:          util.RandomEmail(),
		FullName:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		Role:           util.DepositorRole,
	}

	return user, password
}
