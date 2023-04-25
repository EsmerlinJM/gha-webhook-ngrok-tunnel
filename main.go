package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v51/github"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

var (
	githubToken = os.Getenv("GITHUB_TOKEN")
	repo = os.Getenv("REPO")
	owner = os.Getenv("OWNER")
	webHookEndpoint = os.Getenv("WEBHOOK_ENDPOINT")
)

func main() {
	// if _, err := os.Stat("action.yml"); err == nil {
	// 	githubToken = os.Getenv("INPUT_GITHUB_PAT_TOKEN")
	// 	repo = os.Getenv("INPUT_REPO")
	// 	owner = os.Getenv("INPUT_OWNER")
	// 	webHookEndpoint = os.Getenv("INPUT_WEBHOOK_ENDPOINT")
	// }

    if err := run(context.Background()); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    ngrokHooks, err := fetchNgrokHooks(ctx)

	if err != nil {
		return err
	}

	if len(ngrokHooks) > 0 {
		for _, hook := range ngrokHooks {
			url := hook.Config["url"].(string)

			if strings.Contains(url, "ngrok") {
				deleteNgrokHook(ctx, hook.ID)
				break
			}
		}
	}

	tunnel, err := makeNgrokTunnel(ctx)

	if err != nil {
		return err
	}

	tunnelUrl := tunnel.URL()+webHookEndpoint

	output := fmt.Sprintf("::set-output name=webhook_url::%s", tunnelUrl)

	fmt.Println(output)

	config := map[string]interface{} {
		"url": flag.String("url", tunnelUrl, "The URL to which the payloads will be delivered."),
	}

	webHook := &github.Hook{
		Name:         flag.String("Name", "ngrok tunnel", "Name of webhook"),
		Events:       []string{"push", "pull_request"},
		Active:       flag.Bool("Active", true, "Tunnel is active"),
		Config: config,
	}

	fmt.Println("Creating hook...")

	time.Sleep(3 * time.Second)
	
	hookCreated := createNgrokHook(ctx, webHook)

	log.Println("Hook created: ", *hookCreated.URL)

	for {
		conn, err := tunnel.Accept()
		if err != nil {
			return err
		}

		log.Println("accepted connection from", conn.RemoteAddr())

		go func() {
			des := os.Getenv("PRIVATE_ADDRESS")
			err := handleConnection(ctx, des, conn)
			log.Println("connection closed:", err)
		}()
	}
}

func fetchNgrokHooks(ctx context.Context) ([]*github.Hook, error) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	repo := os.Getenv("REPO")
	owner := os.Getenv("OWNER")

    tokenService := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
    tc := oauth2.NewClient(ctx, tokenService)
    client := github.NewClient(tc)

    hooks, _, err := client.Repositories.ListHooks(ctx, owner, repo, nil)

	if err != nil {
		fmt.Printf("Problem in getting repository information %v\n", err)
	}
	
    if _, ok := err.(*github.AbuseRateLimitError); ok {
        log.Println("hit rate limit")
    }


	return hooks, err
}

func createNgrokHook(ctx context.Context, hook *github.Hook) (*github.Hook) {
    tokenService := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
    tc := oauth2.NewClient(ctx, tokenService)
    client := github.NewClient(tc)

    hook, _, err := client.Repositories.CreateHook(ctx, owner, repo, hook)

	if err != nil {
		fmt.Printf("Problem in getting repository information %v\n", err)
		os.Exit(1)
	}

	return hook
}

func deleteNgrokHook(ctx context.Context, hookId *int64) {
    tokenService := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
    tc := oauth2.NewClient(ctx, tokenService)
    client := github.NewClient(tc)

    hook, err := client.Repositories.DeleteHook(ctx, owner, repo, *hookId)

	if err != nil {
		fmt.Printf("Problem in getting repository information %v\n", err)
		os.Exit(1)
	}

	log.Println("Hook deleted:", hook.Status)
}

func makeNgrokTunnel(ctx context.Context) (ngrok.Tunnel, error) {
	tun, err := ngrok.Listen(ctx,
        config.HTTPEndpoint(),
        ngrok.WithAuthtokenFromEnv(),
    )

    return tun, err
}

func handleConnection(ctx context.Context, dest string, conn net.Conn) error {
	next, err := net.Dial("tcp", dest)
	if err != nil {
		return err
	}

	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		_, err := io.Copy(next, conn)
		return err
	})
	g.Go(func() error {
		_, err := io.Copy(conn, next)
		return err
	})

	return g.Wait()
}

