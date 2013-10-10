package main

import (
	"code.google.com/p/goauth2/oauth"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	port    = flag.String("p", "8000", "Port number (default 8000)")
	orgName = flag.String("org", "", "GitHub organization name")
)

func getPullsForRepo(wg *sync.WaitGroup, c *github.Client, repoName string, repoMap map[string][]github.PullRequest) {
	defer wg.Done()
	opt := &github.PullRequestListOptions{"open", "", ""}
	pulls, _, err := c.PullRequests.List(*orgName, repoName, opt)
	if err != nil {
		fmt.Println(err)
	}
	repoMap[repoName] = pulls
}

func getPulls() map[string][]github.PullRequest {
	var wg sync.WaitGroup
	repoMap := map[string][]github.PullRequest{}
	githubToken := os.Getenv("GITHUB_API_TOKEN")
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: githubToken},
	}
	client := github.NewClient(t.Client())
	repos, _, err := client.Repositories.ListByOrg(*orgName, nil)
	if err != nil {
		fmt.Println(err)
	}
	for _, repo := range repos {
		wg.Add(1)
		go getPullsForRepo(&wg, client, *repo.Name, repoMap)
	}
	wg.Wait()
	return repoMap
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	repoMap := getPulls()
	t, err := template.New("index.html").ParseFiles("templates/index.html")
	if err != nil {
		log.Panic(err)
	}
	// Render the template
	err = t.Execute(w, map[string]interface{}{"RepoMap": repoMap})
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	flag.Parse()
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	fmt.Println("Running on localhost:" + *port)
	log.Fatal(http.ListenAndServe(":"+*port, r))
}
