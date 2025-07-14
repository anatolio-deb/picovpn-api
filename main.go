package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	pb "github.com/anatolio-deb/picovpnd/grpc"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var TELEGRAM_BOT_TOKEN string = ""

func main() {
	// Your secret bot token.
	TELEGRAM_BOT_TOKEN = os.Getenv("TELEGRAM_BOT_TOKEN")
	// Set gin mode to release.
	r := gin.New()

	gin.SetMode(gin.ReleaseMode)
	// Set gin logger.

	r.Use(
		authMiddleware(TELEGRAM_BOT_TOKEN),
		cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST"},
			AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			AllowOriginFunc: func(origin string) bool {
				return strings.Contains(origin, "picovpn.ru")
			},
			MaxAge: 24 * time.Hour,
		}),
		gin.Recovery(),
		gin.LoggerWithConfig(gin.LoggerConfig{
			Formatter: func(param gin.LogFormatterParams) string {
				return "[" + param.TimeStamp.Format(time.RFC3339) + "] " +
					param.Method + " " +
					param.Path + " " +
					param.ClientIP + " " +
					param.ErrorMessage + " " +
					param.Latency.String() + "\n"
			},
			Output: os.Stdout,
		}),
	)
	r.POST("/api/auth", showInitDataMiddleware)
	r.GET("/api/users/", userGet)
	r.POST("/api/users", userAdd)
	r.POST("/api/daemons", registerDaemon)
	r.GET("/api/plans/", plansGet)
	r.GET("/api/daemons", daemonsGet)
	r.POST("/api/plans", planUpdate)

	go LockExpiredUsers()

	// Run the server on port 8080 with TLS.
	// Make sure to replace the paths to your SSL certificate and key files.
	// You can use Let's Encrypt or any other certificate authority.
	// Make sure to have the certificate and key files in the specified paths.
	// If you are using Let's Encrypt, you can use the following command to generate the certificate and key files:
	// sudo certbot certonly --standalone -d picovpn.ru -d www.picovpn.ru
	// Make sure to have the certificate and key files in the specified paths.
	// If you are using a self-signed certificate, you can use the following command to generate the certificate and key files:
	// openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes
	// err := r.RunTLS(":8080", "/etc/letsencrypt/live/picovpn.ru/fullchain.pem", "/etc/letsencrypt/live/picovpn.ru/privkey.pem")
	err := r.Run(":8000")
	if err != nil {
		panic(err)
	}
}

func LockExpiredUsers() {
	ticker := time.NewTicker(time.Minute)
	done := make(chan bool)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				h, m, _ := time.Now().Clock()
				if m == 0 && (h == 9 || h == 15) {
					daemons, err := DaemonsGetAll()
					if err != nil {
						log.Println(err)
					}

					plans, err := PlansGetExpired()
					if err != nil {
						log.Println(err)
					}
					for _, p := range plans {
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
							resp, err := c.UserLock(ctx, &pb.UserLockRequest{
								Username: p.User.TelegramUsername,
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
				}
			}
		}
	}()

	<-ctx.Done()
	stop()
	done <- true
}
