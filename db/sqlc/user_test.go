package db

import (
	"context"
	"testing"

	"github.com/billy-le/simple-bank/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	hashedPassword, err := util.RandomHashedPassword()
	require.NoError(t, err)
	arg := CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}

	user, err := testStore.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)

	require.NotZero(t, user.CreatedAt)
	require.True(t, user.PasswordChangedAt.IsZero())
	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestUpdateUserOnlyFullName(t *testing.T) {
	oldUser := createRandomUser(t)
	newFullName := util.RandomOwner()

	user, err := testStore.UpdateUser(context.Background(), UpdateUserParams{
		Username: oldUser.Username,
		FullName: pgtype.Text{
			String: newFullName,
			Valid:  true,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, newFullName, user.FullName)
	require.Equal(t, oldUser.Email, user.Email)
	require.Equal(t, oldUser.HashedPassword, user.HashedPassword)
}

func TestUpdateUserOnlyEmail(t *testing.T) {
	oldUser := createRandomUser(t)
	email := util.RandomEmail()

	user, err := testStore.UpdateUser(context.Background(), UpdateUserParams{
		Username: oldUser.Username,
		Email: pgtype.Text{
			String: email,
			Valid:  true,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, email, user.Email)
	require.Equal(t, oldUser.FullName, user.FullName)
	require.Equal(t, oldUser.HashedPassword, user.HashedPassword)
}

func TestUpdateUserOnlyPassword(t *testing.T) {
	oldUser := createRandomUser(t)
	hashedPassword, err := util.RandomHashedPassword()
	require.NoError(t, err)

	user, err := testStore.UpdateUser(context.Background(), UpdateUserParams{
		Username: oldUser.Username,
		HashedPassword: pgtype.Text{
			String: hashedPassword,
			Valid:  true,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, hashedPassword, user.HashedPassword)
	require.Equal(t, oldUser.Email, user.Email)
	require.Equal(t, oldUser.FullName, user.FullName)
}

func TestUpdateUserAllFields(t *testing.T) {
	oldUser := createRandomUser(t)
	hashedPassword, err := util.RandomHashedPassword()
	require.NoError(t, err)
	newEmail := util.RandomEmail()
	newFullName := util.RandomOwner()

	user, err := testStore.UpdateUser(context.Background(), UpdateUserParams{
		Username: oldUser.Username,
		HashedPassword: pgtype.Text{
			String: hashedPassword,
			Valid:  true,
		},
		Email: pgtype.Text{
			String: newEmail,
			Valid:  true,
		},
		FullName: pgtype.Text{
			String: newFullName,
			Valid:  true,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, hashedPassword, user.HashedPassword)
	require.Equal(t, newEmail, user.Email)
	require.Equal(t, newFullName, user.FullName)

	require.NotEqual(t, oldUser.HashedPassword, user.HashedPassword)
	require.NotEqual(t, oldUser.Email, user.Email)
	require.NotEqual(t, oldUser.FullName, user.FullName)
}
