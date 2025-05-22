package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

type contextKey string

const (
	_initDataKey contextKey = "init-data"
)

// Returns new context with specified init data.
func withInitData(ctx context.Context, initData initdata.InitData) context.Context {
	return context.WithValue(ctx, _initDataKey, initData)
}

// Returns the init data from the specified context.
func ctxInitData(ctx context.Context) (initdata.InitData, bool) {
	initData, ok := ctx.Value(_initDataKey).(initdata.InitData)
	return initData, ok
}

// Middleware which authorizes the external client.
func authMiddleware(token string) gin.HandlerFunc {
	return func(context *gin.Context) {
		// We expect passing init data in the Authorization header in the following format:
		// <auth-type> <auth-data>
		// <auth-type> must be "tma", and <auth-data> is Telegram Mini Apps init data.
		authParts := strings.Split(context.GetHeader("authorization"), " ")
		if len(authParts) != 2 {
			context.AbortWithStatusJSON(401, map[string]any{
				"message": "Unauthorized",
			})
			return
		}

		authType := authParts[0]
		authData := authParts[1]

		switch authType {
		case "tma":
			// Validate init data. We consider init data sign valid for 1 hour from their
			// creation moment.
			if err := initdata.Validate(authData, token, time.Hour); err != nil {
				context.AbortWithStatusJSON(401, map[string]any{
					"message": err.Error(),
				})
				return
			}

			// Parse init data. We will surely need it in the future.
			initData, err := initdata.Parse(authData)
			if err != nil {
				context.AbortWithStatusJSON(500, map[string]any{
					"message": err.Error(),
				})
				return
			}

			context.Request = context.Request.WithContext(
				withInitData(context.Request.Context(), initData),
			)
		}
	}
}

// Middleware which shows the user init data.
func showInitDataMiddleware(context *gin.Context) {
	initData, ok := ctxInitData(context.Request.Context())
	if !ok {
		context.AbortWithStatusJSON(401, map[string]any{
			"message": "Init data not found",
		})
		return
	}

	context.JSON(200, initData)
}

func main() {
	// Your secret bot token.
	token := os.Getenv("TG_BOT_TOKEN")

	r := gin.New()

	r.Use(authMiddleware(token))
	r.GET("/auth", showInitDataMiddleware)

	if err := r.Run(":3000"); err != nil {
		panic(err)
	}
}
