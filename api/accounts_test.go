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

	mockdb "github.com/billy-le/simple-bank/db/mock"
	db "github.com/billy-le/simple-bank/db/sqlc"
	"github.com/billy-le/simple-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetAccount(t *testing.T) {
	account := createRandomAccount()

	testCases := []struct {
		name           string
		accountID      int64
		buildStubs     func(store *mockdb.MockStore)
		checkResponses func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "Ok",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyAndMatch(t, recorder.Body, account)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalServerError",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "InvalidID",
			accountID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)

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
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", testCase.accountID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)
			server.router.ServeHTTP(recorder, request)
			testCase.checkResponses(t, recorder)
		})
	}
}

func TestListAccounts(t *testing.T) {
	accounts := []db.Account{}
	for i := 0; i < 20; i++ {
		account := createRandomAccount()
		accounts = append(accounts, account)
	}

	testCases := []struct {
		name           string
		pageID         int32
		pageSize       int32
		buildStubs     func(store *mockdb.MockStore)
		checkResponses func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "Ok",
			pageID:   1,
			pageSize: 5,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(1).Return(accounts[0:5], nil)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)

				var gotAccounts []db.Account
				err = json.Unmarshal(data, &gotAccounts)
				require.NoError(t, err)
				require.Len(t, gotAccounts, 5)
				for i, acc := range gotAccounts {
					require.Equal(t, accounts[i], acc)
				}
			},
		},
		{
			name:     "BadRequest",
			pageID:   1,
			pageSize: 11,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:     "InternalServerError",
			pageID:   1,
			pageSize: 5,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(1).Return([]db.Account{}, sql.ErrConnDone)
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
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts?page_id=%d&page_size=%d", testCase.pageID, testCase.pageSize)

			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)
			server.router.ServeHTTP(recorder, request)
			testCase.checkResponses(t, recorder)
		})
	}
}

func TestCreateAccount(t *testing.T) {
	newAccount := db.Account{
		ID:       util.RandomInt(0, 1000),
		Owner:    util.RandomOwner(),
		Currency: util.RandomCurrency(),
		Balance:  0,
	}

	testCases := []struct {
		name           string
		Owner          string
		Currency       string
		buildStubs     func(store *mockdb.MockStore)
		checkResponses func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "Ok",
			Owner:    newAccount.Owner,
			Currency: newAccount.Currency,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(1).Return(newAccount, nil)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyAndMatch(t, recorder.Body, newAccount)
			},
		},
		{
			name:     "InternalServerError",
			Owner:    newAccount.Owner,
			Currency: newAccount.Currency,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(1).Return(db.Account{}, sql.ErrConnDone)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:     "BadRequest",
			Owner:    newAccount.Owner,
			Currency: "CAD",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)

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
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			var jsonString = fmt.Sprintf(`{"owner": "%s", "currency": "%s"}`, testCase.Owner, testCase.Currency)
			var jsonBody = []byte(jsonString)

			bodyReader := bytes.NewReader(jsonBody)

			url := "/accounts"
			request, err := http.NewRequest(http.MethodPost, url, bodyReader)
			require.NoError(t, err)
			server.router.ServeHTTP(recorder, request)
			testCase.checkResponses(t, recorder)
		})
	}
}

func TestUpdateAccount(t *testing.T) {
	account := createRandomAccount()
	amount := util.RandomMoney()

	updatedAccount := db.Account{
		ID:       account.ID,
		Owner:    account.Owner,
		Currency: account.Currency,
		Balance:  account.Balance + amount,
	}

	testCases := []struct {
		name           string
		accountID      int64
		amount         int64
		buildStubs     func(store *mockdb.MockStore)
		checkResponses func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "Ok",
			accountID: account.ID,
			amount:    amount,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().AddAccountBalance(gomock.Any(), gomock.Eq(db.AddAccountBalanceParams{ID: account.ID, Amount: amount})).Times(1).Return(updatedAccount, nil)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyAndMatch(t, recorder.Body, updatedAccount)
			},
		},
		{
			name:      "InvalidID",
			accountID: -1,
			amount:    amount,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().AddAccountBalance(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "InvalidAmount",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().AddAccountBalance(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			amount:    amount,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().AddAccountBalance(gomock.Any(), gomock.Eq(db.AddAccountBalanceParams{
					ID:     account.ID,
					Amount: amount,
				})).Times(1).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalServerError",
			accountID: account.ID,
			amount:    amount,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().AddAccountBalance(gomock.Any(), gomock.Eq(db.AddAccountBalanceParams{
					ID:     account.ID,
					Amount: amount,
				})).Times(1).Return(db.Account{}, sql.ErrConnDone)
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
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			jsonString := ""
			if testCase.amount != 0 {
				jsonString = fmt.Sprintf(`{"amount": %d}`, testCase.amount)
			}
			fmt.Println(jsonString)
			var jsonBody = []byte(jsonString)
			bodyReader := bytes.NewReader(jsonBody)
			url := fmt.Sprintf("/accounts/%d", testCase.accountID)

			request, err := http.NewRequest(http.MethodPut, url, bodyReader)
			require.NoError(t, err)
			server.router.ServeHTTP(recorder, request)
			testCase.checkResponses(t, recorder)
		})
	}
}

func TestDeleteAccount(t *testing.T) {
	account := db.Account{
		ID:       util.RandomInt(0, 1000),
		Owner:    "User",
		Balance:  0,
		Currency: "USD",
	}

	testCases := []struct {
		name           string
		accountID      int64
		buildStubs     func(store *mockdb.MockStore)
		checkResponses func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "Ok",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name:      "BadRequest",
			accountID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), gomock.Any()).Times(0)

			},
			checkResponses: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "LogError",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(sql.ErrConnDone)
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
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", testCase.accountID)

			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)
			server.router.ServeHTTP(recorder, request)
			testCase.checkResponses(t, recorder)
		})
	}
}

func createRandomAccount() db.Account {
	return db.Account{
		ID:       util.RandomInt(0, 1000),
		Owner:    util.RandomOwner(),
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}
}

func requireBodyAndMatch(t *testing.T, body *bytes.Buffer, account db.Account) db.Account {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAccount db.Account
	err = json.Unmarshal(data, &gotAccount)
	require.NoError(t, err)
	require.Equal(t, gotAccount, account)
	return gotAccount
}
