package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/billy-le/simple-bank/db/mock"
	db "github.com/billy-le/simple-bank/db/sqlc"
	"github.com/billy-le/simple-bank/token"
	"github.com/billy-le/simple-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetUser(t *testing.T) {
	user := createRandomUser(t)

	userRow := db.User{
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
	}

	testCases := []struct {
		name           string
		username       string
		setupAuth      func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs     func(store *mockdb.MockStore)
		checkResponses func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "Ok",
			username: user.Username,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(userRow, nil)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyAndMatchUser(t, recorder.Body, userRow)
			},
		},
		{
			name:     "NotFound",
			username: user.Username,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(db.User{}, sql.ErrNoRows)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:     "InternalServerError",
			username: user.Username,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(db.User{}, sql.ErrConnDone)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			testCase.buildStubs(store)
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/users/%s", testCase.username)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			testCase.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			testCase.checkResponses(t, recorder)
		})
	}
}

func TestCreateUser(t *testing.T) {
	newUser := createRandomUser(t)

	createdUser := db.User{
		Username: newUser.Username,
		FullName: newUser.FullName,
		Email:    newUser.Email,
	}

	testCases := []struct {
		name           string
		Username       string
		Email          string
		FullName       string
		Password       string
		buildStubs     func(store *mockdb.MockStore)
		checkResponses func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "Ok",
			Username: newUser.Username,
			Email:    newUser.Email,
			FullName: newUser.FullName,
			Password: newUser.HashedPassword,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(1).Return(createdUser, nil)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyAndMatchUser(t, recorder.Body, db.User{
					Username: newUser.Username,
					FullName: newUser.FullName,
					Email:    newUser.Email,
				})
			},
		},
		{
			name:     "InternalServerError",
			Username: newUser.Username,
			Email:    newUser.Email,
			FullName: newUser.FullName,
			Password: newUser.HashedPassword,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(1).Return(db.User{}, sql.ErrConnDone)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:     "BadRequest",
			Username: "",
			Email:    newUser.Email,
			FullName: newUser.FullName,
			Password: newUser.HashedPassword,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		testCase := testCases[i]
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			testCase.buildStubs(store)
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			var jsonString = fmt.Sprintf(`{"username": "%s", "email": "%s", "full_name": "%s", "password": "%s"}`, testCase.Username, testCase.Email, testCase.FullName, testCase.Password)
			var jsonBody = []byte(jsonString)

			bodyReader := bytes.NewReader(jsonBody)

			url := "/users"
			request, err := http.NewRequest(http.MethodPost, url, bodyReader)
			require.NoError(t, err)
			server.router.ServeHTTP(recorder, request)
			testCase.checkResponses(t, recorder)
		})
	}
}

func createRandomUser(t *testing.T) db.User {
	return db.User{
		Username:       util.RandomOwner(),
		Email:          util.RandomEmail(),
		FullName:       util.RandomOwner(),
		HashedPassword: util.RandomHashedPassword(),
	}
}

func requireBodyAndMatchUser(t *testing.T, body *bytes.Buffer, user db.User) db.User {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser db.User
	err = json.Unmarshal(data, &gotUser)
	require.NoError(t, err)
	require.Equal(t, gotUser, user)
	return gotUser
}
