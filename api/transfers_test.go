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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateTransfer(t *testing.T) {
	currency := util.USD
	amount := int64(10)
	user1 := createRandomUser(t)
	user2 := createRandomUser(t)
	account1 := createRandomAccount(user1.Username)
	account2 := createRandomAccount(user2.Username)

	testCases := []struct {
		name           string
		fromAccountID  int64
		toAccountID    int64
		amount         int64
		currency       string
		setupAuth      func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs     func(store *mockdb.MockStore)
		checkResponses func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:          "Ok",
			fromAccountID: account1.ID,
			toAccountID:   account2.ID,
			amount:        amount,
			currency:      currency,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user1.Username, time.Minute)

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(1).Return(account2, nil)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				})).Times(1).Return(db.TransferTxResult{
					Transfer: db.Transfer{
						FromAccountID: account1.ID,
						ToAccountID:   account2.ID,
						Amount:        amount,
					},
					FromAccount: db.Account{
						ID:       account1.ID,
						Owner:    account1.Owner,
						Balance:  account1.Balance - amount,
						Currency: currency,
					},
					ToAccount: db.Account{
						ID:       account2.ID,
						Owner:    account2.Owner,
						Balance:  account2.Balance + amount,
						Currency: currency,
					},
					FromEntry: db.Entry{
						AccountID: account1.ID,
						Amount:    amount,
					},
					ToEntry: db.Entry{
						AccountID: account2.ID,
						Amount:    amount,
					},
				}, nil)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:          "InvalidCurrency",
			fromAccountID: account1.ID,
			toAccountID:   account2.ID,
			amount:        amount,
			currency:      util.EUR,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user1.Username, time.Minute)

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				})).Times(0)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var errRes gin.H
				err = json.Unmarshal(data, &errRes)
				require.NoError(t, err)

				var errString = fmt.Sprintf("account [%d] currency mismatch: [%s] vs [EUR]", account1.ID, account1.Currency)
				require.Equal(t, errRes["error"], errString)
			},
		},
		{
			name:          "AccountNotFound",
			fromAccountID: account1.ID,
			toAccountID:   account2.ID,
			amount:        amount,
			currency:      currency,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user1.Username, time.Minute)

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				})).Times(0)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var errRes gin.H
				err = json.Unmarshal(data, &errRes)
				require.NoError(t, err)

				require.Equal(t, errRes["error"], "sql: no rows in result set")
			},
		},
		{
			name:          "Account1BadRequest",
			fromAccountID: 0,
			toAccountID:   account2.ID,
			amount:        amount,
			currency:      currency,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user1.Username, time.Minute)

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				})).Times(0)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var errRes gin.H
				err = json.Unmarshal(data, &errRes)
				require.NoError(t, err)

				require.Equal(t, errRes["error"], "Key: 'createTransferRequest.FromAccountID' Error:Field validation for 'FromAccountID' failed on the 'required' tag")
			},
		},
		{
			name:          "Account2BadRequest",
			fromAccountID: account1.ID,
			toAccountID:   0,
			amount:        amount,
			currency:      currency,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user1.Username, time.Minute)

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil).MaxTimes(1)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				})).Times(0)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var errRes gin.H
				err = json.Unmarshal(data, &errRes)
				require.NoError(t, err)

				require.Equal(t, errRes["error"], "Key: 'createTransferRequest.ToAccountID' Error:Field validation for 'ToAccountID' failed on the 'required' tag")
			},
		},
		{
			name:          "AccountInternalServerError",
			fromAccountID: account1.ID,
			toAccountID:   account2.ID,
			amount:        amount,
			currency:      currency,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user1.Username, time.Minute)

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				})).Times(0)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var errRes gin.H
				err = json.Unmarshal(data, &errRes)
				require.NoError(t, err)

				require.Equal(t, errRes["error"], "sql: connection is already closed")
			},
		},
		{
			name:          "TransferInternalServerError",
			fromAccountID: account1.ID,
			toAccountID:   account2.ID,
			amount:        amount,
			currency:      currency,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user1.Username, time.Minute)

			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Times(1).Return(account2, nil)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        amount,
				})).Times(1).Return(db.TransferTxResult{}, sql.ErrConnDone)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var errRes gin.H
				err = json.Unmarshal(data, &errRes)
				require.NoError(t, err)

				require.Equal(t, errRes["error"], "sql: connection is already closed")
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

			var jsonString = fmt.Sprintf(`{"from_account_id": %d, "to_account_id": %d, "amount": %d, "currency": "%s"}`, testCase.fromAccountID, testCase.toAccountID, testCase.amount, testCase.currency)
			var jsonBody = []byte(jsonString)

			bodyReader := bytes.NewReader(jsonBody)

			url := "/transfers"
			request, err := http.NewRequest(http.MethodPost, url, bodyReader)
			require.NoError(t, err)

			testCase.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			testCase.checkResponses(t, recorder)
		})
	}
}
