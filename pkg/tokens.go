package pkg

import (
	"fmt"

	"github.com/gofrs/uuid"
)

func GetToken() string {
	token, err := uuid.NewV4()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return token.String()
}
