package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	pb "github.com/anatolio-deb/picovpnd/grpc"
	"github.com/gin-gonic/gin"
	"github.com/tonkeeper/tonapi-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// func try(context *gin.Context) {
// 	plan := Plan{
// 		ExpiresAt: time.Now().AddDate(0, 1, 0),
// 	}
// 	result := DB.Create(&plan)
// 	if result.Error != nil {
// 		context.AbortWithStatusJSON(500, map[string]any{
// 			"message": result.Error,
// 		})
// 		return
// 	}

// 	user := &User{TelegramID: initData.User.ID, PlanID: plan.ID, Plan: plan}
// 	result = DB.Create(&user)
// 	if result.Error != nil {
// 		context.AbortWithStatusJSON(500, map[string]any{
// 			"message": result.Error,
// 		})
// 		return
// 	}
// }

func userGet(context *gin.Context) {
	initData, ok := ctxInitData(context.Request.Context())
	if !ok {
		context.AbortWithStatusJSON(http.StatusUnauthorized, map[string]any{
			"message": "Init data not found",
		})
		return
	}
	user, err := UserGetByTelegramID(initData.User.ID)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusNotFound, map[string]any{
			"message": err,
		})
		return
	}
	context.IndentedJSON(http.StatusOK, user)
}

func userAdd(context *gin.Context) {
	b, err := io.ReadAll(context.Request.Body)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}
	password := Password{}
	err = json.Unmarshal(b, &password)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}
	if !password.IsValid() {
		context.AbortWithStatusJSON(http.StatusBadRequest, map[string]any{
			"message": "Password is not valid",
		})
		return
	}
	initData, ok := ctxInitData(context.Request.Context())
	if !ok {
		context.AbortWithStatusJSON(http.StatusUnauthorized, map[string]any{
			"message": "Init data not found",
		})
		return
	}
	// Check if user already exists
	user, err := UserGetByTelegramID(initData.User.ID)
	if err != nil {
		// If user does not exist, create a new one

		user = &User{
			TelegramUsername: initData.User.Username,
			TelegramID:       initData.User.ID,
		}

		result := DB.Create(&user)
		if result.Error != nil {
			context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
				"message": result.Error,
			})
			return
		}

		plan := Plan{ExpiresAt: time.Now().AddDate(0, 1, 0), UserID: user.ID, User: *user}
		result = DB.Create(&plan)
		if result.Error != nil {
			context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
				"message": result.Error,
			})
			return
		}
	}

	daemons, err := DaemonsGetAll()
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}
	if len(daemons) == 0 {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": "No daemons found",
		})
		return
	}
	// Propogate new user to ocserve server instances through the daemons
	for _, daemon := range daemons {
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(daemon.CertPEM) {
			log.Printf("could not append certificate for daemon %s: %v", daemon.Address, err)
			context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
				"message": "Failed to append certificate",
			})
			return
		}
		creds := credentials.NewClientTLSFromCert(certPool, daemon.Address)
		conn, err := grpc.NewClient(fmt.Sprintf(daemon.Address+":%d", daemon.Port), grpc.WithTransportCredentials(creds))
		if err != nil {
			log.Printf("did not connect to daemon %s: %v", daemon.Address, err)
			context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
				"message": err,
			})
			return
		}
		defer conn.Close()
		c := pb.NewOpenConnectServiceClient(conn)
		r, err := c.UserAdd(context.Request.Context(), &pb.UserAddRequest{
			Username: initData.User.Username,
			Password: password.Password,
		})
		if err != nil {
			log.Printf("could not add user: %v", err)
			context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
				"message": err,
			})
			return
		}
		if r.Error != "" {
			log.Printf("error adding user: %s", r.Error)
			context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
				"message": r.Error,
			})
			return
		}
		log.Printf("User %s added successfully on daemon %s", initData.User.Username, daemon.Address)
		context.IndentedJSON(http.StatusOK, map[string]any{
			"message":  "User added successfully",
			"initData": initData,
		})
	}

	context.IndentedJSON(http.StatusOK, map[string]any{
		"message":  "New user created",
		"initData": initData})
}

func registerDaemon(context *gin.Context) {
	b, err := io.ReadAll(context.Request.Body)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}
	daemon := Daemon{}
	err = json.Unmarshal(b, &daemon)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}
	daemonRec, err := DaemonGetByAddress(daemon.Address)
	if err != nil {
		result := DB.Create(&daemon)
		if result.Error != nil {
			context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
				"message": result.Error,
			})
			return
		}
	} else {
		daemonRec.Port = daemon.Port
		daemonRec.CertPEM = daemon.CertPEM
		result := DB.Save(&daemonRec)
		if result.Error != nil {
			context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
				"message": result.Error,
			})
			return
		}
	}
	context.IndentedJSON(http.StatusOK, daemon)
}

func plansGet(context *gin.Context) {
	initData, ok := ctxInitData(context.Request.Context())
	if !ok {
		context.AbortWithStatusJSON(http.StatusUnauthorized, map[string]any{
			"message": "Init data not found",
		})
		return
	}
	plan, err := PlansGetByTelegramUserID(initData.User.ID)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusNotFound, map[string]any{
			"message": err,
		})
		return
	}
	context.IndentedJSON(http.StatusOK, plan)

}

func daemonsGet(context *gin.Context) {
	daemons, err := DaemonsGetAll()
	if err != nil {
		context.AbortWithStatusJSON(http.StatusNotFound, map[string]any{
			"message": err,
		})
		return
	}
	public := make([]DaemonPublic, len(daemons))
	for i, d := range daemons {
		public[i] = DaemonPublic{
			Address: d.Address,
		}
	}
	context.IndentedJSON(http.StatusOK, public)
}

func planUpdate(context *gin.Context) {
	b, err := io.ReadAll(context.Request.Body)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}
	request := PlanUpdate{}
	err = json.Unmarshal(b, &request)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}

	log.Println("Transaction BOC:", request.BOC)

	// find transaction by boc

	client, err := tonapi.NewClient(tonapi.TonApiURL, tonapi.WithToken(os.Getenv("TON_API_KEY")))
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}

	response, err := client.GetBlockchainAccountTransactions(context, tonapi.GetBlockchainAccountTransactionsParams{
		AccountID: os.Getenv("TON_ACCOUNT_ID"),
		Limit:     tonapi.NewOptInt32(100),
	})

	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": err,
		})
		return
	}

	var transaction *tonapi.Transaction
	transactions := response.GetTransactions()
	for i, t := range transactions {
		message := t.GetInMsg()
		if message.IsSet() && message.Value.GetRawBody().Value == request.Comment {
			transaction = &transactions[i]
			break

			// TODO: get transaction status
			// 		for{
			// 			client.GetBlockchainTransaction(context, tonapi.GetBlockchainTransactionParams{
			// 	TransactionID: t. ,
			// })
			// 		}
		}
	}

	if transaction == nil {
		context.AbortWithStatusJSON(http.StatusNotFound, map[string]any{
			"message": "transaction not found",
		})
		return
	}

	initData, ok := ctxInitData(context.Request.Context())
	if !ok {
		context.AbortWithStatusJSON(http.StatusUnauthorized, map[string]any{
			"message": "Init data not found",
		})
		return
	}
	plan, err := PlansGetByTelegramUserID(initData.User.ID)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusNotFound, map[string]any{
			"message": err,
		})
		return
	}

	// unlock user if locked
	if time.Until(plan.ExpiresAt) <= 0 {
		daemons, err := DaemonsGetAll()
		if err != nil {
			log.Println(err)
		}

		for _, daemon := range daemons {
			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(daemon.CertPEM) {
				log.Println(err)
				continue
			}
			creds := credentials.NewClientTLSFromCert(certPool, daemon.Address)
			conn, err := grpc.NewClient(fmt.Sprintf(daemon.Address+":%d", daemon.Port), grpc.WithTransportCredentials(creds))
			if err != nil {
				log.Println(err)
				continue
			}
			defer conn.Close()
			c := pb.NewOpenConnectServiceClient(conn)
			resp, err := c.UserUnlock(context, &pb.UserUnlockRequest{
				Username: plan.User.TelegramUsername,
			})
			if err != nil {
				log.Println(err)
				continue
			}
			if resp.Error != "" {
				log.Println(err)
				continue
			}
		}
	}

	plan.ExpiresAt = time.Now().AddDate(0, request.Months, 0)
	result := DB.Save(plan)
	if result.Error != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
			"message": result.Error,
		})
		return
	}
	context.IndentedJSON(http.StatusOK, plan)
}
