package main

import (
	"github.com/tonkeeper/tonapi-go"
)

func getTransactionByComment(comment string) (*tonapi.Transaction, error) {
	// Create a new client with default settings
	// If you want to use testnet, use tonapi.TestnetTonApiURL
	// You can use TonAPI.io without a token by passing &tonapi.Security{} as the second parameter,
	// but note that TonAPI.io has strict rate limits, so it's better to get a Token from tonconsole.com
	// in the TonAPI section - it's completely free

	// Get account information
	// account, err := client.GetAccount(context.Background(), tonapi.GetAccountParams{
	// 	AccountID: os.Getenv("TON_ACCOUNT_ID"),
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Printf("Account Balance: %v\n", account.Balance)

	return nil, nil
}
