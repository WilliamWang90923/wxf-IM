package token

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParseJwtToken(t *testing.T) {
	tk1 := &Token{
		Account: "test1",
		App:     "wxf",
		Exp:     time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	secret := "123456"

	tokenString, err := Generate(secret, tk1)
	assert.Nil(t, err)
	t.Log(tokenString)

	tk2, err := Parse(secret, tokenString)
	assert.Nil(t, err)
	assert.Equal(t, "test1", tk2.Account)
}
