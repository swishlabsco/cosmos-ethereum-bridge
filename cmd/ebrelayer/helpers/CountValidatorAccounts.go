package helpers

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
)

// Counts the number of validator accounts
func CountValidatorAccounts(arr []keys.Info, validatorPrefix string) int {
	count := 0

	for _, info := range arr {
		if strings.HasPrefix(info.GetName(), validatorPrefix) {
			count++
		}
	}

	return count
}